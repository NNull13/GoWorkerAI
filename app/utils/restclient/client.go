package restclient

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
)

type RestClient struct {
	baseURL    string
	headers    map[string]string
	httpClient *http.Client
}

func NewRestClient(baseURL string, headers map[string]string) *RestClient {
	return &RestClient{
		baseURL:    baseURL,
		headers:    headers,
		httpClient: &http.Client{},
	}
}

func (c *RestClient) setHeaders(req *http.Request, headers map[string]string) {
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	for key, value := range c.headers {
		req.Header.Set(key, value)
	}
}

func (c *RestClient) doRequest(ctx context.Context, request *http.Request) ([]byte, int, error) {
	request = request.WithContext(ctx)

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, 0, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	return body, response.StatusCode, err
}

func (c *RestClient) Get(ctx context.Context, endpoint string, headers map[string]string) ([]byte, int, error) {
	url := c.baseURL + endpoint
	request, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, 0, err
	}
	c.setHeaders(request, headers)
	return c.doRequest(ctx, request)
}

func (c *RestClient) Post(ctx context.Context, endpoint string, body any, headers map[string]string) ([]byte, int, error) {
	url := c.baseURL + endpoint
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, 0, err
	}
	request, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, 0, err
	}
	c.setHeaders(request, headers)
	return c.doRequest(ctx, request)
}

func (c *RestClient) Put(ctx context.Context, endpoint string, body any, headers map[string]string) ([]byte, int, error) {
	url := c.baseURL + endpoint
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, 0, err
	}
	request, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, 0, err
	}
	c.setHeaders(request, headers)
	return c.doRequest(ctx, request)
}

func (c *RestClient) Delete(ctx context.Context, endpoint string, headers map[string]string) ([]byte, int, error) {
	url := c.baseURL + endpoint
	request, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return nil, 0, err
	}
	c.setHeaders(request, headers)
	return c.doRequest(ctx, request)
}
