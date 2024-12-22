package telegram

import (
  "context"
  "fmt"

  telegram "github.com/go-telegram/bot"
  "github.com/ushakovn/outfit/internal/app/tracker"
  "github.com/ushakovn/outfit/internal/deps/storage/mongodb"
  "github.com/ushakovn/outfit/internal/models"
  "github.com/ushakovn/outfit/pkg/cache"
)

type Transport struct {
  deps Dependencies
}

type Dependencies struct {
  Tracker  *tracker.Tracker
  Telegram *telegram.Bot
  Mongodb  *mongodb.Client

  cache dependenciesCache
}

type dependenciesCache struct {
  trackings *cache.Cache[models.ChatId, trackingIndex, models.ProductURL]
}

type trackingIndex = int

func NewTransport(deps Dependencies) *Transport {
  deps.cache = makeDependenciesCache()

  return &Transport{deps: deps}
}

func makeDependenciesCache() dependenciesCache {
  return dependenciesCache{
    trackings: cache.NewCache[
      models.ChatId,
      trackingIndex,
      models.ProductURL,
    ](),
  }
}

func (b *Transport) Start(ctx context.Context) error {
  b.registerHandlers(ctx)

  err := b.beforeStartChecks(ctx)
  if err != nil {
    return fmt.Errorf("b.beforeStartChecks: %w", err)
  }

  go b.deps.Telegram.Start(ctx)

  return nil
}

func (b *Transport) beforeStartChecks(ctx context.Context) error {
  err := b.checkTrackingIndex(ctx)
  if err != nil {
    return fmt.Errorf("b.checkTrackingIndex: %w", err)
  }
  return nil
}
