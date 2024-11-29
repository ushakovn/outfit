package telegram

import (
  "context"
  "strings"

  telegram "github.com/go-telegram/bot"
  tgmodels "github.com/go-telegram/bot/models"
  log "github.com/sirupsen/logrus"
  "github.com/ushakovn/outfit/internal/models"
)

func (b *Bot) registerHandlers(ctx context.Context) {
  b.registerStartHandler(ctx)
  b.registerTrackingInputUrlMenuHandler(ctx)
  b.registerTrackingInputSizesMenuHandler(ctx)
}

func (b *Bot) registerStartHandler(_ context.Context) {
  b.deps.Telegram.RegisterHandler(
    telegram.HandlerTypeMessageText, "/start",
    telegram.MatchTypeExact, b.handleStartMenu,
  )
}

func (b *Bot) registerTrackingInputUrlMenuHandler(ctx context.Context) {
  b.deps.Telegram.RegisterHandlerMatchFunc(
    func(update *tgmodels.Update) bool {
      if isBackButtonMessage(update) {
        return false
      }

      chatId, ok := findChatIdInUpdate(update)
      if !ok {
        return false
      }

      session, err := b.findSession(ctx, chatId)
      if err != nil {
        log.
          WithField("chat_id", chatId).
          WithField("previous_menu", models.TrackingInsertMenu).
          Errorf("b.findSession: %v", err)

        return false
      }

      return session.Message.Menu == models.TrackingInsertMenu
    },
    b.handleTrackingInputUrlMenu,
  )
}

func (b *Bot) registerTrackingInputSizesMenuHandler(ctx context.Context) {
  b.deps.Telegram.RegisterHandlerMatchFunc(
    func(update *tgmodels.Update) bool {
      if isBackButtonMessage(update) {
        return false
      }

      chatId, ok := findChatIdInUpdate(update)
      if !ok {
        return false
      }

      session, err := b.findSession(ctx, chatId)
      if err != nil {
        log.
          WithField("chat_id", chatId).
          WithField("previous_menu", models.TrackingInputUrlMenu).
          Errorf("b.findSession: %v", err)

        return false
      }

      return session.Message.Menu == models.TrackingInputUrlMenu
    },
    b.handleTrackingInputSizesMenu,
  )
}

func isBackButtonMessage(update *tgmodels.Update) bool {
  if update == nil || update.Message == nil {
    return false
  }
  return strings.Contains(update.Message.Text, "Назад")
}
