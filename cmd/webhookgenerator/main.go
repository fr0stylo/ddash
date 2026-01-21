package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type payload struct {
	Name        string `json:"name"`
	Environment string `json:"environment"`
	Reference   string `json:"reference"`
}

func main() {
	configPath := flag.String("config", "", "path to YAML config")
	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	interval, err := time.ParseDuration(cfg.Interval)
	if err != nil {
		fmt.Fprintln(os.Stderr, "invalid interval duration:", err)
		os.Exit(1)
	}
	if interval <= 0 {
		fmt.Fprintln(os.Stderr, "interval must be positive")
		os.Exit(1)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	client := &http.Client{Timeout: 10 * time.Second}
	for {
		if err := sendWebhook(client, cfg); err != nil {
			fmt.Fprintln(os.Stderr, "webhook error:", err)
		}
		<-ticker.C
	}
}

func loadConfig(path string) (config, error) {
	if strings.TrimSpace(path) == "" {
		return config{}, fmt.Errorf("config path is required")
	}

	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	if err := v.ReadInConfig(); err != nil {
		return config{}, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg config
	if err := v.Unmarshal(&cfg); err != nil {
		return config{}, fmt.Errorf("failed to decode config: %w", err)
	}

	cfg.BaseURL = strings.TrimSpace(cfg.BaseURL)
	cfg.Token = strings.TrimSpace(cfg.Token)
	cfg.Secret = strings.TrimSpace(cfg.Secret)
	cfg.Service = strings.TrimSpace(cfg.Service)
	cfg.Environment = strings.TrimSpace(cfg.Environment)
	cfg.Interval = strings.TrimSpace(cfg.Interval)

	if cfg.BaseURL == "" || cfg.Token == "" || cfg.Secret == "" || cfg.Service == "" || cfg.Environment == "" {
		return config{}, fmt.Errorf("config must include base_url, token, secret, service, environment")
	}
	if cfg.Interval == "" {
		return config{}, fmt.Errorf("interval must be provided")
	}

	parsed, err := time.ParseDuration(cfg.Interval)
	if err != nil {
		return config{}, fmt.Errorf("invalid interval duration: %w", err)
	}
	if parsed <= 0 {
		return config{}, fmt.Errorf("interval must be positive")
	}

	return cfg, nil
}

func sendWebhook(client *http.Client, cfg config) error {
	ref, err := randomSHA(7)
	if err != nil {
		return fmt.Errorf("failed to generate reference: %w", err)
	}

	body, err := json.Marshal(payload{
		Name:        cfg.Service,
		Environment: cfg.Environment,
		Reference:   ref,
	})
	if err != nil {
		return fmt.Errorf("failed to encode payload: %w", err)
	}

	signature := sign(body, cfg.Secret)
	request, err := http.NewRequestWithContext(context.Background(), http.MethodPost, strings.TrimRight(cfg.BaseURL, "/")+"/webhooks/custom", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}

	request.Header.Set("Authorization", "Bearer "+cfg.Token)
	request.Header.Set("X-Webhook-Signature", signature)
	request.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusMultipleChoices {
		payload, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook failed: %s", strings.TrimSpace(string(payload)))
	}

	fmt.Printf("Webhook status: %s (ref %s)\n", resp.Status, ref)
	return nil
}

func randomSHA(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("invalid length")
	}
	bytesNeeded := (length + 1) / 2
	raw := make([]byte, bytesNeeded)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	hexValue := hex.EncodeToString(raw)
	return hexValue[:length], nil
}

func sign(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
