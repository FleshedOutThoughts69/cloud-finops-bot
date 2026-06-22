// internal/slack/client.go

package slack

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "time"

)

// Message represents a Slack message payload.
type Message struct {
    Text        string `json:"text"`
    Channel     string `json:"channel,omitempty"`
    Username    string `json:"username,omitempty"`
    IconEmoji   string `json:"icon_emoji,omitempty"`
}

// Send sends a Slack notification via webhook.
func Send(webhookURL, channel, text string) error {
    if webhookURL == "" {
        return fmt.Errorf("webhook URL is empty")
    }

    msg := Message{
        Text:      text,
        Channel:   channel,
        Username:  "FinOps Bot",
        IconEmoji: ":robot_face:",
    }

    payload, err := json.Marshal(msg)
    if err != nil {
        return fmt.Errorf("failed to marshal Slack message: %w", err)
    }

    client := &http.Client{Timeout: 10 * time.Second}
    resp, err := client.Post(webhookURL, "application/json", bytes.NewReader(payload))
    if err != nil {
        return fmt.Errorf("failed to send Slack message: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("Slack returned non-200 status: %d", resp.StatusCode)
    }

    return nil
}