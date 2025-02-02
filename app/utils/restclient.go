package utils

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

type RestClient struct {
	baseURL    string
	headers    map[string]string
	httpClient *http.Client
}

func NewRestClient(baseURL string, headers map[string]string) *RestClient {
	return &RestClient{
		baseURL: baseURL,
		headers: headers,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
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

func (c *RestClient) doRequest(request *http.Request) ([]byte, int, error) {
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, 0, err
	}
	defer response.Body.Close()
	var body []byte
	body, err = io.ReadAll(response.Body)
	return body, response.StatusCode, err
}

func (c *RestClient) Get(endpoint string, headers map[string]string) ([]byte, int, error) {
	url := c.baseURL + endpoint
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, 0, err
	}
	c.setHeaders(request, headers)
	return c.doRequest(request)
}

func (c *RestClient) Post(endpoint string, body any, headers map[string]string) ([]byte, int, error) {
	url := c.baseURL + endpoint
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, 0, err
	}
	request, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, 0, err
	}
	c.setHeaders(request, headers)
	return c.doRequest(request)
}

func (c *RestClient) Put(endpoint string, body any, headers map[string]string) ([]byte, int, error) {
	url := c.baseURL + endpoint
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, 0, err
	}
	request, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, 0, err
	}
	c.setHeaders(request, headers)
	return c.doRequest(request)
}

func (c *RestClient) Delete(endpoint string, headers map[string]string) ([]byte, int, error) {
	url := c.baseURL + endpoint
	request, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return nil, 0, err
	}
	c.setHeaders(request, headers)
	return c.doRequest(request)
}
