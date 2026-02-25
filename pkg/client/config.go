package client

import (
	"context"
	"io"
	"net/http"
	"time"

	"go.asgard-ai.com/asgard-sdk-go/pkg/models"
)

const defaultHTTPTimeout = 300 * time.Second

// Client defines the interface for interacting with Edge Server BotProvider APIs.
type Client interface {
	NewStreamer(ctx context.Context, message *models.GenericBotMessage) (BotProviderStreamer, error)
	SendMessage(ctx context.Context, message *models.GenericBotMessage, isDebug bool) (*models.GenericBotReply, error)
	UploadBlob(ctx context.Context, customChannelID string, reader io.Reader, filename string, mime *string) (*models.Blob, error)
}

// BotProviderClient is a typed client for Edge Server BotProvider endpoints.
type BotProviderClient struct {
	config *BotProviderConfig
}

// BotProviderConfig holds the configuration for connecting to the bot provider
type BotProviderConfig struct {
	HTTPClient        *http.Client
	EdgeServerHost    string
	Namespace         string
	BotProviderName   string
	BotProviderApiKey string
}

// NewBotProviderClient creates a BotProvider API client with default HTTP settings.
func NewBotProviderClient(edgeServerHost, namespace, botProviderName, botProviderAPIKey string) Client {
	return NewBotProviderClientWithConfig(&BotProviderConfig{
		HTTPClient:        &http.Client{Timeout: defaultHTTPTimeout},
		EdgeServerHost:    edgeServerHost,
		Namespace:         namespace,
		BotProviderName:   botProviderName,
		BotProviderApiKey: botProviderAPIKey,
	})
}

// NewBotProviderClientWithConfig creates a BotProvider API client from config.
func NewBotProviderClientWithConfig(config *BotProviderConfig) Client {
	if config == nil {
		config = &BotProviderConfig{}
	}

	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{Timeout: defaultHTTPTimeout}
	}

	return &BotProviderClient{config: config}
}
