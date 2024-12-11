package main

import (
  "context"
  "net/http"
  "os"
  "os/signal"
  "syscall"

  "github.com/go-resty/resty/v2"
  log "github.com/sirupsen/logrus"
  tgtransport "github.com/ushakovn/outfit/internal/app/telegram"
  "github.com/ushakovn/outfit/internal/app/tracker"
  "github.com/ushakovn/outfit/internal/config"
  "github.com/ushakovn/outfit/internal/deps/parsers/kixbox"
  "github.com/ushakovn/outfit/internal/deps/parsers/lamoda"
  "github.com/ushakovn/outfit/internal/deps/parsers/lime"
  "github.com/ushakovn/outfit/internal/deps/parsers/oktyabr"
  "github.com/ushakovn/outfit/internal/deps/parsers/ridestep"
  "github.com/ushakovn/outfit/internal/deps/storage/mongodb"
  tgbot "github.com/ushakovn/outfit/internal/deps/telegram"
  "github.com/ushakovn/outfit/internal/models"
  "github.com/ushakovn/outfit/pkg/logger"
  "github.com/ushakovn/outfit/pkg/parser/xpath"

  _ "github.com/ushakovn/boiler/pkg/app"
)

func main() {
  ctx := context.Background()

  logger.Init()

  log.Warn("telegram bot app initializing")

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

  httpClient := resty.NewWithClient(http.DefaultClient)
  xpathParser := xpath.NewParser(xpath.Dependencies{Client: httpClient})

  lamodaParser := lamoda.NewParser(lamoda.Dependencies{Xpath: xpathParser})
  kixboxParser := kixbox.NewParser(kixbox.Dependencies{Xpath: xpathParser})
  oktyabrParser := oktyabr.NewParser(oktyabr.Dependencies{Xpath: xpathParser})
  limeParser := lime.NewParser(lime.Dependencies{Client: httpClient})
  ridestepParser := ridestep.NewParser(ridestep.Dependencies{Xpath: xpathParser})

  trackerClient := tracker.NewTracker(tracker.Dependencies{
    Mongodb: mongoClient,
    Parsers: map[models.ProductType]models.Parser{
      models.ProductTypeLamoda:   lamodaParser,
      models.ProductTypeKixbox:   kixboxParser,
      models.ProductTypeOktyabr:  oktyabrParser,
      models.ProductTypeLime:     limeParser,
      models.ProductTypeRidestep: ridestepParser,
    },
  })

  telegramBotClient, err := tgbot.NewBotClient(tgbot.Config{
    Token: config.Get(ctx, config.TelegramToken).String(),
  })
  if err != nil {
    log.Fatalf("tgbot.NewBotClient: %v", err)
  }

  telegramBotTransport := tgtransport.NewTransport(tgtransport.Dependencies{
    Tracker:  trackerClient,
    Telegram: telegramBotClient,
    Mongodb:  mongoClient,
  })

  telegramBotTransport.Start(ctx)

  exitSignal := make(chan os.Signal)
  signal.Notify(exitSignal, syscall.SIGINT, syscall.SIGTERM)
  <-exitSignal

  log.Warn("telegram bot app terminating")
}
