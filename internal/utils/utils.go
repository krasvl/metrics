package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/go-resty/resty/v2"
)

// GetHash generates a HMAC-SHA256 hash of the given data using the provided key.
func GetHash(key string, data []byte) string {
	if key == "" {
		return ""
	}
	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// WithRestyRetry retries a Resty request with exponential backoff.
func WithRestyRetry(request func() (*resty.Response, error)) (*resty.Response, error) {
	var resp *resty.Response
	var err error
	for _, delay := range []int{0, 1, 3, 5} {
		time.Sleep(time.Duration(delay) * time.Second)
		resp, err = request()
		if resp.StatusCode() != http.StatusServiceUnavailable {
			return resp, err
		}
	}
	return resp, err
}

// WithFileRetry retries opening a file with exponential backoff.
func WithFileRetry(open func() (*os.File, error)) (*os.File, error) {
	var f *os.File
	var err error
	for _, delay := range []int{0, 1, 3, 5} {
		time.Sleep(time.Duration(delay) * time.Second)
		f, err = open()
		if !errors.Is(err, syscall.EBUSY) {
			return f, err
		}
	}
	return f, err
}
