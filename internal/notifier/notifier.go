package notifier

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hnrobert/feishu-github-tracker/internal/config"
	"github.com/hnrobert/feishu-github-tracker/internal/logger"
)

// Notifier handles sending notifications to Feishu webhooks
type Notifier struct {
	bots   map[string]string
	client *http.Client
}

const maxWebhookResponseBytes int64 = 1 << 20

// New creates a new Notifier
func New(botsConfig config.FeishuBotsConfig) *Notifier {
	bots := make(map[string]string)
	for _, bot := range botsConfig.FeishuBots {
		bots[bot.Alias] = bot.URL
	}

	return &Notifier{
		bots: bots,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Send sends a notification to the specified targets
func (n *Notifier) Send(targets []string, payload map[string]any) error {
	var errs []string

	for _, target := range targets {
		url := n.resolveURL(target)
		if url == "" {
			logger.Warn("Failed to resolve target: %s", target)
			continue
		}

		if err := n.sendToWebhook(url, payload); err != nil {
			logger.Error("Failed to send notification to target %s: %v", targetLabel(target), err)
			errs = append(errs, err.Error())
		} else {
			logger.Info("Successfully sent notification to %s", targetLabel(target))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to send to some targets: %s", strings.Join(errs, "; "))
	}

	return nil
}

func (n *Notifier) resolveURL(target string) string {
	// First check if it's an alias
	if url, exists := n.bots[target]; exists {
		return url
	}

	// Check if it's already a URL
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		return target
	}

	return ""
}

func (n *Notifier) sendToWebhook(url string, payload map[string]any) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	logger.Debug("Sending %d-byte webhook payload", len(jsonData))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return errors.New("failed to create webhook request")
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return errors.New("failed to send webhook request")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxWebhookResponseBytes+1))
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}
	if int64(len(body)) > maxWebhookResponseBytes {
		return fmt.Errorf("webhook response exceeds %d bytes", maxWebhookResponseBytes)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("received non-2xx status code %d", resp.StatusCode)
	}

	logger.Debug("Webhook response was %d bytes", len(body))
	return nil
}

func targetLabel(target string) string {
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		return "direct webhook"
	}
	return target
}
