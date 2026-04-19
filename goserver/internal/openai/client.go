package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"goserver/internal/config"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient(cfg config.OpenAIConfig) *Client {
	return &Client{
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:  cfg.APIKey,
		httpClient: &http.Client{
			Timeout: 2 * time.Minute,
		},
	}
}

func (client *Client) UploadBatchInputFile(ctx context.Context, filename string, contents []byte) (*FileObject, error) {
	fields := map[string]string{
		"purpose": "batch",
	}

	var file FileObject
	if err := client.doMultipart(ctx, http.MethodPost, "/v1/files", fields, "file", filename, "application/jsonl", bytes.NewReader(contents), &file); err != nil {
		return nil, err
	}

	return &file, nil
}

func (client *Client) UploadUserFile(ctx context.Context, filename, contentType string, reader io.Reader) (*FileObject, error) {
	fields := map[string]string{
		"purpose": "user_data",
	}

	var file FileObject
	if err := client.doMultipart(ctx, http.MethodPost, "/v1/files", fields, "file", filepath.Base(filename), contentType, reader, &file); err != nil {
		return nil, err
	}

	return &file, nil
}

func (client *Client) GetFile(ctx context.Context, fileID string) (*FileObject, error) {
	var file FileObject
	if err := client.doJSON(ctx, http.MethodGet, "/v1/files/"+fileID, nil, &file); err != nil {
		return nil, err
	}

	return &file, nil
}

func (client *Client) WaitForFileProcessing(ctx context.Context, fileID string) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	timeout := time.NewTimer(2 * time.Minute)
	defer timeout.Stop()

	for {
		file, err := client.GetFile(ctx, fileID)
		if err != nil {
			return err
		}

		switch strings.ToLower(strings.TrimSpace(file.Status)) {
		case "", "processed", "ready":
			return nil
		case "error", "failed", "cancelled":
			return fmt.Errorf("file %s failed to process with status %q", fileID, file.Status)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout.C:
			return fmt.Errorf("timed out waiting for file %s to process", fileID)
		case <-ticker.C:
		}
	}
}

func (client *Client) DeleteFile(ctx context.Context, fileID string) error {
	return client.doJSON(ctx, http.MethodDelete, "/v1/files/"+fileID, nil, nil)
}

func (client *Client) CreateBatch(ctx context.Context, inputFileID, endpoint, completionWindow string, metadata map[string]string) (*Batch, error) {
	requestBody := map[string]any{
		"input_file_id":     inputFileID,
		"endpoint":          endpoint,
		"completion_window": completionWindow,
		"metadata":          metadata,
	}

	var batch Batch
	if err := client.doJSON(ctx, http.MethodPost, "/v1/batches", requestBody, &batch); err != nil {
		return nil, err
	}

	return &batch, nil
}

func (client *Client) RetrieveBatch(ctx context.Context, batchID string) (*Batch, error) {
	var batch Batch
	if err := client.doJSON(ctx, http.MethodGet, "/v1/batches/"+batchID, nil, &batch); err != nil {
		return nil, err
	}

	return &batch, nil
}

func (client *Client) LoadFileContent(ctx context.Context, fileID string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, client.baseURL+"/v1/files/"+fileID+"/content", nil)
	if err != nil {
		return nil, err
	}

	request.Header.Set("Authorization", "Bearer "+client.apiKey)
	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if err := assertSuccess(response); err != nil {
		return nil, err
	}

	return io.ReadAll(response.Body)
}

func (client *Client) doJSON(ctx context.Context, method, path string, requestBody any, out any) error {
	var body io.Reader
	if requestBody != nil {
		payload, err := json.Marshal(requestBody)
		if err != nil {
			return err
		}
		body = bytes.NewReader(payload)
	}

	request, err := http.NewRequestWithContext(ctx, method, client.baseURL+path, body)
	if err != nil {
		return err
	}

	request.Header.Set("Authorization", "Bearer "+client.apiKey)
	if requestBody != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if err := assertSuccess(response); err != nil {
		return err
	}

	if out == nil {
		io.Copy(io.Discard, response.Body)
		return nil
	}

	return json.NewDecoder(response.Body).Decode(out)
}

func (client *Client) doMultipart(
	ctx context.Context,
	method, path string,
	fields map[string]string,
	fileField, fileName, contentType string,
	reader io.Reader,
	out any,
) error {
	var payload bytes.Buffer
	writer := multipart.NewWriter(&payload)

	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			return err
		}
	}

	part, err := writer.CreateFormFile(fileField, fileName)
	if err != nil {
		return err
	}

	if _, err := io.Copy(part, reader); err != nil {
		return err
	}

	if err := writer.Close(); err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, method, client.baseURL+path, &payload)
	if err != nil {
		return err
	}

	request.Header.Set("Authorization", "Bearer "+client.apiKey)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if contentType != "" {
		request.Header.Set("X-Upload-Content-Type", contentType)
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if err := assertSuccess(response); err != nil {
		return err
	}

	if out == nil {
		io.Copy(io.Discard, response.Body)
		return nil
	}

	return json.NewDecoder(response.Body).Decode(out)
}

func assertSuccess(response *http.Response) error {
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return nil
	}

	body, _ := io.ReadAll(response.Body)
	var payload apiErrorResponse
	if err := json.Unmarshal(body, &payload); err == nil && strings.TrimSpace(payload.Error.Message) != "" {
		return fmt.Errorf(payload.Error.Message)
	}

	if text := strings.TrimSpace(string(body)); text != "" {
		return fmt.Errorf("openai request failed with status %d: %s", response.StatusCode, text)
	}

	return fmt.Errorf("openai request failed with status %d", response.StatusCode)
}
