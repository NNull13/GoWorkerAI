package models

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
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
)

var _ Interface = &LLMClient{}

type LLMClient struct {
	restClient      *restclient.RestClient
	storage         storage.Interface
	cache           sync.Map
	model           string
	embeddingsModel string
}

func NewLLMClient(db storage.Interface, model, embModel string) *LLMClient {
	baseURL := os.Getenv("LLM_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:1234"
	}
	return &LLMClient{
		restClient:      restclient.NewRestClient(baseURL, nil),
		storage:         db,
		model:           model,
		embeddingsModel: embModel,
	}
}

func (mc *LLMClient) Think(ctx context.Context, messages []Message, temp float64, maxTokens int) (string, error) {
	response, err := mc.generateResponse(ctx, messages, nil, temp, maxTokens)
	if err != nil {
		return "", err
	}
	return response.Choices[0].Message.Content, nil
}

func (mc *LLMClient) YesOrNo(ctx context.Context, messages []Message) (bool, error) {
	// agrega instrucción fuerte al final del prompt del usuario
	msgs := append([]Message{}, Message{
		Role:    "system",
		Content: "Return ONLY a JSON object like: {\"answer\": true} or {\"answer\": false}. No extra text.",
	})
	msgs = append(msgs, messages...)

	for i := 0; i < 3; i++ {
		response, err := mc.generateResponse(ctx, msgs, nil, 0.1, 25)
		if err != nil {
			return false, err
		}

		lt := strings.ToLower(strings.TrimSpace(response.Choices[0].Message.Content))
		switch {
		case strings.Contains(lt, "true"), strings.Contains(lt, "yes"):
			return true, nil
		case strings.Contains(lt, "false"), strings.Contains(lt, "no"):
			return false, nil
		default:
			log.Printf("🤔 Unexpected answer: %s", response.Choices[0].Message.Content)
		}
	}
	return false, fmt.Errorf("unexpected yes/no")
}

func (mc *LLMClient) GenerateSummary(ctx context.Context, history []storage.Record) (string, error) {
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

	content := "Here is the history of task execution. Summarize it:"
	for _, entry := range history {
		content += fmt.Sprintf("\nRole: %s | Content: %s | Tool: %s | Step: %d", entry.Role, entry.Content, entry.Tool, entry.StepID)
	}

	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: content},
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

func (mc *LLMClient) Process(ctx context.Context, messages []Message, toolkit map[string]tools.Tool,
	taskID string, stepID int) (string, error) {
	response, err := mc.generateResponse(ctx, messages, toolkit, 0.2, -1)
	if err != nil {
		return "", err
	}

	message := response.Choices[0].Message
	for i := 0; i < 5; i++ {
		messages = append(messages, mc.handleToolCalls(ctx, toolkit, message.ToolCalls, taskID, stepID)...)
		if response, err = mc.generateResponse(ctx, messages, toolkit, 0.2, -1); err != nil {
			return "", err
		}
		message = response.Choices[0].Message
		if len(message.ToolCalls) == 0 {
			break
		}
	}

	return message.Content, nil
}

func (mc *LLMClient) handleToolCalls(ctx context.Context, toolkit map[string]tools.Tool,
	toolCalls []toolCall, taskID string, stepID int) (messages []Message) {
	messages = append(messages, Message{Role: "assistant", ToolCalls: toolCalls})

	for i, call := range toolCalls {
		log.Printf("▶️ Executing tool call %v: %v", i, call)
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

		if err := mc.storage.SaveHistory(ctx, storage.Record{
			TaskID:     taskID,
			StepID:     int64(stepID),
			Role:       "tool",
			Tool:       tool.Name,
			Content:    result,
			Parameters: call.Function.Arguments,
			CreatedAt:  time.Now(),
		}); err != nil {
			log.Printf("⚠️ Error saving history for tool %s: %v", tool.Name, err)
		}

		messages = append(messages,
			Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: call.ID,
			},
		)
	}

	return messages
}

func (mc *LLMClient) generateResponse(ctx context.Context, messages []Message, tools map[string]tools.Tool, temp float64, maxTokens int) (*ResponseLLM, error) {
	payload := requestPayload{
		Model:       mc.model,
		Tools:       functionsToPayload(tools),
		Messages:    messages,
		Temperature: temp,
		MaxTokens:   maxTokens,
	}

	return mc.sendRequestAndParse(ctx, payload, 3)
}

func functionsToPayload(functions map[string]tools.Tool) (payload []functionPayload) {
	names := make([]string, 0, len(functions))
	for name := range functions {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		t := functions[name]
		payload = append(payload, functionPayload{Type: "function", Function: t})
	}
	return payload
}

func (mc *LLMClient) sendRequestAndParse(ctx context.Context, payload requestPayload, maxRetries int) (*ResponseLLM, error) {
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
			if err != nil {
				time.Sleep(time.Duration(math.Pow(2, float64(i))) * 100 * time.Millisecond)
			}
			response, status, err = mc.restClient.Post(ctx, endpoint, payload, nil)
			if err != nil {
				log.Printf("⚠️ Attempt %d failed: HTTP %d | Response: %s | Error: %v | Payload %v",
					i+1, status, string(response), err, payload)
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
