package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	utils2 "GoEngineerAI/app/utils"
)

const endpoint = "/v1/chat/completions"
const model = "qwen2.5-coder-7b-instruct"

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Action struct {
	Action   string `json:"action"`
	Filename string `json:"filename,omitempty"`
	Content  string `json:"content,omitempty"`
	Result   string `json:"result,omitempty"`
}

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
	SystemFingerprint string `json:"system_fingerprint"`
}

type ModelClient struct {
	restClient *utils2.RestClient
}

func NewModelClient() *ModelClient {
	return &ModelClient{
		restClient: utils2.NewRestClient("http://localhost:1234", nil),
	}
}

func (mc *ModelClient) Think(messages []Message) (string, error) {
	payload := requestPayload{
		Model:       model,
		Messages:    messages,
		Temperature: 0.69,
		MaxTokens:   -1,
		Stream:      false,
	}

	generatedResponse, err := mc.sendRequestAndParse(payload, 3)
	if err != nil {
		return "", err
	}

	if generatedResponse.Choices == nil {
		return "", errors.New("no choices found")
	}
	return generatedResponse.Choices[0].Message.Content, nil
}

func (mc *ModelClient) Process(messages []Message) (*Action, error) {
	payload := requestPayload{
		Model:       model,
		Messages:    messages,
		Temperature: 0.69,
		MaxTokens:   -1,
		Stream:      false,
	}

	generatedResponse, err := mc.sendRequestAndParse(payload, 3)
	if err != nil {
		return nil, err
	}

	rawContent := strings.TrimSpace(generatedResponse.Choices[0].Message.Content)
	rawContent = strings.ReplaceAll(rawContent, "```json", "")
	rawContent = strings.ReplaceAll(rawContent, "```", "")
	marker := ",\n  \"content\": "
	start := strings.Index(rawContent, marker)
	end := strings.LastIndex(rawContent, "}")

	var contentAction string
	if start != -1 && end != -1 && start < end {
		contentAction = rawContent[start+len(marker)+1 : end-2]
		contentAction = utils2.UnescapeIfNeeded(contentAction)
		rawContent = utils2.RemoveSubstring(rawContent, start, end)
	}

	var action Action
	if err = json.Unmarshal([]byte(rawContent), &action); err != nil {
		return nil, fmt.Errorf("failed to parse response as JSON: %w response: %s", err, rawContent)
	}

	if contentAction != "" {
		action.Content += contentAction
	}

	return &action, nil
}

func (mc *ModelClient) YesOrNo(messages []Message, retry int) (bool, error) {
	systemPrompt := Message{
		Role:    "system",
		Content: "Answer only with 'true' for yes or 'false' for no. No additional text.",
	}
	messages = append([]Message{systemPrompt}, messages...)

	payload := requestPayload{
		Model:       model,
		Messages:    messages,
		Temperature: 0.0,
		MaxTokens:   1,
		Stream:      false,
	}

	generatedResponse, err := mc.sendRequestAndParse(payload, 3)
	if err != nil {
		return false, err
	}

	cleanedResponse := strings.ToLower(generatedResponse.Choices[0].Message.Content)
	if cleanedResponse != "true" && cleanedResponse != "false" {
		return false, fmt.Errorf("not boolean response: %s", cleanedResponse)
	}
	return cleanedResponse == "true", err
}

func (mc *ModelClient) sendRequestAndParse(payload requestPayload, retry int) (*responseLLM, error) {
	var err error
	var response []byte
	var status int
	var generatedResponse *responseLLM

	for i := 0; i < retry; i++ {
		response, status, err = mc.restClient.Post(endpoint, payload, nil)
		if status != 200 {
			err = fmt.Errorf("http error %v, response: %v , error %w", status, string(response), err)
			log.Printf("%s", err.Error())
			continue
		}

		generatedResponse, err = parseLLMResponse(response)
		if err != nil {
			log.Printf("Error parsing response: %s err: %v", response, err)
			continue
		}
		break
	}

	if err != nil {
		return nil, err
	}

	return generatedResponse, nil
}

func parseLLMResponse(jsonData []byte) (*responseLLM, error) {
	var llmResponse responseLLM
	err := json.Unmarshal(jsonData, &llmResponse)
	if err != nil {
		return nil, err
	}
	return &llmResponse, nil
}
