package rag

import (
	"context"
	"os"
	"path/filepath"

	"github.com/google/uuid"

	"GoWorkerAI/app/models"
	"GoWorkerAI/app/utils"
)

const (
	chunkSize  = 500
	overlap    = 100
	vectorSize = 2560

	collectionName = "rag"
)

type Client struct {
	vectors vectorStore
	model   models.Interface
}

func NewClient(model models.Interface) Interface {
	vectors, err := NewQdrantStore(collectionName)
	if err != nil {
		panic(err)
	}
	return &Client{
		model:   model,
		vectors: vectors,
	}
}

func (c Client) Search(ctx context.Context, text string, filters map[string]string, k int) ([]VectorDoc, error) {
	vec, err := c.model.EmbedText(ctx, text)
	if err != nil {
		return nil, err
	}
	return c.vectors.Query(ctx, vec, filters, k)
}

func (c Client) InitContext(ctx context.Context) error {
	alreadyExists, err := c.vectors.InitContext(ctx, vectorSize)
	if err != nil {
		return err
	}
	if alreadyExists {
		return nil
	}

	folderRag := os.Getenv("FOLDER_RAG")
	if folderRag == "" {
		folderRag = "./rag_data"
	}
	paths, err := utils.LoadFilesFromDir(folderRag)
	if err != nil {
		return err
	}

	for _, p := range paths {
		var text string
		if text, err = utils.ReadFile(p); err != nil {
			return err
		}

		chunks := ChunkText(text, chunkSize, overlap)
		batch := make([]VectorDoc, 0, len(chunks))

		for i, ch := range chunks {
			var vec []float32
			if vec, err = c.model.EmbedText(ctx, ch); err != nil {
				return err
			}
			batch = append(batch, VectorDoc{
				ID:      uuid.New().String(),
				Content: ch,
				Metadata: map[string]any{
					"source": filepath.Base(p),
					"chunk":  i,
				},
				Vector: vec,
			})
		}

		if err = c.vectors.UpsertBatch(ctx, batch); err != nil {
			return err
		}
	}

	return nil
}

func ChunkText(text string, size, overlap int) []string {
	runes := []rune(text)
	var chunks []string

	for start := 0; start < len(runes); start += size - overlap {
		end := start + size
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[start:end]))
		if end == len(runes) {
			break
		}
	}

	return chunks
}
