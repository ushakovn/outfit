package telegram

import (
	"context"

	telegram "github.com/go-telegram/bot"
	"github.com/ushakovn/outfit/internal/provider/mongodb"
	"github.com/ushakovn/outfit/internal/tracker"
)

type Bot struct {
	deps Dependencies
}

type Dependencies struct {
	Tracker  *tracker.Tracker
	Telegram *telegram.Bot
	Mongodb  *mongodb.Client
	cache    cache
}

type cache struct {
	TrackingURLs map[chatSelectedTracking]string
}

type chatSelectedTracking struct {
	ChatId int64
	Index  int
}

func NewBot(deps Dependencies) *Bot {
	deps.cache = cache{
		TrackingURLs: make(map[chatSelectedTracking]string),
	}
	return &Bot{deps: deps}
}

func (b *Bot) Start(ctx context.Context) {
	b.registerHandlers(ctx)

	go b.deps.Telegram.Start(ctx)
}
