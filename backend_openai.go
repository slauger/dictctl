package main

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func transcribeOpenAI(audioFile, language, model, apiKey string) (string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add audio file
	f, err := os.Open(audioFile)
	if err != nil {
		return "", err
	}
	defer f.Close()

	part, err := writer.CreateFormFile("file", filepath.Base(audioFile))
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, f); err != nil {
		return "", err
	}

	writer.WriteField("model", model)
	writer.WriteField("language", language)
	writer.WriteField("response_format", "text")
	writer.Close()

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/audio/transcriptions", &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("openai request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("openai API error (%d): %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	return strings.TrimSpace(string(respBody)), nil
}
