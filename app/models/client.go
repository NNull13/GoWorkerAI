package models

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"

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

var v = validator.New()

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
	response, err := mc.generateResponse(ctx, messages, nil, temp, maxTokens, NoneToolChoice)
	if err != nil {
		return "", err
	}
	return response.Choices[0].Message.Content, nil
}

func (mc *LLMClient) TrueOrFalse(ctx context.Context, msgs []Message) (bool, string, error) {
	toolsPreset := tools.NewToolkitFromPreset(tools.PresetApprover)
	for attempt := 0; attempt < 3; attempt++ {
		resp, err := mc.generateResponse(ctx, msgs, toolsPreset, 0.13, -1, RequiredToolChoice)
		if err != nil {
			return false, "", err
		}
		if resp == nil || len(resp.Choices) == 0 {
			continue
		}

		msg := resp.Choices[0].Message
		if len(msg.ToolCalls) == 0 {
			log.Printf("TrueOrFalse attempt %d: model returned no tool call, content=%q", attempt, msg.Content)
			continue
		}

		toolTask := tools.ToolTask{Key: msg.ToolCalls[0].Function.Name}
		toolTask.Parameters, _ = utils.ParseArguments(msg.ToolCalls[0].Function.Arguments)
		funcTool, _ := toolsPreset[toolTask.Key]
		if funcTool.HandlerFunc != nil {
			var result string
			result, err = toolsPreset[toolTask.Key].HandlerFunc(toolTask)
			switch {
			case err != nil:
				if errors.Is(err, tools.ErrorRejected) {
					return false, result, nil
				}
				log.Printf("TrueOrFalse attempt %d: model returned tool call error: %s", attempt, err)
				continue
			default: //success
				return true, result, nil
			}
		}

	}

	return false, "", fmt.Errorf("yes/no: model did not call approve_plan or reject_plan after retries")
}

func (mc *LLMClient) Delegate(ctx context.Context, options, context, sysPrompt string) (*DelegateAction, error) {
	sys := Message{
		Role: "system",
		Content: sysPrompt + `Tooling policy:
- Use tool delegate_task to delegate atomic subtasks to a specific worker.
- If no worker fits or information is missing, choose "none" as worker to avoid task.
- Should always respond with an existing worker key in case you ask for any task.
- If you are not sure on which worker to delegate, choose the best from the list as assigned worker.
- If the task is finished, you must choose "none" as worker and "finish" as task and the reason of the finish as context.`,
	}
	user := Message{
		Role: "user",
		Content: "Context:\n" + context +
			"\nAvailable workers:\n" + options +
			"\nPick the single best worker and the next subtask now. " +
			"Always respond by calling delegate_task(Worker, Task, Context). " +
			"If you judge the overall task finished, call delegate_task with Worker=\"none\" and Task=\"finish\".",
	}

	msgs := []Message{sys, user}
	toolsPreset := tools.NewToolkitFromPreset(tools.PresetDelegate)
	for attempt := 0; attempt < 3; attempt++ {
		resp, err := mc.generateResponse(ctx, msgs, toolsPreset, 0.13, -1, RequiredToolChoice)
		if err != nil {
			return nil, err
		}
		if resp == nil || len(resp.Choices) == 0 {
			continue
		}

		msg := resp.Choices[0].Message
		if len(msg.ToolCalls) == 0 {
			log.Printf("Delegate attempt %d: model returned no tool call, content=%v", attempt, msg)
			continue
		}

		var action DelegateAction
		if err = json.Unmarshal([]byte(msg.ToolCalls[0].Function.Arguments), &action); err != nil {
			log.Printf("Delegate attempt %d: model returned invalid tool call arguments: %v", attempt, err)
			continue
		}
		if err = v.Struct(action); err != nil {
			log.Printf("Delegate attempt %d: model returned invalid tool call arguments: %v", attempt, err)
			continue
		}
		return &action, nil

	}

	return nil, fmt.Errorf("delegate: model did not choose any team member after retries")
}

func (mc *LLMClient) GenerateSummary(ctx context.Context, task string, history []storage.Record) (string, error) {
	content := "Here is the task:\n" + task + "\n"

	var recordHistory string
	if len(history) > 0 {
		recordHistory = storage.RecordListToString(history, 50)
		content = "Here is the history of task execution. Summarize it:" + recordHistory
	}

	messages := []Message{
		{Role: SystemRole, Content: SummarySystemPrompt},
		{Role: UserRole, Content: content},
	}

	response, err := mc.generateResponse(ctx, messages, nil, 0.10, 3850, NoneToolChoice)
	if err != nil {
		return "", err
	}
	if len(response.Choices) == 0 {
		return "", errors.New("empty LLM response for summary")
	}
	return response.Choices[0].Message.Content, nil
}

func (mc *LLMClient) Process(ctx context.Context, memberKey string, audit *log.Logger, messages []Message,
	toolkit map[string]tools.Tool, taskID string, stepID int) (string, error) {
	toolChoice := AutoToolChoice
	temp, maxTokens := 0.15, -1
	response, err := mc.generateResponse(ctx, messages, toolkit, temp, maxTokens, toolChoice)
	if err != nil {
		return "", err
	}

	message := response.Choices[0].Message
	toolChoice = NoneToolChoice
	newMessages := mc.handleToolCalls(ctx, audit, toolkit, message.ToolCalls, taskID, stepID, memberKey)
	messages = append(messages, newMessages...)
	if response, err = mc.generateResponse(ctx, messages, toolkit, temp, maxTokens, toolChoice); err != nil {
		return "", err
	}
	message = response.Choices[0].Message

	if len(message.Content) != 0 {
		if err = mc.storage.SaveHistory(ctx, storage.Record{
			TaskID:    taskID,
			MemberID:  memberKey,
			SubTaskID: int64(stepID),
			Role:      AssistantRole,
			Content:   message.Content,
			CreatedAt: time.Now(),
		}); err != nil {
			log.Printf("⚠️ Error saving history for task %s: %v", taskID, err)
		}

		return message.Content, nil
	}

	return "Successfully processed", nil
}

func (mc *LLMClient) handleToolCalls(ctx context.Context, audit *log.Logger, toolkit map[string]tools.Tool,
	toolCalls []toolCall, taskID string, stepID int, memberKey string) (messages []Message) {
	messages = append(messages, Message{Role: AssistantRole, ToolCalls: toolCalls})

	for _, call := range toolCalls {
		audit.Printf("▶️ Executing: %v", call)
		toolTask := tools.ToolTask{Key: call.Function.Name}
		toolTask.Parameters, _ = utils.ParseArguments(call.Function.Arguments)
		tool, exists := toolkit[toolTask.Key]
		if !exists || tool.HandlerFunc == nil {
			messages = append(messages, Message{
				Role:       ToolRole,
				Content:    fmt.Sprintf("ERROR: tool '%s' not found or missing handler", toolTask.Key),
				ToolCallID: call.ID,
			})
			continue
		}

		result, err := tool.HandlerFunc(toolTask)
		if err != nil {
			audit.Printf("⚠️ Tool %s execution failed: %v", tool.Name, err)
			result = err.Error()
		}

		if err = mc.storage.SaveHistory(ctx, storage.Record{
			TaskID:     taskID,
			SubTaskID:  int64(stepID),
			MemberID:   memberKey,
			Role:       ToolRole,
			Tool:       tool.Name,
			Content:    result,
			Parameters: call.Function.Arguments,
			CreatedAt:  time.Now(),
		}); err != nil {
			audit.Printf("⚠️ Error saving history for tool %s: %v", tool.Name, err)
		}

		// Always add tool result to messages, even if empty
		// This preserves conversation context for the LLM
		resultContent := result
		if len(result) == 0 {
			audit.Printf("⚠️ Tool %s returned empty result", tool.Name)
			resultContent = fmt.Sprintf("[Tool %s executed but returned no output]", tool.Name)
		} else {
			audit.Printf("✅ Tool %s executed successfully, result: %s", tool.Name, result)
		}

		messages = append(messages,
			Message{
				Role:       ToolRole,
				Content:    resultContent,
				ToolCallID: call.ID,
			},
		)
	}

	return messages
}

func (mc *LLMClient) generateResponse(ctx context.Context, messages []Message, tools map[string]tools.Tool,
	temp float64, maxTokens int, toolChoice any) (*ResponseLLM, error) {
	messagesCurated := make([]Message, 0, len(messages))
	hasUserPrompt := false
	for _, msg := range messages {
		if len(msg.Content) > 0 {
			messagesCurated = append(messagesCurated, msg)
		}
		if msg.Role == UserRole {
			hasUserPrompt = true
		}
	}
	if !hasUserPrompt {
		return nil, errors.New("no user prompt found")
	}

	payload := requestPayload{
		Model:       mc.model,
		Tools:       functionsToPayload(tools),
		Messages:    messagesCurated,
		Temperature: temp,
		MaxTokens:   maxTokens,
		ToolChoice:  toolChoice,
	}

	return mc.sendRequestAndParse(ctx, payload)
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

func (mc *LLMClient) sendRequestAndParse(ctx context.Context, payload requestPayload) (*ResponseLLM, error) {
	respBytes, status, err := mc.restClient.Post(ctx, endpoint, payload, nil)
	if err == nil && status >= 200 && status < 300 {
		if status == 400 {
			log.Printf("⚠️ HTTP400 LLM request failed: request %v", payload)
		}
		var out ResponseLLM
		if uErr := json.Unmarshal(respBytes, &out); uErr != nil {
			err = fmt.Errorf("unmarshal: %w", uErr)
		}
		return &out, nil
	}
	return nil, fmt.Errorf("request failed: %w", err)
}
