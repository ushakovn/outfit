package telegram

import (
  "context"

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

func (b *Transport) Start(ctx context.Context) {
  b.registerHandlers(ctx)

  go b.deps.Telegram.Start(ctx)
}
