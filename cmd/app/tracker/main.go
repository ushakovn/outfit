package main

import (
  "context"
  "flag"
  "net/http"

  "github.com/go-resty/resty/v2"
  log "github.com/sirupsen/logrus"
  "github.com/ushakovn/outfit/internal/app/tracker"
  "github.com/ushakovn/outfit/internal/deps/parsers/kixbox"
  "github.com/ushakovn/outfit/internal/deps/parsers/lamoda"
  "github.com/ushakovn/outfit/internal/deps/parsers/lime"
  "github.com/ushakovn/outfit/internal/deps/parsers/oktyabr"
  "github.com/ushakovn/outfit/internal/deps/parsers/ridestep"
  "github.com/ushakovn/outfit/internal/deps/parsers/traektoria"
  "github.com/ushakovn/outfit/internal/deps/storage/mongodb"
  "github.com/ushakovn/outfit/internal/models"
  "github.com/ushakovn/outfit/pkg/logger"
  "github.com/ushakovn/outfit/pkg/parser/xpath"

  _ "github.com/ushakovn/boiler/pkg/app"
)

var productType models.ProductType

func main() {
  ctx := context.Background()

  logger.Init()

  log.Warn("tracker cron app initializing")

  flag.StringVar(&productType, "type", "", "product type")
  flag.Parse()

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
    log.Fatalf("mongodb.NewClient: %v", err)
  }

  httpClient := resty.NewWithClient(http.DefaultClient)
  xpathParser := xpath.NewParser(xpath.Dependencies{Client: httpClient})

  lamodaParser := lamoda.NewParser(lamoda.Dependencies{Xpath: xpathParser})
  kixboxParser := kixbox.NewParser(kixbox.Dependencies{Xpath: xpathParser})
  oktyabrParser := oktyabr.NewParser(oktyabr.Dependencies{Xpath: xpathParser})
  limeParser := lime.NewParser(lime.Dependencies{Client: httpClient})
  ridestepParser := ridestep.NewParser(ridestep.Dependencies{Xpath: xpathParser})
  traektoriaParser := traektoria.NewParser(traektoria.Dependencies{Client: httpClient})

  trackerCron := tracker.NewTrackerCron(productType, tracker.Dependencies{
    Mongodb: mongoClient,
    Parsers: map[models.ProductType]models.Parser{
      models.ProductTypeLamoda:     lamodaParser,
      models.ProductTypeKixbox:     kixboxParser,
      models.ProductTypeOktyabr:    oktyabrParser,
      models.ProductTypeLime:       limeParser,
      models.ProductTypeRidestep:   ridestepParser,
      models.ProductTypeTraektoria: traektoriaParser,
    },
  })

  if err = trackerCron.Start(ctx); err != nil {
    log.Fatalf("trackerCron.Start: %v", err)
  }

  log.Warn("tracker cron app terminating")
}
