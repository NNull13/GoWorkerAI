package models

import "GoWorkerAI/app/tools"

type toolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function toolFunction `json:"function"`
}

type toolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type functionPayload struct {
	Type     string     `json:"type"`
	Function tools.Tool `json:"function"`
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

type requestPayload struct {
	Model       string            `json:"model"`
	Messages    []Message         `json:"messages"`
	Temperature float64           `json:"temperature"`
	MaxTokens   int               `json:"max_tokens"`
	Tools       []functionPayload `json:"tools"`
}

type embeddingRequestPayload struct {
	Model string      `json:"model"`
	Input interface{} `json:"input"`
}

type embeddingItem struct {
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

type embeddingResponse struct {
	Data  []embeddingItem `json:"data"`
	Model string          `json:"model"`
}
