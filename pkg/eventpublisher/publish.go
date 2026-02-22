package eventpublisher

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func (c Client) Publish(ctx context.Context, event Event) (string, error) {
	body, resolvedType, err := BuildEventBody(event)
	if err != nil {
		return "", err
	}
	if err := c.publishBody(ctx, body); err != nil {
		return "", err
	}
	return resolvedType, nil
}

func (c Client) publishBody(ctx context.Context, body []byte) error {
	endpoint := strings.TrimSpace(c.Endpoint)
	token := strings.TrimSpace(c.Token)
	secret := strings.TrimSpace(c.Secret)
	if endpoint == "" || token == "" || secret == "" {
		return fmt.Errorf("endpoint/token/secret are required")
	}

	httpClient := c.HTTPClient
	if httpClient == nil {
		timeout := c.Timeout
		if timeout <= 0 {
			timeout = 10 * time.Second
		}
		httpClient = &http.Client{Timeout: timeout}
	}

	requestURL := strings.TrimRight(endpoint, "/") + "/webhooks/cdevents"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Webhook-Signature", sign(body, secret))
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusMultipleChoices {
		payload, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook rejected: status=%s body=%s", resp.Status, strings.TrimSpace(string(payload)))
	}
	return nil
}

func sign(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
