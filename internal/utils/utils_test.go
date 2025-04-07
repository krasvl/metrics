package utils

import (
	"errors"
	"net/http"
	"os"
	"syscall"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
)

func TestGetHash(t *testing.T) {
	key := "testkey"
	data := []byte("testdata")
	expectedHash := GetHash(key, data)

	assert.NotEmpty(t, expectedHash, "Hash should not be empty")
	assert.Equal(t, GetHash("", data), "", "Hash should be empty when key is empty")
}

func TestWithRestyRetry(t *testing.T) {
	attempts := 0
	mockRequest := func() (*resty.Response, error) {
		attempts++
		if attempts < 3 {
			return &resty.Response{RawResponse: &http.Response{StatusCode: http.StatusServiceUnavailable}}, nil
		}
		return &resty.Response{RawResponse: &http.Response{StatusCode: http.StatusOK}}, nil
	}

	resp, err := WithRestyRetry(mockRequest)
	assert.NoError(t, err, "Error should be nil")
	assert.Equal(t, http.StatusOK, resp.StatusCode(), "Response status should be OK")
	assert.Equal(t, 3, attempts, "Should retry 3 times before success")
}

func TestWithFileRetry(t *testing.T) {
	attempts := 0
	mockOpen := func() (*os.File, error) {
		attempts++
		if attempts < 3 {
			return nil, syscall.EBUSY
		}
		return nil, errors.New("unexpected nil value")
	}

	file, err := WithFileRetry(mockOpen)
	assert.Error(t, err, "Error should not be nil after retries are exhausted")
	assert.EqualError(t, err, "unexpected nil value", "Error message should match the final error from mockOpen")
	assert.Nil(t, file, "File should be nil as mockOpen returns nil")
	assert.Equal(t, 3, attempts, "Should retry 4 times before giving up")
}
