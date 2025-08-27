package restclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewRestClient(t *testing.T) {
	c := NewRestClient("http://test", map[string]string{"x": "y"})
	if c.baseURL != "http://test" {
		t.Fail()
	}
	if c.headers["x"] != "y" {
		t.Fail()
	}
	if c.httpClient == nil {
		t.Fail()
	}
}

func TestDoRequest(t *testing.T) {
	c := &RestClient{httpClient: &http.Client{Transport: RoundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return nil, errors.New("err")
	})}}
	r, _ := http.NewRequest("GET", "http://test", nil)
	b, s, err := c.doRequestOnce(context.Background(), r)
	if err == nil || s != 0 || len(b) != 0 {
		t.Fail()
	}
}

func TestRestClient(t *testing.T) {
	ctx := context.Background()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer ts.Close()
	cases := []struct {
		name     string
		method   string
		baseURL  string
		endpoint string
		body     any
		expectOK bool
	}{
		{"get_ok", http.MethodGet, ts.URL, "/", nil, true},
		{"post_ok", http.MethodPost, ts.URL, "/", map[string]string{"x": "y"}, true},
		{"put_ok", http.MethodPut, ts.URL, "/", map[string]string{"x": "y"}, true},
		{"delete_ok", http.MethodDelete, ts.URL, "/", nil, true},
		{"invalid_url", http.MethodGet, "://bad", "", nil, false},
		{"json_error", http.MethodPost, ts.URL, "/", func() {}, false},
		{"server_closed", http.MethodGet, "", "/", nil, false},
	}
	for _, cse := range cases {
		t.Run(cse.name, func(t *testing.T) {
			var rc *RestClient
			if cse.name == "server_closed" {
				s := httptest.NewServer(nil)
				s.Close()
				rc = NewRestClient(s.URL, nil)
			} else {
				rc = NewRestClient(cse.baseURL, nil)
			}
			var b []byte
			var s int
			var err error
			switch cse.method {
			case http.MethodGet:
				b, s, err = rc.Get(ctx, cse.endpoint, nil)
			case http.MethodPost:
				b, s, err = rc.Post(ctx, cse.endpoint, cse.body, nil)
			case http.MethodPut:
				b, s, err = rc.Put(ctx, cse.endpoint, cse.body, nil)
			case http.MethodDelete:
				b, s, err = rc.Delete(ctx, cse.endpoint, nil)
			}
			if cse.expectOK && (err != nil || s != http.StatusOK || string(b) != "ok") {
				t.Fail()
			}
			if !cse.expectOK && err == nil {
				t.Fail()
			}
		})
	}
}

type RoundTripFunc func(*http.Request) (*http.Response, error)

func (f RoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
