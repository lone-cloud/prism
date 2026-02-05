package telegram

import (
	"context"
	"fmt"

	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegoutil"
)

type Client struct {
	bot *telego.Bot
}

func NewClient(token string) (*Client, error) {
	if token == "" {
		return nil, nil
	}

	bot, err := telego.NewBot(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	return &Client{bot: bot}, nil
}

func (c *Client) GetMe() (*telego.User, error) {
	if c == nil || c.bot == nil {
		return nil, fmt.Errorf("telegram client not initialized")
	}
	return c.bot.GetMe(context.Background())
}

func (c *Client) SendMessage(chatID int64, text string) error {
	if c == nil || c.bot == nil {
		return fmt.Errorf("telegram client not initialized")
	}

	msg := telegoutil.Message(
		telegoutil.ID(chatID),
		text,
	).WithParseMode(telego.ModeHTML)

	_, err := c.bot.SendMessage(context.Background(), msg)
	return err
}

func (c *Client) IsAvailable() bool {
	if c == nil || c.bot == nil {
		return false
	}
	_, err := c.bot.GetMe(context.Background())
	return err == nil
}
