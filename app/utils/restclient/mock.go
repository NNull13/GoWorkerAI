package restclient

import "github.com/stretchr/testify/mock"

type MockRestClient struct {
	mock.Mock
}

func (m *MockRestClient) Get(endpoint string, headers map[string]string) ([]byte, int, error) {
	args := m.Called(endpoint, headers)
	return args.Get(0).([]byte), args.Get(1).(int), args.Error(2)
}

func (m *MockRestClient) Post(endpoint string, body any, headers map[string]string) ([]byte, int, error) {
	args := m.Called(endpoint, body, headers)
	return args.Get(0).([]byte), args.Get(1).(int), args.Error(2)
}

func (m *MockRestClient) Put(endpoint string, body any, headers map[string]string) ([]byte, int, error) {
	args := m.Called(endpoint, body, headers)
	return args.Get(0).([]byte), args.Get(1).(int), args.Error(2)
}

func (m *MockRestClient) Delete(endpoint string, headers map[string]string) ([]byte, int, error) {
	args := m.Called(endpoint, headers)
	return args.Get(0).([]byte), args.Get(1).(int), args.Error(2)
}
