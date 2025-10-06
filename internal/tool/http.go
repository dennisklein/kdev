package tool

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// fetchHTTPContent performs a GET request and returns the response body content.
// It ensures proper error handling and response body cleanup.
func fetchHTTPContent(ctx context.Context, client *http.Client, url string) (data []byte, err error) {
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
