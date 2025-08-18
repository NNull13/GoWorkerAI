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
	sys := Message{
		Role: "system",
		Content: `You must make a binary decision by calling just ONE tool:
		- "approve_plan"  (when the answer is TRUE)
		- "reject_plan"   (when the answer is FASLE)`,
	}

	msgs := make([]Message, 0, len(messages)+1) // +1 for system message
	msgs = append(msgs, sys)
	msgs = append(msgs, messages...)
	toolsPreset := tools.NewToolkitFromPreset(tools.PresetPlanReviewer)
	for attempt := 0; attempt < 3; attempt++ {
		resp, err := mc.generateResponse(ctx, msgs, toolsPreset, 0.13, -1)
		if err != nil {
			return false, err
		}
		if resp == nil || len(resp.Choices) == 0 {
			continue
		}

		msg := resp.Choices[0].Message
		if len(msg.ToolCalls) == 0 {
			log.Printf("YesOrNo attempt %d: model returned no tool call, content=%q", attempt, msg.Content)
			continue
		}

		name := msg.ToolCalls[0].Function.Name
		switch name {
		case "approve_plan":
			return true, nil
		case "reject_plan":
			return false, nil
		default:
			log.Printf("🤔 Unexpected answer: %+v", msg)
		}

	}

	return false, fmt.Errorf("yes/no: model did not call approve_plan or reject_plan after retries")
}

func (mc *LLMClient) GenerateSummary(ctx context.Context, task string, auditLogs []string,
	history []storage.Record) (string, error) {
	systemPrompt := `You will receive the task to be completed and a flat history of task execution entries in as a series of audit logs:
	Your job: produce a compact, high-signal, strictly chronological timeline of the execution, enabling a separate 
	evaluator to decide YES/NO readiness using this timeline if the task is complete
	Rules for summary:
	- Do not include the task itself in the summary.
	- Do not include the audit logs in the summary.
	- Only include in the timeline the executions that are relevant to the task.	
	- Output ONLY a numbered list of entries.
	- Start at 1 and increment by 1.
	- Exactly one line per entry. No text before, between, or after entries.
        - Write each entry as an explicit, past-tense execution statement (what was DONE), not an instruction.
	- Required Output Format is:
	"1. [Description of the first entry]\n2. [Description of the next entry]\n...\nN. [Final entry]\n".`

	content := "Here is the task:\n" + task + "\nHere is the audit logs:"
	for _, log := range auditLogs {
		content += fmt.Sprintf("\n%s", log)
	}

	if len(history) > 0 {
		content = "Here is the history of task execution. Summarize it:"
		for _, entry := range history {
			content += fmt.Sprintf("\nRole: %s | Content: %s | Tool: %s | Step: %d | ID: %d",
				entry.Role, entry.Content, entry.Tool, entry.StepID, entry.ID)
		}
	}
	
	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: content},
	}

	response, err := mc.generateResponse(ctx, messages, nil, 0.25, -1)
	if err != nil {
		return "", err
	}
	if len(response.Choices) == 0 {
		return "", errors.New("empty LLM response for summary")
	}
	return response.Choices[0].Message.Content, nil
}

func (mc *LLMClient) Process(ctx context.Context, audit *log.Logger, messages []Message, toolkit map[string]tools.Tool,
	taskID string, stepID int) (string, error) {
	response, err := mc.generateResponse(ctx, messages, toolkit, 0.2, -1)
	if err != nil {
		return "", err
	}

	message := response.Choices[0].Message
	for i := 0; i < 5; i++ {
		newMessages := mc.handleToolCalls(ctx, audit, toolkit, message.ToolCalls, taskID, stepID)
		messages = append(messages, newMessages...)
		if response, err = mc.generateResponse(ctx, messages, toolkit, 0.2, -1); err != nil {
			return "", err
		}
		message = response.Choices[0].Message
		if len(message.ToolCalls) == 0 {
			break
		}
	}

	if err = mc.storage.SaveHistory(ctx, storage.Record{
		TaskID:    taskID,
		StepID:    int64(stepID),
		Role:      "assistant",
		Content:   message.Content,
		CreatedAt: time.Now(),
	}); err != nil {
		log.Printf("⚠️ Error saving history for task %s: %v", taskID, err)
	}

	return message.Content, nil
}

func (mc *LLMClient) handleToolCalls(ctx context.Context, audit *log.Logger, toolkit map[string]tools.Tool,
	toolCalls []toolCall, taskID string, stepID int) (messages []Message) {
	messages = append(messages, Message{Role: "assistant", ToolCalls: toolCalls})

	for i, call := range toolCalls {
		audit.Printf("▶️ Executing tool call %v: %v", i, call)
		toolTask := tools.ToolTask{Key: call.Function.Name}
		toolTask.Parameters, _ = utils.ParseArguments(call.Function.Arguments)
		tool, exists := toolkit[toolTask.Key]
		if !exists || tool.HandlerFunc == nil {
			audit.Printf("⚠️ Tool not found or missing handler: %s", toolTask.Key)
			continue
		}

		result, err := tool.HandlerFunc(toolTask)
		if err != nil {
			audit.Printf("⚠️ Tool %s execution failed: %v", tool.Name, err)
			result = err.Error()
		}

		if err = mc.storage.SaveHistory(ctx, storage.Record{
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
					i, status, string(response), err, payload)
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
