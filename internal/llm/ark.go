package llm

import (
	"context"
	"errors"
	"strings"

	"backend/internal/config"

	"github.com/volcengine/volcengine-go-sdk/service/arkruntime"
	arkmodel "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

type Client struct {
	ark   *arkruntime.Client
	model string
}

var global *Client

func Init(cfg config.LLMConfig) error {
	if cfg.Model == "" {
		return errors.New("llm model is required")
	}

	opts := make([]arkruntime.ConfigOption, 0, 2)
	if cfg.BaseURL != "" {
		opts = append(opts, arkruntime.WithBaseUrl(cfg.BaseURL))
	}
	if cfg.Region != "" {
		opts = append(opts, arkruntime.WithRegion(cfg.Region))
	}

	var arkClient *arkruntime.Client
	if cfg.APIKey != "" {
		arkClient = arkruntime.NewClientWithApiKey(cfg.APIKey, opts...)
	} else if cfg.AK != "" && cfg.SK != "" {
		arkClient = arkruntime.NewClientWithAkSk(cfg.AK, cfg.SK, opts...)
	} else {
		return errors.New("llm credentials missing: api_key or ak/sk required")
	}

	global = &Client{
		ark:   arkClient,
		model: cfg.Model,
	}
	return nil
}

func Get() *Client {
	return global
}

func (c *Client) Model() string {
	if c == nil {
		return ""
	}
	return c.model
}

func (c *Client) ChatCompletion(ctx context.Context, messages []*arkmodel.ChatCompletionMessage) (string, arkmodel.Usage, error) {
	req := arkmodel.ChatCompletionRequest{
		Model:    c.model,
		Messages: messages,
	}

	resp, err := c.ark.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", arkmodel.Usage{}, err
	}
	if len(resp.Choices) == 0 || resp.Choices[0] == nil {
		return "", arkmodel.Usage{}, errors.New("empty llm response")
	}
	return extractContent(resp.Choices[0].Message.Content), resp.Usage, nil
}

func extractContent(content *arkmodel.ChatCompletionMessageContent) string {
	if content == nil {
		return ""
	}
	if content.StringValue != nil {
		return *content.StringValue
	}
	if content.ListValue != nil {
		var b strings.Builder
		for _, part := range content.ListValue {
			if part == nil {
				continue
			}
			if part.Type == arkmodel.ChatCompletionMessageContentPartTypeText || part.Type == "" {
				b.WriteString(part.Text)
			}
		}
		return b.String()
	}
	return ""
}
