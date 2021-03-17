package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type ApiClient struct {
	AssembleUrl func(string) string
	RestClient *RestClient
}

type RestClient struct {
	HttpClient *http.Client
}

type ApiError struct {
	Message string `json:"message"`
}

func (t *RestClient) Post(ctx context.Context, url string, payload interface{}) (*http.Response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return t.HttpClient.Do(req)
}

func decodeError(resp *http.Response) error {
	var body ApiError
	err := json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		return err
	}
	return fmt.Errorf("HTTP %d: %s", resp.StatusCode, body.Message)
}

// V1OpendexdCreate issues a POST request to /api/v1/opendexd/create
func (t *ApiClient) V1OpendexdCreate(ctx context.Context, password string) error {
	url := t.AssembleUrl("/v1/opendexd/create")
	payload := map[string]interface{}{
		"password": password,
	}
	resp, err := t.RestClient.Post(ctx, url, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return decodeError(resp)
	}

	return nil
}

// V1OpendexdRestore issues a POST request to /api/v1/opendexd/restore
func (t *ApiClient) V1OpendexdRestore() {

}

// V1OpendexdUnlock issues a POST request to /api/v1/opendexd/unlock
func (t *ApiClient) V1OpendexdUnlock(ctx context.Context, password string) error {
	url := t.AssembleUrl("/v1/opendexd/unlock")
	payload := map[string]interface{}{
		"password": password,
	}
	resp, err := t.RestClient.Post(ctx, url, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return decodeError(resp)
	}

	return nil
}
