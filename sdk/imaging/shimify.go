// Package shimify provides a client for the Shimify API.
package shimify

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type ImageClient struct {
	BaseURL string
	APIKey  string
	client  *http.Client
}

type ResizeRequest struct {
	Image  string `json:"image"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Format string `json:"format,omitempty"`
}

type ResizeResponse struct {
	Success bool   `json:"success"`
	ID      string `json:"id"`
}

type ConvertRequest struct {
	Image   string `json:"image"`
	Format  string `json:"format"`
	Quality int    `json:"quality,omitempty"`
}

type ProcessResponse struct {
	Success bool   `json:"success"`
	ID      string `json:"id"`
	Data    string `json:"data,omitempty"` // base64 result
}

func New(baseURL, apiKey string) *ImageClient {
	return &ImageClient{
		BaseURL: baseURL,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *ImageClient) Resize(imageData []byte, width, height int, format string) ([]byte, error) {
	encoded := base64.StdEncoding.EncodeToString(imageData)

	req := ResizeRequest{
		Image:  encoded,
		Width:  width,
		Height: height,
		Format: format,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", c.BaseURL+"/v1/images/resize", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Key", c.APIKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s", string(respBody))
	}

	var result ResizeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return c.GetImage(result.ID)
}

func (c *ImageClient) GetImage(id string) ([]byte, error) {
	httpReq, err := http.NewRequest("GET", c.BaseURL+"/v1/images/"+id, nil)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("X-API-Key", c.APIKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("image not found")
	}

	return io.ReadAll(resp.Body)
}

func (c *ImageClient) Convert(imageData []byte, format string, quality int) ([]byte, error) {
	encoded := base64.StdEncoding.EncodeToString(imageData)

	req := ConvertRequest{
		Image:   encoded,
		Format:  format,
		Quality: quality,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", c.BaseURL+"/v1/images/convert", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Key", c.APIKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s", string(respBody))
	}

	var result ResizeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return c.GetImage(result.ID)
}

func (c *ImageClient) UploadAndResize(imageURL string, width, height int, format string) ([]byte, error) {
	resp, err := http.Get(imageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read image: %w", err)
	}

	return c.Resize(imageData, width, height, format)
}
