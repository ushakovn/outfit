package sender

import (
  telegram "github.com/go-telegram/bot"
  "github.com/ushakovn/outfit/internal/deps/storage/mongodb"
  "github.com/ushakovn/outfit/internal/models"
)

type Sender struct {
  config Config
  deps   Dependencies
}

type Config struct {
  IsCron      bool
  ProductType models.ProductType
}

type Dependencies struct {
  Telegram *telegram.Bot
  Mongodb  *mongodb.Client
}

func NewSenderCron(typ models.ProductType, deps Dependencies) *Sender {
  return &Sender{
    config: Config{
      IsCron:      true,
      ProductType: typ,
    },
    deps: deps,
  }
}
