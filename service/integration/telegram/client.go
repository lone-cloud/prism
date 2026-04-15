package telegram

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

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

func (c *Client) SendPhoto(chatID int64, imageURL, caption string) error {
	if c == nil || c.bot == nil {
		return fmt.Errorf("telegram client not initialized")
	}

	resp, err := http.Get(imageURL) //nolint:noctx
	if err != nil {
		return fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	tmp, err := os.CreateTemp("", "prism-telegram-*.jpg")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmp.Name())

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to write image: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	f, err := os.Open(tmp.Name())
	if err != nil {
		return fmt.Errorf("failed to open temp file: %w", err)
	}
	defer f.Close()

	params := &telego.SendPhotoParams{
		ChatID:    telegoutil.ID(chatID),
		Photo:     telego.InputFile{File: f},
		Caption:   caption,
		ParseMode: telego.ModeHTML,
	}

	_, err = c.bot.SendPhoto(context.Background(), params)
	return err
}

func (c *Client) IsAvailable() bool {
	if c == nil || c.bot == nil {
		return false
	}
	_, err := c.bot.GetMe(context.Background())
	return err == nil
}
