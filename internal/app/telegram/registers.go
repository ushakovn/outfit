package telegram

import (
  "context"
  "strings"

  telegram "github.com/go-telegram/bot"
  tgmodels "github.com/go-telegram/bot/models"
  log "github.com/sirupsen/logrus"
  "github.com/ushakovn/outfit/internal/models"
)

func (b *Transport) registerHandlers(ctx context.Context) {
  b.registerCommandHandler(ctx, registerCommandHandlerParams{
    Command: "/start",
    Handler: b.handleStartMenu,
  })

  b.registerTextHandler(ctx, registerTextHandlerParams{
    Menu:    models.TrackingInsertMenu,
    Handler: b.handleTrackingInputUrlMenu,
  })

  b.registerTextHandler(ctx, registerTextHandlerParams{
    Menu:    models.TrackingInputUrlMenu,
    Handler: b.handleTrackingInputSizesMenu,
  })

  b.registerTextHandler(ctx, registerTextHandlerParams{
    Menu:    models.TrackingCommentMenu,
    Handler: b.handleTrackingInputCommentMenu,
  })
}

type registerCommandHandlerParams struct {
  Command string
  Handler func(ctx context.Context, bot *telegram.Bot, update *tgmodels.Update)
}

func (b *Transport) registerCommandHandler(_ context.Context, params registerCommandHandlerParams) {
  b.deps.Telegram.RegisterHandler(
    telegram.HandlerTypeMessageText, params.Command,
    telegram.MatchTypeExact, params.Handler,
  )
}

type registerTextHandlerParams struct {
  Menu    models.SessionMenu
  Handler func(ctx context.Context, bot *telegram.Bot, update *tgmodels.Update)
}

func (b *Transport) registerTextHandler(ctx context.Context, params registerTextHandlerParams) {
  b.deps.Telegram.RegisterHandlerMatchFunc(
    func(update *tgmodels.Update) bool {
      if isBackButtonMessage(update) || isNextButtonMessage(update) {
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
          WithField("menu", params.Menu).
          Errorf("b.findSession: %v", err)

        return false
      }

      return session.Message.Menu == params.Menu
    },
    params.Handler,
  )
}

func hasButtonText(update *tgmodels.Update, text string) bool {
  if update == nil || update.Message == nil {
    return false
  }
  return strings.Contains(update.Message.Text, text)
}

func isBackButtonMessage(update *tgmodels.Update) bool {
  return hasButtonText(update, "Назад")
}

func isNextButtonMessage(update *tgmodels.Update) bool {
  return hasButtonText(update, "Далее")
}
