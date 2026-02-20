package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	mrand "math/rand"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	cdeventsapi "github.com/cdevents/sdk-go/pkg/api"
	cdeventsv05 "github.com/cdevents/sdk-go/pkg/api/v05"
	"github.com/spf13/viper"
)

var nonPurlChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

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
	rng := mrand.New(mrand.NewSource(time.Now().UnixNano()))

	client := &http.Client{Timeout: 10 * time.Second}
	for {
		if err := sendWebhook(client, cfg, rng); err != nil {
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
	for i := range cfg.Services {
		cfg.Services[i] = strings.TrimSpace(cfg.Services[i])
	}
	cfg.Environment = strings.TrimSpace(cfg.Environment)
	for i := range cfg.Environments {
		cfg.Environments[i] = strings.TrimSpace(cfg.Environments[i])
	}
	cfg.Interval = strings.TrimSpace(cfg.Interval)

	if cfg.BaseURL == "" || cfg.Token == "" || cfg.Secret == "" {
		return config{}, fmt.Errorf("config must include base_url, token, and secret")
	}
	if len(nonEmpty(cfg.Services)) == 0 && cfg.Service == "" {
		return config{}, fmt.Errorf("config must include service or services")
	}
	if len(nonEmpty(cfg.Environments)) == 0 && cfg.Environment == "" {
		return config{}, fmt.Errorf("config must include environment or environments")
	}
	if cfg.Interval == "" {
		return config{}, fmt.Errorf("interval must be provided")
	}

	if len(nonEmpty(cfg.Services)) == 0 {
		cfg.Services = []string{cfg.Service}
	} else {
		cfg.Services = nonEmpty(cfg.Services)
	}

	if len(nonEmpty(cfg.Environments)) == 0 {
		cfg.Environments = []string{cfg.Environment}
	} else {
		cfg.Environments = nonEmpty(cfg.Environments)
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

func sendWebhook(client *http.Client, cfg config, rng *mrand.Rand) error {
	service := cfg.Services[rng.Intn(len(cfg.Services))]
	environment := cfg.Environments[rng.Intn(len(cfg.Environments))]

	ref, err := randomSHA(7)
	if err != nil {
		return fmt.Errorf("failed to generate reference: %w", err)
	}

	event, err := cdeventsv05.NewServiceDeployedEvent()
	if err != nil {
		return fmt.Errorf("failed to create cdevent: %w", err)
	}

	event.SetSource(strings.TrimRight(cfg.BaseURL, "/"))
	event.SetSubjectId("service/" + service)
	event.SetSubjectEnvironment(&cdeventsapi.Reference{Id: environment})
	event.SetSubjectArtifactId(fmt.Sprintf("pkg:generic/%s@%s", sanitizePurlName(service), ref))

	body, err := cdeventsapi.AsJsonBytes(event)
	if err != nil {
		return fmt.Errorf("failed to encode cdevent: %w", err)
	}

	signature := sign(body, cfg.Secret)
	request, err := http.NewRequestWithContext(context.Background(), http.MethodPost, strings.TrimRight(cfg.BaseURL, "/")+"/webhooks/cdevents", bytes.NewReader(body))
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

	fmt.Printf("Webhook status: %s (service=%s env=%s ref=%s)\n", resp.Status, service, environment, ref)
	return nil
}

func sanitizePurlName(value string) string {
	name := nonPurlChars.ReplaceAllString(strings.TrimSpace(value), "-")
	name = strings.Trim(name, "-.")
	if name == "" {
		return "service"
	}
	return name
}

func randomSHA(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("invalid length")
	}
	bytesNeeded := (length + 1) / 2
	raw := make([]byte, bytesNeeded)
	if _, err := crand.Read(raw); err != nil {
		return "", err
	}
	hexValue := hex.EncodeToString(raw)
	return hexValue[:length], nil
}

func nonEmpty(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		result = append(result, value)
	}
	return result
}

func sign(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
