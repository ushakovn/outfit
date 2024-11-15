package main

import (
  "context"
  "net/http"
  "os"
  "os/signal"
  "syscall"

  "github.com/go-resty/resty/v2"
  "github.com/go-telegram/bot"
  log "github.com/sirupsen/logrus"
  "github.com/ushakovn/outfit/internal/models"
  "github.com/ushakovn/outfit/internal/parser/lamoda"
  "github.com/ushakovn/outfit/internal/provider/mongodb"
  "github.com/ushakovn/outfit/internal/telegram"
  "github.com/ushakovn/outfit/internal/tracker"
  "github.com/ushakovn/outfit/pkg/parser/xpath"
)

func main() {
  ctx := context.Background()

  mongoClient, err := mongodb.NewClient(ctx,
    mongodb.Config{
      Host: "localhost",
      Port: "27017",
      Authentication: &mongodb.Authentication{
        User:     "outfit",
        Password: "scp",
      },
    },
    mongodb.Dependencies{
      Client: http.DefaultClient,
    })
  if err != nil {
    log.Fatal("mongodb.NewClient: %v", err)
  }

  xpathParser := xpath.NewParser(xpath.Dependencies{
    Client: resty.NewWithClient(http.DefaultClient),
  })

  lamodaParser := lamoda.NewParser(lamoda.Dependencies{
    Xpath: xpathParser,
  })

  trackerClient := tracker.NewTracker(tracker.Config{}, tracker.Dependencies{
    Mongodb: mongoClient,
    Parsers: map[models.ProductType]models.Parser{
      models.ProductTypeLamoda: lamodaParser,
    },
  })

  telegramClient, err := bot.New("6205725186:AAFfnWUUclsCcGLR4Uq2U-2vXqQ3PjK1NO4")
  if err != nil {
    log.Fatal("bot.New: %v", err)
  }

  telegramBot := telegram.NewBot(telegram.Dependencies{
    Tracker:  trackerClient,
    Telegram: telegramClient,
    Mongodb:  mongoClient,
  })

  telegramBot.Start(ctx)

  exitSignal := make(chan os.Signal)
  signal.Notify(exitSignal, syscall.SIGINT, syscall.SIGTERM)
  <-exitSignal
}
