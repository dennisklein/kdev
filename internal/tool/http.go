package tool

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

// HTTPClient is an interface that both http.Client and retryablehttp.Client implement.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// getRetryableClient creates a configured retryable HTTP client for production use.
func getRetryableClient() *retryablehttp.Client {
	client := retryablehttp.NewClient()
	client.RetryMax = 3
	client.RetryWaitMin = 1 * time.Second
	client.RetryWaitMax = 10 * time.Second
	client.Logger = nil // Disable logging to avoid cluttering output

	return client
}

// fetchHTTPContent performs a GET request with automatic retry logic and returns the response body content.
// It ensures proper error handling and response body cleanup.
func fetchHTTPContent(ctx context.Context, client HTTPClient, url string) (data []byte, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
