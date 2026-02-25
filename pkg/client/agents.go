package client

import (
	"context"
	"io"

	"go.asgard-ai.com/asgard-sdk-go/pkg/models"
)

// BotAgent handles conversational APIs (message / sse / blob).
type BotAgent interface {
	NewStreamer(ctx context.Context, message *models.GenericBotMessage) (BotProviderStreamer, error)
	SendMessage(ctx context.Context, message *models.GenericBotMessage, isDebug bool) (*models.GenericBotReply, error)
	UploadBlob(ctx context.Context, customChannelID string, reader io.Reader, filename string, mime *string) (*models.Blob, error)
}

// FunctionAgent handles trigger APIs (json / form).
type FunctionAgent interface {
	TriggerJSON(ctx context.Context, payload map[string]interface{}) (interface{}, error)
	TriggerForm(ctx context.Context, payload map[string]interface{}, reader io.Reader, filename string, mime *string) (interface{}, error)
}

type botAgent struct {
	client Client
}

type functionAgent struct {
	client Client
}

// NewBotAgent creates a BotAgent that hides the underlying Client.
func NewBotAgent(edgeServerHost, namespace, botProviderName, botProviderAPIKey string) BotAgent {
	return NewBotAgentWithConfig(&BotProviderConfig{
		EdgeServerHost:    edgeServerHost,
		Namespace:         namespace,
		BotProviderName:   botProviderName,
		BotProviderApiKey: botProviderAPIKey,
	})
}

// NewBotAgentWithConfig creates a BotAgent from config.
func NewBotAgentWithConfig(config *BotProviderConfig) BotAgent {
	return &botAgent{client: NewBotProviderClientWithConfig(config)}
}

// NewFunctionAgent creates a FunctionAgent that hides the underlying Client.
func NewFunctionAgent(edgeServerHost, namespace, botProviderName, botProviderAPIKey string) FunctionAgent {
	return NewFunctionAgentWithConfig(&BotProviderConfig{
		EdgeServerHost:    edgeServerHost,
		Namespace:         namespace,
		BotProviderName:   botProviderName,
		BotProviderApiKey: botProviderAPIKey,
	})
}

// NewFunctionAgentWithConfig creates a FunctionAgent from config.
func NewFunctionAgentWithConfig(config *BotProviderConfig) FunctionAgent {
	return &functionAgent{client: NewBotProviderClientWithConfig(config)}
}

func (a *botAgent) NewStreamer(ctx context.Context, message *models.GenericBotMessage) (BotProviderStreamer, error) {
	return a.client.NewStreamer(ctx, message)
}

func (a *botAgent) SendMessage(ctx context.Context, message *models.GenericBotMessage, isDebug bool) (*models.GenericBotReply, error) {
	return a.client.SendMessage(ctx, message, isDebug)
}

func (a *botAgent) UploadBlob(ctx context.Context, customChannelID string, reader io.Reader, filename string, mime *string) (*models.Blob, error) {
	return a.client.UploadBlob(ctx, customChannelID, reader, filename, mime)
}

func (a *functionAgent) TriggerJSON(ctx context.Context, payload map[string]interface{}) (interface{}, error) {
	return a.client.TriggerJSON(ctx, payload)
}

func (a *functionAgent) TriggerForm(ctx context.Context, payload map[string]interface{}, reader io.Reader, filename string, mime *string) (interface{}, error) {
	return a.client.TriggerForm(ctx, payload, reader, filename, mime)
}
