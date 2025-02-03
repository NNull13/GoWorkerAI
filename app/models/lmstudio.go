package models

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"GoWorkerAI/app/restclient"
	"GoWorkerAI/app/utils"
)

const (
	endpoint = "/v1/chat/completions"
	model    = "qwen2.5-coder-7b-instruct"
)

type requestPayload struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
	Stream      bool      `json:"stream"`
}

type responseLLM struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int     `json:"index"`
		Logprobs     *string `json:"logprobs"`
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
}

func NewLMStudioClient() *LMStudioClient {
	return &LMStudioClient{
		restClient: restclient.NewRestClient("http://localhost:1234", nil),
	}
}

func (mc *LMStudioClient) Think(ctx context.Context, messages []Message) (string, error) {
	payload := requestPayload{
		Model:       model,
		Messages:    messages,
		Temperature: 0.420,
		MaxTokens:   -1,
		Stream:      false,
	}

	generatedResponse, err := mc.sendRequestAndParse(ctx, payload, 3)
	if err != nil {
		return "", err
	}

	if len(generatedResponse.Choices) == 0 {
		return "", errors.New("no valid response from model")
	}

	return strings.TrimSpace(generatedResponse.Choices[0].Message.Content), nil
}

func (mc *LMStudioClient) Process(ctx context.Context, messages []Message) (*ActionTask, error) {
	payload := requestPayload{
		Model:       model,
		Messages:    messages,
		Temperature: 0.7,
		MaxTokens:   500,
	}

	generatedResponse, err := mc.sendRequestAndParse(ctx, payload, 3)
	if err != nil {
		return nil, err
	}
	log.Printf("Generated response: %s", generatedResponse.Choices[0].Message.Content)

	rawContent := strings.TrimSpace(generatedResponse.Choices[0].Message.Content)
	rawContent = strings.ReplaceAll(rawContent, "```json", "")
	rawContent = strings.ReplaceAll(rawContent, "```", "")
	marker := ",\n  \"content\": "
	start := strings.Index(rawContent, marker)
	end := strings.LastIndex(rawContent, "}")

	var contentAction string
	if start != -1 && end != -1 && start < end {
		contentAction = rawContent[start+len(marker)+1 : end-2]
		contentAction = utils.UnescapeIfNeeded(contentAction)
		rawContent = utils.RemoveSubstring(rawContent, start, end)
	}

	var action ActionTask
	if err = json.Unmarshal([]byte(rawContent), &action); err != nil {
		return nil, fmt.Errorf("failed to parse response as JSON: %w response: %s", err, rawContent)
	}

	if contentAction != "" {
		action.Content += contentAction
	}

	return &action, nil
}

func (mc *LMStudioClient) YesOrNo(ctx context.Context, messages []Message) (bool, error) {
	payload := requestPayload{
		Model:       model,
		Messages:    messages,
		Temperature: 0.0,
		MaxTokens:   1,
	}

	generatedResponse, err := mc.sendRequestAndParse(ctx, payload, 3)
	if err != nil {
		return false, err
	}

	response := strings.ToLower(strings.TrimSpace(generatedResponse.Choices[0].Message.Content))
	if response != "true" && response != "false" {
		return false, fmt.Errorf("unexpected response: %s", response)
	}
	return response == "true", nil
}

func (mc *LMStudioClient) sendRequestAndParse(ctx context.Context, payload requestPayload, maxRetries int) (*responseLLM, error) {
	var err error
	var response []byte
	var status int
	var generatedResponse *responseLLM

	for i := 0; i < maxRetries; i++ {
		select {
		case <-ctx.Done():
			log.Println("🚨 Request canceled before execution")
			return nil, ctx.Err()
		default:
			delay := time.Duration(math.Pow(2, float64(i))) * 100 * time.Millisecond
			time.Sleep(delay)

			response, status, err = mc.restClient.Post(ctx, endpoint, payload, nil)
			if err != nil || status != 200 {
				log.Printf("⚠️ Attempt %d failed: HTTP %d | Error: %v", i+1, status, err)
				continue
			}

			err = json.Unmarshal(response, &generatedResponse)
			if err != nil {
				log.Printf("⚠️ Error parsing response: %v", err)
				continue
			}

			return generatedResponse, nil
		}
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", maxRetries, err)
}
