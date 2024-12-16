package telegram

import (
  "context"
  "fmt"

  telegram "github.com/go-telegram/bot"
  "github.com/ushakovn/outfit/internal/app/tracker"
  "github.com/ushakovn/outfit/internal/deps/storage/mongodb"
)

type Transport struct {
  deps Dependencies
}

type Dependencies struct {
  Tracker  *tracker.Tracker
  Telegram *telegram.Bot
  Mongodb  *mongodb.Client
  cache    Cache
}

type Cache struct {
  TrackingURLs map[chatSelectedTracking]string
}

type chatSelectedTracking struct {
  ChatId int64
  Index  int
}

func NewTransport(deps Dependencies) *Transport {
  deps.cache = Cache{
    TrackingURLs: make(map[chatSelectedTracking]string),
  }
  return &Transport{deps: deps}
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
