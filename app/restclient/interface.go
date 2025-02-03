package restclient

import "context"

type Interface interface {
	Get(context.Context, string, map[string]string) ([]byte, int, error)
	Post(context.Context, string, any, map[string]string) ([]byte, int, error)
	Put(context.Context, string, any, map[string]string) ([]byte, int, error)
	Delete(context.Context, string, map[string]string) ([]byte, int, error)
}
