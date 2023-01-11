package notification

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/spf13/viper"
)

type WebhookRequestBody struct {
	Content string `json:"content"`
	Body    string `json:"body"`
}

func SendWebhook(data string) error {

	webhookUrl := viper.GetString("notification.webhook.url")
	if webhookUrl == "" {
		return fmt.Errorf("webhook url not set")
	}

	body := []byte(fmt.Sprintf(`{"content": "%s", "body": "%s"}`, data, data))

	client := &http.Client{}

	req, err := http.NewRequest("POST", webhookUrl, bytes.NewBuffer(body))

	if err != nil {
		return fmt.Errorf("error creating webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)

	if err != nil {
		return fmt.Errorf("error sending webhook request: %w", err)
	}

	defer resp.Body.Close()

	return nil
}
