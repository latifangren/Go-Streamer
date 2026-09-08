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

type TelegramClient struct {
	httpClient *http.Client
	baseURL    string
}

func NewTelegramClient(baseURL ...string) *TelegramClient {
	base := "https://api.telegram.org"
	if len(baseURL) > 0 && baseURL[0] != "" {
		base = strings.TrimRight(baseURL[0], "/")
	}

	return &TelegramClient{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: base,
	}
}

type telegramPayload struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

type telegramResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
}

// SendMessage mengirim pesan teks ke Telegram Chat ID via Bot Token.
func (c *TelegramClient) SendMessage(ctx context.Context, botToken, chatID, text string) error {
	if botToken == "" {
		return errors.New("telegram bot token cannot be empty")
	}
	if chatID == "" {
		return errors.New("telegram chat id cannot be empty")
	}
	if text == "" {
		return errors.New("telegram message text cannot be empty")
	}

	url := fmt.Sprintf("%s/bot%s/sendMessage", c.baseURL, botToken)

	payload := telegramPayload{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "Markdown",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal telegram payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create telegram request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("telegram request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("telegram api error (status %d): %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var tgResp telegramResponse
	if err := json.Unmarshal(respBody, &tgResp); err == nil && !tgResp.OK {
		return fmt.Errorf("telegram api returned not ok: %s", tgResp.Description)
	}

	return nil
}
