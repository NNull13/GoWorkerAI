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
	"GoWorkerAI/app/utils"
)

const (
	endpoint          = "/v1/chat/completions"
	embeddingEndpoint = "/v1/embeddings"
	model             = "qwen2.5-7b-instruct-1m"
	embeddingModel    = "text-embedding-nomic-embed-text-v1.5-embedding"
)

type requestPayload struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
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
	cache      sync.Map
}

func NewLMStudioClient() *LMStudioClient {
	return &LMStudioClient{
		restClient: restclient.NewRestClient("http://localhost:1234", nil),
	}
}

func (mc *LMStudioClient) GenerateResponse(ctx context.Context, messages []Message, temperature float64, maxTokens int) (string, error) {
	enhancedMessages, err := mc.enrichWithEmbeddings(ctx, messages)
	if err != nil {
		return "", err
	}
	payload := requestPayload{
		Model:       model,
		Messages:    enhancedMessages,
		Temperature: temperature,
		MaxTokens:   maxTokens,
	}
	return mc.sendRequestAndParse(ctx, payload, 3)
}

func (mc *LMStudioClient) Process(ctx context.Context, messages []Message) (*ActionTask, error) {
	response, err := mc.GenerateResponse(ctx, messages, 0.2, -1)
	if err != nil {
		return nil, err
	}

	log.Printf("Generated response: %s", response)
	response = strings.TrimSpace(response)
	response = strings.ReplaceAll(response, "```json", "")
	response = strings.ReplaceAll(response, "```", "")
	marker := ",\n  \"content\": "
	start := strings.Index(response, marker)
	end := strings.LastIndex(response, "}")

	var contentAction string
	if start != -1 && end != -1 && start < end {
		contentAction = response[start+len(marker)+1 : end-2]
		contentAction = utils.UnescapeIfNeeded(contentAction)
		response = utils.RemoveSubstring(response, start, end)
	}

	var action ActionTask
	if err = json.Unmarshal([]byte(response), &action); err != nil {
		return nil, fmt.Errorf("failed to parse response as JSON: %w response: %s", err, response)
	}

	if contentAction != "" {
		action.Content += contentAction
	}

	return &action, nil
}

func (mc *LMStudioClient) YesOrNo(ctx context.Context, messages []Message) (bool, error) {
	response, err := mc.GenerateResponse(ctx, messages, 0.0, 1)
	if err != nil {
		return false, err
	}

	lowerResp := strings.ToLower(strings.TrimSpace(response))
	if lowerResp != "true" && lowerResp != "false" {
		return false, fmt.Errorf("unexpected response: %v", response)
	}

	return lowerResp == "true", nil
}

func (mc *LMStudioClient) GenerateSummary(ctx context.Context, history []storage.Record) (string, error) {
	var combinedContent strings.Builder
	for i, entry := range history {
		combinedContent.WriteString(fmt.Sprintf("Record %d %v\n", i, entry))
	}

	summaryPrompt := []Message{
		{Role: "system", Content: "Generate a concise summary of the following task history."},
		{Role: "user", Content: combinedContent.String()},
	}

	return mc.GenerateResponse(ctx, summaryPrompt, 0.1, 500)
}

func (mc *LMStudioClient) sendRequestAndParse(ctx context.Context, payload requestPayload, maxRetries int) (string, error) {
	var err error
	var response []byte
	var status int
	var generatedResponse responseLLM

	for i := 0; i < maxRetries; i++ {
		select {
		case <-ctx.Done():
			log.Println("🚨 Request canceled before execution")
			return "", ctx.Err()
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

			return generatedResponse.Choices[0].Message.Content, nil
		}
	}

	return "", fmt.Errorf("request failed after %d retries: %w", maxRetries, err)
}

func (mc *LMStudioClient) getEmbeddings(ctx context.Context, input string) ([]float64, error) {
	if cached, ok := mc.cache.Load(input); ok {
		return cached.([]float64), nil
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
