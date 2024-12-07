package main

import (
  "context"
  "flag"
  "net/http"

  log "github.com/sirupsen/logrus"
  _ "github.com/ushakovn/boiler/pkg/app"
  "github.com/ushakovn/outfit/internal/app/sender"
  "github.com/ushakovn/outfit/internal/config"
  "github.com/ushakovn/outfit/internal/deps/storage/mongodb"
  tgbot "github.com/ushakovn/outfit/internal/deps/telegram"
  "github.com/ushakovn/outfit/internal/models"
  "github.com/ushakovn/outfit/pkg/logger"
)

var productType models.ProductType

func main() {
  ctx := context.Background()

  logger.Init()

  log.Warn("sender cron app initializing")

  flag.StringVar(&productType, "type", "", "product type")
  flag.Parse()

  mongoClient, err := mongodb.NewClient(ctx,
    mongodb.Config{
      Host: config.Get(ctx, config.MongodbHost).String(),
      Port: config.Get(ctx, config.MongodbPort).String(),
      Authentication: &mongodb.Authentication{
        User:     config.Get(ctx, config.MongodbUser).String(),
        Password: config.Get(ctx, config.MongodbPassword).String(),
      },
    },
    mongodb.Dependencies{
      Client: http.DefaultClient,
    })
  if err != nil {
    log.Fatalf("mongodb.NewClient: %v", err)
  }

  telegramBotClient, err := tgbot.NewBotClient(tgbot.Config{
    Token: config.Get(ctx, config.TelegramToken).String(),
  })
  if err != nil {
    log.Fatalf("tgbot.NewBotClient: %v", err)
  }

  senderCron := sender.NewSenderCron(productType, sender.Dependencies{
    Telegram: telegramBotClient,
    Mongodb:  mongoClient,
  })

  if err = senderCron.Start(ctx); err != nil {
    log.Fatalf("senderCron.Start: %v", err)
  }

  log.Warn("sender cron app terminating")
}
