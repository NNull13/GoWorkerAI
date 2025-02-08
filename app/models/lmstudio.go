package models

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"sync"
	"time"

	"GoWorkerAI/app/storage"
	"GoWorkerAI/app/tools"
	"GoWorkerAI/app/utils"
	"GoWorkerAI/app/utils/restclient"
)

const (
	endpoint          = "/v1/chat/completions"
	embeddingEndpoint = "/v1/embeddings"
	model             = "qwen2.5-7b-instruct-1m"
	embeddingModel    = "text-embedding-nomic-embed-text-v1.5-embedding"
)

type LMStudioClient struct {
	restClient *restclient.RestClient
	storage    storage.Interface
	cache      sync.Map
}

func NewLMStudioClient(db storage.Interface) *LMStudioClient {
	return &LMStudioClient{
		restClient: restclient.NewRestClient("http://localhost:1234", nil),
		storage:    db,
	}
}

func (mc *LMStudioClient) Think(ctx context.Context, messages []Message, temp float64, maxTokens int) (string, error) {
	response, err := mc.generateResponse(ctx, messages, nil, temp, maxTokens)
	if err != nil {
		return "", err
	}
	return response.Choices[0].Message.Content, nil
}

func (mc *LMStudioClient) YesOrNo(ctx context.Context, messages []Message) (bool, error) {
	response, err := mc.generateResponse(ctx, messages, nil, 0.0, 1)
	if err != nil {
		return false, err
	}
	lowerResp := strings.ToLower(strings.TrimSpace(response.Choices[0].Message.Content))
	if lowerResp != "true" && lowerResp != "false" {
		return false, fmt.Errorf("unexpected response: %v", response)
	}
	return lowerResp == "true", nil
}

func (mc *LMStudioClient) GenerateSummary(ctx context.Context, taskID string) (string, error) {
	history, err := mc.storage.GetHistoryByTaskID(ctx, taskID)
	if err != nil {
		return "", err
	}
	if len(history) == 0 {
		return "Task not started yet, incomplete", nil
	}

	systemPrompt := `You are an AI responsible for summarizing task execution histories. 
	Generate a structured summary that includes:
	- A high-level overview of the task.
	- The key actions performed in sequence.
	- Any errors or issues encountered.
	- The tools used and their results.
	- The current state of progress and what remains to be done.
	Ensure that the summary is concise, coherent, and useful for future iterations.`

	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: "Here is the history of task execution. Summarize it:"},
	}

	for _, entry := range history {
		messages = append(messages, Message{
			Role:    entry.Role,
			Content: fmt.Sprintf("Role: %s | Tool: %s | Content: %s", entry.Role, entry.Tool, entry.Content),
		})
	}

	response, err := mc.generateResponse(ctx, messages, nil, 0.1, 500)
	if err != nil {
		return "", err
	}
	if len(response.Choices) == 0 {
		return "", errors.New("empty LLM response for summary")
	}
	return response.Choices[0].Message.Content, nil
}

func (mc *LMStudioClient) Process(ctx context.Context, messages []Message, toolkit map[string]tools.Tool, taskID string, maxIterations int) (string, error) {
	response, err := mc.generateResponse(ctx, messages, toolkit, 0.2, -1)
	if err != nil {
		return "", err
	}

	message := response.Choices[0].Message
	for i := 0; ; i++ {
		messages = append(messages, mc.handleToolCalls(ctx, toolkit, message.ToolCalls, taskID)...)
		if response, err = mc.generateResponse(ctx, messages, toolkit, 0.2, -1); err != nil {
			return "", err
		}
		message = response.Choices[0].Message
		if len(message.ToolCalls) == 0 {
			break
		}
	}

	mc.storage.SaveHistory(ctx, storage.Record{
		TaskID:    taskID,
		Role:      "assistant",
		Content:   message.Content,
		CreatedAt: time.Now(),
	})

	return message.Content, nil
}

func (mc *LMStudioClient) handleToolCalls(ctx context.Context, toolkit map[string]tools.Tool,
	toolCalls []toolCall, taskID string) (messages []Message) {
	for _, call := range toolCalls {
		toolTask := tools.ToolTask{Key: call.Function.Name}
		toolTask.Parameters, _ = utils.ParseArguments(call.Function.Arguments)
		tool, exists := toolkit[toolTask.Key]
		if !exists || tool.HandlerFunc == nil {
			log.Printf("⚠️ Tool not found or missing handler: %s", toolTask.Key)
			continue
		}

		result, err := tool.HandlerFunc(toolTask)
		if err != nil {
			log.Printf("⚠️ Tool %s execution failed: %v", tool.Name, err)
			result = err.Error()
		}

		mc.storage.SaveHistory(ctx, storage.Record{
			TaskID:     taskID,
			Role:       "tool",
			Tool:       tool.Name,
			Content:    result,
			Parameters: call.Function.Arguments,
			CreatedAt:  time.Now(),
		})

		messages = append(messages,
			Message{
				Role:    "tool",
				Content: result,
			},
		)
	}

	return messages
}

func (mc *LMStudioClient) enrichWithEmbeddings(ctx context.Context, messages []Message) ([]Message, error) {
	var enhancedMessages []Message
	for _, msg := range messages {
		embedding, err := mc.getEmbeddings(ctx, msg.Content)
		if err != nil {
			log.Printf("⚠️ Error obtaining embedding for message: %s", msg.Content)
			enhancedMessages = append(enhancedMessages, msg)
			return nil, err
		}

		enhancedMessages = append(enhancedMessages, Message{
			Role:    msg.Role,
			Content: msg.Content + "\n\nEmbedding hash: " + hashEmbedding(embedding),
		})
	}

	return enhancedMessages, nil
}

func (mc *LMStudioClient) getEmbeddings(ctx context.Context, input string) ([]float64, error) {
	if cached, ok := mc.cache.Load(input); ok {
		if emb, ok := cached.([]float64); ok {
			return emb, nil
		}
	}

	payload := embeddingRequestPayload{
		Model: embeddingModel,
		Input: input,
	}

	response, status, err := mc.restClient.Post(ctx, embeddingEndpoint, payload, nil)
	if err != nil || status != 200 {
		return nil, fmt.Errorf("embedding request failed: HTTP %d, error: %w", status, err)
	}

	var embeddingResp embeddingResponse
	if err = json.Unmarshal(response, &embeddingResp); err != nil {
		return nil, fmt.Errorf("error parsing embedding response: %w", err)
	}

	if len(embeddingResp.Data) == 0 {
		return nil, errors.New("no embedding data received")
	}

	mc.cache.Store(input, embeddingResp.Data[0].Embedding)
	return embeddingResp.Data[0].Embedding, nil
}

func hashEmbedding(embedding []float64) string {
	hash := sha256.New()
	for _, value := range embedding {
		hash.Write([]byte(fmt.Sprintf("%.6f", value)))
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func (mc *LMStudioClient) generateResponse(ctx context.Context, messages []Message, tools map[string]tools.Tool, temp float64, maxTokens int) (*ResponseLLM, error) {
	enrichedMessages, err := mc.enrichWithEmbeddings(ctx, messages)
	if err != nil {
		log.Printf("⚠️ Error enriching messages with embeddings: %v", err)
	} else {
		messages = enrichedMessages
	}

	payload := requestPayload{
		Model:       model,
		Tools:       functionsToPayload(tools),
		Messages:    messages,
		Temperature: temp,
		MaxTokens:   maxTokens,
	}

	return mc.sendRequestAndParse(ctx, payload, 3)
}

func functionsToPayload(functions map[string]tools.Tool) (payload []functionPayload) {
	for _, function := range functions {
		payload = append(payload, functionPayload{
			Type:     "function",
			Function: function,
		})
	}
	return payload
}

func (mc *LMStudioClient) sendRequestAndParse(ctx context.Context, payload requestPayload, maxRetries int) (*ResponseLLM, error) {
	var err error
	var response []byte
	var status int
	var generatedResponse ResponseLLM

	for i := 0; i < maxRetries; i++ {
		select {
		case <-ctx.Done():
			log.Println("🚨 Request canceled before execution")
			return nil, ctx.Err()
		default:
			time.Sleep(time.Duration(math.Pow(2, float64(i))) * 100 * time.Millisecond)

			response, status, err = mc.restClient.Post(ctx, endpoint, payload, nil)
			if err != nil || status != 200 {
				log.Printf("⚠️ Attempt %d failed: HTTP %d | Response: %s | Error: %v | Payload %v",
					i+1, string(response), status, err, payload)
				continue
			}

			if err = json.Unmarshal(response, &generatedResponse); err != nil {
				log.Printf("⚠️ Error parsing response: %v", err)
				continue
			}

			return &generatedResponse, nil
		}
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", maxRetries, err)
}
