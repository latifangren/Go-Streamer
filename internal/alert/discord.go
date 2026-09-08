package alert

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type DiscordClient struct {
	httpClient *http.Client
}

func NewDiscordClient() *DiscordClient {
	return &DiscordClient{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type discordEmbed struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Color       int    `json:"color,omitempty"`
	Timestamp   string `json:"timestamp,omitempty"`
}

type discordWebhookPayload struct {
	Embeds []discordEmbed `json:"embeds"`
}

// SendEmbed mengirim pesan embed ke Discord Webhook URL.
func (c *DiscordClient) SendEmbed(ctx context.Context, webhookURL, title, description string, colorHex int) error {
	if webhookURL == "" {
		return errors.New("discord webhook URL cannot be empty")
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)

	payload := discordWebhookPayload{
		Embeds: []discordEmbed{
			{
				Title:       title,
				Description: description,
				Color:       colorHex,
				Timestamp:   timestamp,
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal discord payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create discord request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("discord request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discord webhook error (status %d): %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	return nil
}
