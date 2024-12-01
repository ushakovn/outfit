package tracker

import (
  "errors"

  "github.com/ushakovn/outfit/internal/deps/storage/mongodb"
  "github.com/ushakovn/outfit/internal/models"
)

var ErrUnsupportedProductType = errors.New("unsupported product type")

type Tracker struct {
  config Config
  deps   Dependencies
}

type Config struct {
  IsCron      bool
  ProductType models.ProductType
}

type Dependencies struct {
  Mongodb *mongodb.Client
  Parsers map[models.ProductType]models.Parser
}

func NewTracker(deps Dependencies) *Tracker {
  return &Tracker{deps: deps}
}

func NewTrackerCron(typ models.ProductType, deps Dependencies) *Tracker {
  return &Tracker{
    config: Config{
      IsCron:      true,
      ProductType: typ,
    },
    deps: deps,
  }
}
