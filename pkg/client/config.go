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
	NewStreamer(ctx context.Context, message *models.GenericBotMessage, opts *MessageRequestOptions) (BotProviderStreamer, error)
	SendMessage(ctx context.Context, message *models.GenericBotMessage, opts *MessageRequestOptions) (*models.GenericBotReply, error)
	TriggerJSON(ctx context.Context, payload map[string]interface{}) (interface{}, error)
	TriggerForm(ctx context.Context, payload map[string]interface{}, reader io.Reader, filename string, mime *string) (interface{}, error)
	UploadBlob(ctx context.Context, customChannelID string, reader io.Reader, filename string, mime *string) (*models.Blob, error)
	GenerateSandboxEditorOpenUrl(ctx context.Context, sandboxName string) (string, error)
	SandboxFsList(ctx context.Context, sandboxName, path string) (*models.SandboxFsListResult, error)
	SandboxFsRead(ctx context.Context, sandboxName, path string, offsetBytes, limitBytes *int64) ([]byte, *models.SandboxFsReadMeta, error)
	SandboxFsWrite(ctx context.Context, sandboxName, path string, reader io.Reader, filename string, mode *uint32, createOnly bool) (*models.SandboxFsWriteResult, error)
	SandboxHeartbeat(ctx context.Context, sandboxName string) (*models.SandboxHeartbeatResult, error)
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
	Headers           map[string]string
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
