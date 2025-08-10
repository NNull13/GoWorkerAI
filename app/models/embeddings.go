package models

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"
)

func (mc *LLMClient) EmbedText(ctx context.Context, input string) ([]float32, error) {
	if v, ok := mc.cache.Load(input); ok {
		if emb, ok2 := v.([]float32); ok2 {
			return emb, nil
		}
	}

	if mc.embeddingsModel == "" {
		return nil, errors.New("embeddings model is empty; configure LLMClient.embeddingsModel")
	}

	req := embeddingRequestPayload{
		Model: mc.embeddingsModel,
		Input: input,
	}
	resp, err := mc.sendEmbeddings(ctx, req, 3)
	if err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 {
		return nil, errors.New("no embedding data returned")
	}
	emb := resp.Data[0].Embedding
	mc.cache.Store(input, emb)
	return emb, nil
}

func (mc *LLMClient) sendEmbeddings(ctx context.Context, payload embeddingRequestPayload, maxRetries int) (*embeddingResponse, error) {
	var (
		lastErr error
		body    []byte
		status  int
		out     embeddingResponse
	)

	for i := 0; i < maxRetries; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if i > 0 {
			sleep := time.Duration(100*(1<<uint(i))) * time.Millisecond
			sleep += time.Duration(time.Now().UnixNano() % int64(100*time.Millisecond))
			time.Sleep(sleep)
		}

		b, s, err := mc.restClient.Post(ctx, embeddingEndpoint, payload, nil)
		body, status, lastErr = b, s, err
		if err != nil {
			log.Printf("⚠️ embed attempt %d failed: http=%d err=%v", i+1, status, err)
			continue
		}
		if err := json.Unmarshal(body, &out); err != nil {
			lastErr = fmt.Errorf("parse embeddings json: %w", err)
			log.Printf("⚠️ %v", lastErr)
			continue
		}

		return &out, nil
	}
	return nil, fmt.Errorf("embeddings request failed after %d retries: %w", maxRetries, lastErr)
}
