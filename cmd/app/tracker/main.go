package main

import (
  "context"
  "flag"
  "net/http"

  "github.com/go-resty/resty/v2"
  log "github.com/sirupsen/logrus"
  "github.com/ushakovn/outfit/internal/app/tracker"
  "github.com/ushakovn/outfit/internal/deps/parsers/lamoda"
  "github.com/ushakovn/outfit/internal/deps/storage/mongodb"
  "github.com/ushakovn/outfit/internal/models"
  "github.com/ushakovn/outfit/pkg/parser/xpath"

  _ "github.com/ushakovn/boiler/pkg/app"
)

var productType models.ProductType

func main() {
  ctx := context.Background()

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

  xpathParser := xpath.NewParser(xpath.Dependencies{
    Client: resty.NewWithClient(http.DefaultClient),
  })

  lamodaParser := lamoda.NewParser(lamoda.Dependencies{
    Xpath: xpathParser,
  })

  trackerCron := tracker.NewTrackerCron(productType, tracker.Dependencies{
    Mongodb: mongoClient,
    Parsers: map[models.ProductType]models.Parser{
      models.ProductTypeLamoda: lamodaParser,
    },
  })

  if err = trackerCron.Start(ctx); err != nil {
    log.Fatalf("trackerCron.Start: %v", err)
  }
}
