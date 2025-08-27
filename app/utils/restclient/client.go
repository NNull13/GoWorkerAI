package restclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"time"
)

type retryPolicy struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
	RetryOn    func(status int, err error) bool
}

type RestClient struct {
	baseURL    string
	headers    map[string]string
	httpClient *http.Client
	retry      *retryPolicy
}

func NewRestClient(baseURL string, headers map[string]string) *RestClient {
	return &RestClient{
		baseURL: baseURL,
		headers: headers,
		httpClient: &http.Client{
			Timeout: 90 * time.Second,
		},
		retry: &retryPolicy{
			MaxRetries: 3,
			BaseDelay:  0,
			MaxDelay:   5 * time.Second,
			RetryOn:    defaultRetryOn,
		},
	}
}

func (c *RestClient) setHeaders(req *http.Request, headers map[string]string) {
	req.Header.Set("Content-Type", "application/json")
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
}

func (c *RestClient) doRequestWithRetry(ctx context.Context, req *http.Request) ([]byte, int, error) {
	var lastErr error
	var status int

	for attempt := 0; attempt <= c.retry.MaxRetries; attempt++ {
		if attempt > 0 && req.GetBody != nil {
			rc, gerr := req.GetBody()
			if gerr != nil {
				lastErr = gerr
				break
			}
			req.Body = rc
		}

		respBody, s, err := c.doRequestOnce(ctx, req)
		if err == nil && s >= 200 && s < 300 {
			return respBody, s, nil
		}

		lastErr, status = err, s
		if err != nil {
			log.Printf("[RestClient] ⚠️ attempt=%d url=%s status=%d error=%v", attempt, req.URL.String(), status, err)
		} else {
			log.Printf("[RestClient] ⚠️ attempt=%d url=%s status=%d body-error", attempt, req.URL.String(), status)
		}

		if !c.retry.RetryOn(s, err) || attempt == c.retry.MaxRetries {
			log.Printf("[RestClient] ❌ giving up after attempt=%d url=%s status=%d lastErr=%v", attempt, req.URL.String(), status, lastErr)
			if err == nil && s >= 200 && s < 300 {
				return respBody, s, nil
			}
			if err == nil {
				return respBody, s, errors.New(http.StatusText(s))
			}
			return nil, s, lastErr
		}

		sleep := backoffWithJitter(c.retry.BaseDelay, attempt, c.retry.MaxDelay)
		log.Printf("[RestClient] ⏳ retrying attempt=%d url=%s status=%d after=%s", attempt, req.URL.String(), status, sleep)

		select {
		case <-ctx.Done():
			log.Printf("[RestClient] 🚨 context cancelled url=%s error=%v", req.URL.String(), ctx.Err())
			return nil, 0, ctx.Err()
		case <-time.After(sleep):
		}
	}
	return nil, status, lastErr
}

func (c *RestClient) doRequestOnce(ctx context.Context, req *http.Request) ([]byte, int, error) {
	resp, err := c.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, rErr := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if rErr == nil {
			rErr = errors.New(string(body))
		}
	}
	return body, resp.StatusCode, rErr
}

func (c *RestClient) Get(ctx context.Context, endpoint string, headers map[string]string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+endpoint, nil)
	if err != nil {
		return nil, 0, err
	}
	c.setHeaders(req, headers)
	return c.doRequestWithRetry(ctx, req)
}

func (c *RestClient) Post(ctx context.Context, endpoint string, body any, headers map[string]string) ([]byte, int, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, 0, err
	}
	c.setHeaders(req, headers)
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(jsonBody)), nil
	}
	return c.doRequestWithRetry(ctx, req)
}

func (c *RestClient) Put(ctx context.Context, endpoint string, body any, headers map[string]string) ([]byte, int, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, 0, err
	}
	c.setHeaders(req, headers)
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(jsonBody)), nil
	}
	return c.doRequestWithRetry(ctx, req)
}

func (c *RestClient) Delete(ctx context.Context, endpoint string, headers map[string]string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+endpoint, nil)
	if err != nil {
		return nil, 0, err
	}
	c.setHeaders(req, headers)
	return c.doRequestWithRetry(ctx, req)
}

func defaultRetryOn(status int, err error) bool {
	if err != nil {
		return true
	}
	if status == 0 {
		return true
	}
	if status == 429 {
		return true
	}
	return false
}

func backoffWithJitter(base time.Duration, attempt int, max time.Duration) time.Duration {
	d := time.Duration(float64(base) * math.Pow(2, float64(attempt)))
	if d > max {
		d = max
	}
	j := time.Duration(rand.Float64() * float64(d) * 0.4) // 0–40% jitter
	return d - (j / 2) + j
}
