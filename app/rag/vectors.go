package rag

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
)

type QdrantStore struct {
	client     *qdrant.Client
	collection string
}

func NewQdrantStore(collection string) (vectorStore, error) {
	url := os.Getenv("QDRANT_URL")
	port, _ := strconv.Atoi(os.Getenv("QDRANT_PORT"))
	if url == "" {
		url = "localhost"
	}
	if port == 0 {
		port = 6334
	}
	client, err := qdrant.NewClient(&qdrant.Config{
		Host: url,
		Port: port,
	})
	if err != nil {
		return nil, err
	}
	return &QdrantStore{
		client:     client,
		collection: collection,
	}, nil
}

func (s *QdrantStore) InitContext(ctx context.Context, vectorSize int) (bool, error) {
	exists, err := s.client.CollectionExists(ctx, s.collection)
	if err != nil {
		return false, err
	}
	if !exists {
		if err = s.client.CreateCollection(ctx, &qdrant.CreateCollection{
			CollectionName: s.collection,
			VectorsConfig: &qdrant.VectorsConfig{
				Config: &qdrant.VectorsConfig_Params{
					Params: &qdrant.VectorParams{
						Size:     uint64(vectorSize),
						Distance: qdrant.Distance_Cosine,
					},
				},
			},
		}); err != nil {
			return exists, fmt.Errorf("create collection: %w", err)
		}
	}
	return exists, nil
}

func (s *QdrantStore) Close() error {
	return s.client.Close()
}

func (s *QdrantStore) UpsertBatch(ctx context.Context, docs []VectorDoc) error {
	pts := make([]*qdrant.PointStruct, len(docs))

	for i, d := range docs {
		id := d.ID
		if id == "" {
			id = uuid.New().String()
		}

		payload := map[string]any{
			"text": d.Content,
		}
		for k, v := range d.Metadata {
			payload[k] = v
		}

		pts[i] = &qdrant.PointStruct{
			Id:      qdrant.NewIDUUID(id),
			Vectors: qdrant.NewVectors(d.Vector...),
			Payload: qdrant.NewValueMap(payload),
		}
	}

	_, err := s.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: s.collection,
		Points:         pts,
	})

	return err
}

func (s *QdrantStore) Query(ctx context.Context, vector []float32, filters map[string]string, k int) ([]VectorDoc, error) {
	limit := uint64(k)
	var filter *qdrant.Filter
	if len(filters) > 0 {
		filter = &qdrant.Filter{
			Must: []*qdrant.Condition{},
		}
		for key, f := range filters {
			filter.Must = append(filter.Must, qdrant.NewMatch(key, f))
		}
	}
	resp, err := s.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: s.collection,
		Limit:          &limit,
		Filter:         filter,
		Query:          qdrant.NewQuery(vector...),
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, err
	}

	var out []VectorDoc

	for _, r := range resp {
		md := make(map[string]any)
		for key, v := range r.Payload {
			md[key] = convertQdrantValue(v)
		}

		content := ""
		if val, ok := md["text"]; ok {
			content = fmt.Sprintf("%v", val)
		}

		var id string
		if r.Id != nil {
			switch x := r.Id.PointIdOptions.(type) {
			case *qdrant.PointId_Uuid:
				id = x.Uuid
			case *qdrant.PointId_Num:
				id = fmt.Sprintf("%d", x.Num)
			}
		}

		out = append(out, VectorDoc{
			ID:       id,
			Content:  content,
			Metadata: md,
		})
	}

	return out, nil

}

func convertQdrantValue(v *qdrant.Value) any {
	switch val := v.Kind.(type) {

	case *qdrant.Value_BoolValue:
		return val.BoolValue

	case *qdrant.Value_IntegerValue:
		return val.IntegerValue

	case *qdrant.Value_DoubleValue:
		return val.DoubleValue

	case *qdrant.Value_StringValue:
		return val.StringValue

	case *qdrant.Value_NullValue:
		return nil

	case *qdrant.Value_ListValue:
		out := make([]any, len(val.ListValue.Values))
		for i, lv := range val.ListValue.Values {
			out[i] = convertQdrantValue(lv)
		}
		return out

	case *qdrant.Value_StructValue:
		out := make(map[string]any)
		for k, nv := range val.StructValue.Fields {
			out[k] = convertQdrantValue(nv)
		}
		return out
	}

	return nil
}
