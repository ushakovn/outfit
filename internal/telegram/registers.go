package telegram

import (
	"context"

	telegram "github.com/go-telegram/bot"
	tgmodels "github.com/go-telegram/bot/models"
	log "github.com/sirupsen/logrus"
	"github.com/ushakovn/outfit/internal/models"
)

func (b *Bot) registerHandlers(ctx context.Context) {
	b.registerStartHandler(ctx)
	b.registerTrackingFieldsMenuHandler(ctx)
}

func (b *Bot) registerStartHandler(_ context.Context) {
	b.deps.Telegram.RegisterHandler(
		telegram.HandlerTypeMessageText, "/start",
		telegram.MatchTypeExact, b.handleStartMenu,
	)
}

func (b *Bot) registerTrackingFieldsMenuHandler(ctx context.Context) {
	b.deps.Telegram.RegisterHandlerMatchFunc(
		func(update *tgmodels.Update) bool {
			chatId, ok := findChatIdInUpdate(update)
			if !ok {
				return false
			}

			session, err := b.findSession(ctx, chatId)
			if err != nil {
				log.Errorf("telegram.handleTrackingFieldsMenu: findSession: %v", err)

				return false
			}

			return session.Message.Menu == models.TrackingInsertMenu && update.Message.Text != "Назад"
		},
		b.handleTrackingFieldsMenu,
	)
}
