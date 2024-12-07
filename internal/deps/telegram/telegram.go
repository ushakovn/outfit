package telegram

import (
  "fmt"

  tgbot "github.com/go-telegram/bot"
  log "github.com/sirupsen/logrus"
)

type Config struct {
  Token string
}

func NewBotClient(config Config) (*tgbot.Bot, error) {
  bot, err := tgbot.New(config.Token)
  if err != nil {
    return nil, fmt.Errorf("tgbot.New: %w", err)
  }
  log.Info("telegram bot client connection successfully")

  return bot, nil
}
