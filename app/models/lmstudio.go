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

	"GoWorkerAI/app/restclient"
	"GoWorkerAI/app/storage"
	"GoWorkerAI/app/tools"
	"GoWorkerAI/app/utils"
)

const (
	endpoint          = "/v1/chat/completions"
	embeddingEndpoint = "/v1/embeddings"
	model             = "qwen2.5-7b-instruct-1m"
	embeddingModel    = "text-embedding-nomic-embed-text-v1.5-embedding"
)

type requestPayload struct {
	Model       string            `json:"model"`
	Messages    []Message         `json:"messages"`
	Temperature float64           `json:"temperature"`
	MaxTokens   int               `json:"max_tokens"`
	Tools       []FunctionPayload `json:"tools"`
}

type FunctionPayload struct {
	Type     string     `json:"type"`
	Function tools.Tool `json:"function"`
}

type embeddingRequestPayload struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type embeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ResponseLLM struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int     `json:"index"`
		Logprobs     *string `json:"logprobs,omitempty"`
		FinishReason string  `json:"finish_reason"`
		Message      Message `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

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

func (mc *LMStudioClient) generateResponse(ctx context.Context, messages []Message, tools map[string]tools.Tool, temp float64, maxTokens int) (*ResponseLLM, error) {
	enhancedMessages, err := mc.enrichWithEmbeddings(ctx, messages)
	if err != nil {
		return nil, err
	}
	payload := requestPayload{
		Model:       model,
		Tools:       functionsToPayload(tools),
		Messages:    enhancedMessages,
		Temperature: temp,
		MaxTokens:   maxTokens,
	}
	return mc.sendRequestAndParse(ctx, payload, 3)
}

func (mc *LMStudioClient) Process(ctx context.Context, messages []Message, toolkit map[string]tools.Tool,
	taskID string, maxIterations int) (string, error) {
	response, err := mc.doProcess(ctx, messages, toolkit, taskID, maxIterations)
	if err != nil {
		return "", err
	}

	mc.storage.SaveIteration(ctx, storage.Iteration{
		TaskID:    taskID,
		Role:      "assistant",
		Content:   response,
		CreatedAt: time.Now(),
	})

	return response, nil
}

func (mc *LMStudioClient) doProcess(ctx context.Context, messages []Message, toolkit map[string]tools.Tool, taskID string, maxIterations int) (string, error) {
	response, err := mc.generateResponse(ctx, messages, toolkit, 0.2, -1)
	if err != nil {
		return "", err
	}

	for i := 0; i < maxIterations; i++ {
		message := response.Choices[0].Message

		if len(message.ToolCalls) == 0 {
			return message.Content, nil
		}

		toolMessages := []Message{}
		for _, call := range message.ToolCalls {
			toolTask := tools.ToolTask{Key: call.Function.Name}
			toolTask.Parameters, _ = utils.ParseArguments(call.Function.Arguments)
			tool := toolkit[toolTask.Key]
			if tool.HandlerFunc == nil {
				return "", fmt.Errorf("handler function is nil for tool: %s", toolTask.Key)
			}

			var result string
			if result, err = tool.HandlerFunc(toolTask); err != nil {
				continue
			}

			log.Printf("✅ Message tool %s parameters: %s result: %s", tool.Name, toolTask.Parameters, result)
			mc.storage.SaveIteration(ctx, storage.Iteration{
				TaskID:    taskID,
				Role:      "tool",
				Tool:      toolkit[toolTask.Key].Name,
				Content:   result,
				CreatedAt: time.Now(),
			})

			toolMessages = append(toolMessages, Message{
				Role:    "tool",
				Content: result,
			})
			assistantToolCallMessage := Message{
				Role: "assistant",
				ToolCalls: []ToolCall{
					{
						ID:   response.Choices[0].Message.ToolCalls[0].ID,
						Type: "function",
						Function: ToolFunction{
							Name:      response.Choices[0].Message.ToolCalls[0].Function.Name,
							Arguments: response.Choices[0].Message.ToolCalls[0].Function.Arguments,
						},
					},
				},
			}

			messages = append(messages, assistantToolCallMessage)
			messages = append(messages, toolMessages...)
		}

		response, err = mc.generateResponse(ctx, messages, toolkit, 0.2, -1)
		if err != nil {
			return "", err
		}
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

	messages := []Message{
		{Role: "system", Content: "Generate a structured summary of the following task history, ensuring accuracy of each step."},
	}

	for _, entry := range history {
		messages = append(messages, Message{
			Role:    entry.Role,
			Content: fmt.Sprintf(entry.Content),
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
				log.Printf("⚠️ Attempt %d failed: HTTP %d | Error: %v", i+1, status, err)
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

func (mc *LMStudioClient) getEmbeddings(ctx context.Context, input string) ([]float64, error) {
	if cached, ok := mc.cache.Load(input); ok {
		if _, ok = cached.([]float64); ok {
			return cached.([]float64), nil
		}
	}

	payload := embeddingRequestPayload{
		Model: embeddingModel,
		Input: input,
	}

	response, status, err := mc.restClient.Post(ctx, embeddingEndpoint, payload, nil)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	if status != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", status)
	}

	var embeddingResp embeddingResponse
	if err = json.Unmarshal(response, &embeddingResp); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	if len(embeddingResp.Data) == 0 {
		return nil, errors.New("no embedding data received")
	}

	mc.cache.Store(input, embeddingResp.Data[0].Embedding)
	return embeddingResp.Data[0].Embedding, nil
}

func (mc *LMStudioClient) enrichWithEmbeddings(ctx context.Context, messages []Message) ([]Message, error) {
	var enhancedMessages []Message

	for _, msg := range messages {
		embedding, err := mc.getEmbeddings(ctx, msg.Content)
		if err != nil {
			log.Printf("⚠️ Error obtaining embedding for message: %s", msg.Content)
			enhancedMessages = append(enhancedMessages, msg)
			continue
		}

		embeddingStr := fmt.Sprintf("Embedding hash: %s", hashEmbedding(embedding))
		enhancedMessages = append(enhancedMessages, Message{
			Role:    msg.Role,
			Content: msg.Content + "\n\n" + embeddingStr,
		})
	}

	return enhancedMessages, nil
}

func hashEmbedding(embedding []float64) string {
	hash := sha256.New()
	for _, value := range embedding {
		hash.Write([]byte(fmt.Sprintf("%.6f", value)))
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func functionsToPayload(functions map[string]tools.Tool) (payload []FunctionPayload) {
	for _, function := range functions {
		payload = append(payload, FunctionPayload{
			"function",
			function,
		})
	}
	return payload
}
