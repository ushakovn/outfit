package main

import (
  "context"
  "flag"
  "net/http"

  "github.com/go-telegram/bot"
  log "github.com/sirupsen/logrus"
  _ "github.com/ushakovn/boiler/pkg/app"
  "github.com/ushakovn/outfit/internal/app/sender"
  "github.com/ushakovn/outfit/internal/config"
  "github.com/ushakovn/outfit/internal/deps/storage/mongodb"
  "github.com/ushakovn/outfit/internal/models"
  "github.com/ushakovn/outfit/pkg/logger"
)

var productType models.ProductType

func main() {
  ctx := context.Background()

  logger.Init()

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
  log.Infof("mongodb connection sucessfully")

  telegramClient, err := bot.New(config.Get(ctx, config.TelegramToken).String())
  if err != nil {
    log.Fatalf("bot.New: %v", err)
  }
  log.Infof("telegram client connection sucessfully")

  senderCron := sender.NewSenderCron(productType, sender.Dependencies{
    Telegram: telegramClient,
    Mongodb:  mongoClient,
  })

  log.Infof("sender cron starting now")

  if err = senderCron.Start(ctx); err != nil {
    log.Fatalf("senderCron.Start: %v", err)
  }
  log.Infof("sender cron completed successfully")
}
