package client

import "net/http"

// BotProviderConfig holds the configuration for connecting to the bot provider
type BotProviderConfig struct {
	HTTPClient        *http.Client
	EdgeServerHost    string
	Namespace         string
	BotProviderName   string
	BotProviderApiKey string
}
