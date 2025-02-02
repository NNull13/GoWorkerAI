package restclient

type Interface interface {
	Get(endpoint string, headers map[string]string) ([]byte, int, error)
	Post(endpoint string, body any, headers map[string]string) ([]byte, int, error)
	Put(endpoint string, body any, headers map[string]string) ([]byte, int, error)
	Delete(endpoint string, headers map[string]string) ([]byte, int, error)
}
