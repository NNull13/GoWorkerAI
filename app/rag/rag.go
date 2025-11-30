package rag

import "context"

type VectorDoc struct {
	ID       string
	Content  string
	Metadata map[string]any
	Vector   []float32
}

type Interface interface {
	Search(ctx context.Context, text string, filters map[string]string, k int) ([]VectorDoc, error)
	InitContext(context.Context) error
}

type vectorStore interface {
	UpsertBatch(ctx context.Context, docs []VectorDoc) error
	Query(ctx context.Context, vector []float32, filters map[string]string, k int) ([]VectorDoc, error)
	InitContext(ctx context.Context, vectorSize int) (bool, error)
	Close() error
}
