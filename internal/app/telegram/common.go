package telegram

import (
  "context"
  "errors"
  "fmt"
  "strings"
  "time"

  telegram "github.com/go-telegram/bot"
  tgmodels "github.com/go-telegram/bot/models"
  tginline "github.com/go-telegram/ui/keyboard/inline"
  tgreply "github.com/go-telegram/ui/keyboard/reply"
  tgslider "github.com/go-telegram/ui/slider"
  "github.com/samber/lo"
  log "github.com/sirupsen/logrus"
  "github.com/ushakovn/outfit/internal/app/telegram/assets"
  "github.com/ushakovn/outfit/internal/app/tracker"
  "github.com/ushakovn/outfit/internal/deps/storage/mongodb"
  "github.com/ushakovn/outfit/internal/models"
  "github.com/ushakovn/outfit/pkg/stringer"
  "github.com/ushakovn/outfit/pkg/validator"
)

func (b *Transport) findSession(ctx context.Context, chatID int64) (*models.Session, error) {
  res, err := b.deps.Mongodb.Get(ctx, mongodb.GetParams{
    CommonParams: mongodb.CommonParams{
      Database:   "outfit",
      Collection: "sessions",
      StructType: models.Session{},
    },
    Filters: map[string]any{
      "chat_id": chatID,
    },
  })
  if err != nil {
    return nil, fmt.Errorf("b.deps.Mongodb.Get: %w", err)
  }

  session, ok := res.(*models.Session)
  if !ok {
    return nil, fmt.Errorf("cast %v with type: %[1]T to: %T failed", res, new(models.Session))
  }

  return session, nil
}

func (b *Transport) deleteTracking(ctx context.Context, session *models.Session) error {
  _, err := b.deps.Mongodb.Delete(ctx, mongodb.DeleteParams{
    CommonParams: mongodb.CommonParams{
      Database:   "outfit",
      Collection: "trackings",
    },
    Filters: map[string]any{
      "chat_id": session.Tracking.ChatId,
      "url":     session.Tracking.URL,
    },
  })
  if err != nil {
    return fmt.Errorf("b.deps.Mongodb.Delete: %w", err)
  }

  return nil
}

func (b *Transport) insertTracking(ctx context.Context, tracking models.Tracking) error {
  _, err := b.deps.Mongodb.Insert(ctx, mongodb.InsertParams{
    CommonParams: mongodb.CommonParams{
      Database:   "outfit",
      Collection: "trackings",
    },
    Document: tracking,
  })
  if err != nil {
    return fmt.Errorf("b.deps.Mongodb.Insert: %w", err)
  }

  return nil
}

type sendMessageParams struct {
  ChatId int64
  Text   string
  Reply  tgmodels.ReplyMarkup
}

func (b *Transport) sendMessage(ctx context.Context, params sendMessageParams) error {
  _, err := b.deps.Telegram.SendMessage(ctx, &telegram.SendMessageParams{
    ChatID:      params.ChatId,
    Text:        params.Text,
    ParseMode:   tgmodels.ParseModeHTML,
    ReplyMarkup: params.Reply,
    LinkPreviewOptions: &tgmodels.LinkPreviewOptions{
      IsDisabled: lo.ToPtr(true),
    },
  })
  if err != nil {
    return fmt.Errorf("b.deps.Telegram.SendMessage: %w", err)
  }

  return nil
}

type upsertSessionParams struct {
  ChatId    int64
  Menu      models.SessionMenu
  MessageID *int64
  Tracking  *models.Tracking
}

func (b *Transport) upsertSession(ctx context.Context, params upsertSessionParams) error {
  session := models.Session{
    ChatId: params.ChatId,
    Message: models.SessionMessage{
      Id:        params.MessageID,
      Menu:      params.Menu,
      UpdatedAt: time.Now(),
    },
    Tracking: params.Tracking,
  }

  _, err := b.deps.Mongodb.Upsert(ctx, mongodb.UpdateParams{
    GetParams: mongodb.GetParams{
      CommonParams: mongodb.CommonParams{
        Database:   "outfit",
        Collection: "sessions",
        StructType: models.Session{},
      },
      Filters: map[string]any{
        "chat_id": session.ChatId,
      },
    },
    Document: session,
  })
  if err != nil {
    return fmt.Errorf("b.deps.Mongodb.Upsert: %w", err)
  }

  return nil
}

func (b *Transport) findTracking(ctx context.Context, chatId int64, url string) (*models.Tracking, error) {
  res, err := b.deps.Mongodb.Get(ctx, mongodb.GetParams{
    CommonParams: mongodb.CommonParams{
      Database:   "outfit",
      Collection: "trackings",
      StructType: models.Tracking{},
    },
    Filters: map[string]any{
      "url":     url,
      "chat_id": chatId,
    },
  })
  if err != nil {
    if errors.Is(err, mongodb.ErrNotFound) {
      return nil, nil
    }
    return nil, fmt.Errorf("b.deps.Mongodb.Get: %w", err)
  }

  tracking, ok := res.(*models.Tracking)
  if !ok {
    return nil, fmt.Errorf("cast %v with type: %[1]T to: %T failed", res, new(models.Tracking))
  }

  return tracking, nil
}

func (b *Transport) listTrackings(ctx context.Context, chatID int64) ([]*models.Tracking, error) {
  res, err := b.deps.Mongodb.Find(ctx, mongodb.FindParams{
    CommonParams: mongodb.CommonParams{
      Database:   "outfit",
      Collection: "trackings",
      StructType: models.Tracking{},
    },
    Filters: map[string]any{
      "chat_id": chatID,
    },
    Limit: 100,
  })
  if err != nil {
    return nil, fmt.Errorf("b.deps.Mongodb.Find: %v", err)
  }

  list := make([]*models.Tracking, 0, len(res))

  for _, record := range res {
    tracking, ok := record.(*models.Tracking)
    if !ok {
      return nil, fmt.Errorf("cast %v with type: %[1]T to: %T failed", record, new(models.Tracking))
    }

    list = append(list, tracking)
  }

  return list, nil
}

func (b *Transport) checkProductURL(url string) error {
  if err := b.deps.Tracker.CheckProductURL(url); err != nil {
    return fmt.Errorf("b.deps.Tracker.CheckProductURL: %w", err)
  }

  return nil
}

func (b *Transport) createMessage(ctx context.Context, url string) (*models.SendableMessage, error) {
  message, err := b.deps.Tracker.CreateMessage(ctx, tracker.CreateMessageParams{
    URL: url,
  })
  if err != nil {
    return nil, fmt.Errorf("b.deps.Tracker.CreateMessage: %w", err)
  }

  return message, nil
}

func parseTrackingInputURL(fields string) (url string, err string) {
  url = stringer.ExtractURL(fields)

  if e := validator.URL(url); e != nil {
    err = `Кажется, введенная ссылка имеет неверный формат 😟

Пример корректной ссылки 💬 
https://www.lamoda.ru/p/rtlacv500501/clothes-carharttwip-dzhinsy/

Попробуйте еще раз 😉
`
    return "", err
  }

  return url, ""
}

func parseTrackingInputSizes(fields string) (values []string, err string) {
  sizesSlice := strings.Split(fields, ",")

  if len(sizesSlice) == 0 {
    err = `Не удалось найти список размеров для товара 😟

Пример корректного ввода 💬
S INT, M INT

Попробуйте еще раз 😉
`

    return nil, err
  }

  for _, value := range sizesSlice {
    value = strings.TrimSpace(value)

    values = append(values, value)
  }

  return values, ""
}

func findChatIdInUpdate(update *tgmodels.Update) (int64, bool) {
  if update != nil && update.Message != nil && update.Message.Chat.ID != 0 {
    return update.Message.Chat.ID, true
  }
  return 0, false
}

func findChatIdInMaybeInaccessible(msg tgmodels.MaybeInaccessibleMessage) (int64, bool) {
  if msg.Message != nil && msg.Message.Chat.ID != 0 {
    return msg.Message.Chat.ID, true
  }
  if msg.InaccessibleMessage != nil && msg.InaccessibleMessage.Chat.ID != 0 {
    return msg.InaccessibleMessage.Chat.ID, true
  }
  return 0, false
}

func newInlineKeyboard(bot *telegram.Bot, prefix string) *tginline.Keyboard {
  return tginline.New(bot,
    tginline.OnError(func(err error) {
      log.Errorf("telegram.InlineKeyboard: %v", err)
    }),
    tginline.WithPrefix(prefix),
    tginline.NoDeleteAfterClick(),
  )
}

func newReplyKeyboard(prefix string) *tgreply.ReplyKeyboard {
  return tgreply.New(
    tgreply.WithPrefix(prefix),
    tgreply.IsOneTimeKeyboard(),
    tgreply.ResizableKeyboard(),
  )
}

type trackingSliderParams struct {
  ChatId    int64
  Bot       *telegram.Bot
  Trackings []*models.Tracking
}

func (b *Transport) newTrackingSlider(params trackingSliderParams) *tgslider.Slider {
  slides := make([]tgslider.Slide, 0, len(params.Trackings))

  for _, tracking := range params.Trackings {
    res := models.Sendable(params.ChatId).
      SetTrackingPtr(tracking).
      BuildTrackingMessage()

    if !res.IsValid {
      continue
    }

    slide := tgslider.Slide{
      Text:  telegram.EscapeMarkdown(res.Message.Text.Value),
      Photo: tracking.ParsedProduct.ImageURL,
    }

    if slide.Photo == "" {
      slide.Photo = string(assets.NoPhoto)
      slide.IsUpload = true
    }

    slides = append(slides, slide)
  }

  return tgslider.New(params.Bot, slides,
    tgslider.OnError(func(err error) {
      log.Errorf("telegram.TrackingSlider: %v", err)
    }),
    tgslider.WithPrefix("tracking"),
    tgslider.OnSelect("Удалить", true, b.handleTrackingDeleteMenu),
    tgslider.OnCancel("Назад", true, b.handleTrackingSilentMenu),
  )
}