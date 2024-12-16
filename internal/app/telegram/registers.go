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
    Menus:   []models.SessionMenu{models.TrackingInsertMenu},
    Handler: b.handleTrackingInputUrlMenu,
  })

  b.registerTextHandler(ctx, registerTextHandlerParams{
    Menus: []models.SessionMenu{
      models.TrackingInputUrlMenu,
      models.TrackingInputSizesMenu,
    },
    Handler: b.handleTrackingInputSizesMenu,
  })

  b.registerTextHandler(ctx, registerTextHandlerParams{
    Menus: []models.SessionMenu{
      models.TrackingSearchInputMenu,
      models.TrackingSearchSilentInputMenu,
    },
    Handler: b.handleTrackingSearchShowMenu,
  })

  b.registerTextHandler(ctx, registerTextHandlerParams{
    Menus:   []models.SessionMenu{models.TrackingCommentMenu},
    Handler: b.handleTrackingInputCommentMenu,
  })

  b.registerTextHandler(ctx, registerTextHandlerParams{
    Menus:   []models.SessionMenu{models.IssueInputTypeMenu},
    Handler: b.handleIssueInputTextMenu,
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
  Menus   []models.SessionMenu
  Handler func(ctx context.Context, bot *telegram.Bot, update *tgmodels.Update)
}

func (b *Transport) registerTextHandler(ctx context.Context, params registerTextHandlerParams) {
  b.deps.Telegram.RegisterHandlerMatchFunc(
    func(update *tgmodels.Update) bool {
      if isControlButtonText(update) {
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
          WithField("menus", params.Menus).
          Errorf("b.findSession: %v", err)

        return false
      }

      for _, menu := range params.Menus {
        if session.Message.Menu == menu {
          return true
        }
      }

      return false
    },
    params.Handler,
  )
}

func containsButtonText(update *tgmodels.Update, parts []string) bool {
  if update == nil || update.Message == nil {
    return false
  }

  for _, text := range parts {
    if strings.Contains(update.Message.Text, text) {
      return true
    }
  }

  return false
}

func isControlButtonText(update *tgmodels.Update) bool {
  return containsButtonText(update, []string{
    "Назад", "Далее", "Включить",
    "Помощь", "Подтвердить", "Пропустить",
  })
}
