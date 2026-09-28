package i18n

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// AzureTranslator calls the Azure AI Translator REST API (v3.0). It is selected
// automatically by NewTranslator when AZURE_TRANSLATOR_KEY is set.
//
// Required env:
//   AZURE_TRANSLATOR_KEY      — resource key
// Optional env:
//   AZURE_TRANSLATOR_REGION   — resource region (e.g. "uaenorth"); required for
//                               regional (non-global) resources
//   AZURE_TRANSLATOR_ENDPOINT — defaults to the global endpoint
type AzureTranslator struct {
	key      string
	region   string
	endpoint string
	client   *http.Client
}

const defaultAzureEndpoint = "https://api.cognitive.microsofttranslator.com"

func newAzureTranslator(key string) *AzureTranslator {
	endpoint := os.Getenv("AZURE_TRANSLATOR_ENDPOINT")
	if endpoint == "" {
		endpoint = defaultAzureEndpoint
	}
	return &AzureTranslator{
		key:      key,
		region:   os.Getenv("AZURE_TRANSLATOR_REGION"),
		endpoint: strings.TrimRight(endpoint, "/"),
		client:   &http.Client{Timeout: 8 * time.Second},
	}
}

// Translate sends one string to Azure and returns the translated text.
func (a *AzureTranslator) Translate(ctx context.Context, text, from, to string) (string, error) {
	if strings.TrimSpace(text) == "" {
		return text, nil
	}
	url := fmt.Sprintf("%s/translate?api-version=3.0&from=%s&to=%s", a.endpoint, from, to)
	body, err := json.Marshal([]map[string]string{{"Text": text}})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Ocp-Apim-Subscription-Key", a.key)
	if a.region != "" {
		req.Header.Set("Ocp-Apim-Subscription-Region", a.region)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("azure translator: status %d", resp.StatusCode)
	}

	// Response shape: [{"translations":[{"text":"...","to":"ar"}]}]
	var out []struct {
		Translations []struct {
			Text string `json:"text"`
			To   string `json:"to"`
		} `json:"translations"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if len(out) == 0 || len(out[0].Translations) == 0 {
		return "", errors.New("azure translator: empty response")
	}
	return out[0].Translations[0].Text, nil
}
