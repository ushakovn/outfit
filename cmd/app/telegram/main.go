package main

import (
  "context"
  "net/http"
  "os"
  "os/signal"
  "syscall"

  "github.com/go-resty/resty/v2"
  tgbot "github.com/go-telegram/bot"
  log "github.com/sirupsen/logrus"
  "github.com/ushakovn/outfit/internal/app/telegram"
  "github.com/ushakovn/outfit/internal/app/tracker"
  "github.com/ushakovn/outfit/internal/config"
  "github.com/ushakovn/outfit/internal/deps/parsers/lamoda"
  "github.com/ushakovn/outfit/internal/deps/storage/mongodb"
  "github.com/ushakovn/outfit/internal/models"
  "github.com/ushakovn/outfit/pkg/parser/xpath"

  _ "github.com/ushakovn/boiler/pkg/app"
)

func main() {
  ctx := context.Background()

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

  xpathParser := xpath.NewParser(xpath.Dependencies{
    Client: resty.NewWithClient(http.DefaultClient),
  })

  lamodaParser := lamoda.NewParser(lamoda.Dependencies{
    Xpath: xpathParser,
  })

  trackerClient := tracker.NewTracker(tracker.Dependencies{
    Mongodb: mongoClient,
    Parsers: map[models.ProductType]models.Parser{
      models.ProductTypeLamoda: lamodaParser,
    },
  })

  telegramClient, err := tgbot.New(config.Get(ctx, config.TelegramToken).String())
  if err != nil {
    log.Fatalf("bot.New: %v", err)
  }

  telegramTransport := telegram.NewTransport(telegram.Dependencies{
    Tracker:  trackerClient,
    Telegram: telegramClient,
    Mongodb:  mongoClient,
  })

  telegramTransport.Start(ctx)

  exitSignal := make(chan os.Signal)
  signal.Notify(exitSignal, syscall.SIGINT, syscall.SIGTERM)
  <-exitSignal
}
