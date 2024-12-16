package telegram

import (
  "context"
  "errors"
  "fmt"
  "strings"
  "time"
  "unicode/utf8"

  set "github.com/deckarep/golang-set/v2"
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
  "golang.org/x/net/html"
)

type sendErrorMessageParams struct {
  ChatId int64
  Text   string
  Menu   models.SessionMenu
}

func (b *Transport) sendErrorMessage(ctx context.Context, params sendErrorMessageParams) {
  err := b.sendMessage(ctx, sendMessageParams{
    ChatId: params.ChatId,
    Text:   params.Text,
  })
  if err != nil {
    log.
      WithField("chat_id", params.ChatId).
      WithField("menu", params.Menu).
      Errorf("b.sendMessage: %v", err)

    return
  }
}

func makeCutSizeValuesString(values []string) string {
  if len(values) > 3 {
    cop := make([]string, 3)
    copy(cop, values)

    sizesString := strings.Join(cop, ", ")
    sizesString = strings.TrimSpace(sizesString)

    return sizesString
  }

  sizesString := strings.Join(values, ", ")
  sizesString = strings.TrimSpace(sizesString)

  return sizesString
}

func setTrackingSizes(tracking *models.Tracking, sizes []string) {
  tracking.Sizes.Values = sizes
}

func setTrackingFlag(tracking *models.Tracking, flag bool) {
  tracking.Flags.WithOptional = flag
}

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
  Entities  *models.SessionEntities
}

func (b *Transport) upsertSession(ctx context.Context, params upsertSessionParams) error {
  session := models.Session{
    ChatId: params.ChatId,
    Message: models.SessionMessage{
      Id:   params.MessageID,
      Menu: params.Menu,
    },
    Tracking:  params.Tracking,
    Entities:  params.Entities,
    UpdatedAt: time.Now(),
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

func parseIssueText(fields string) (text string) {
  text = html.UnescapeString(fields)
  text = strings.TrimSpace(fields)

  return text
}

func parseTrackingComment(fields string) (comment string, err string) {
  comment = html.UnescapeString(fields)
  comment = strings.TrimSpace(fields)

  if comment == "" {
    err = `Кажется, комментарий пустой 😟
Пример корректного комментария 💬

Кепка Stussy черная 
#stussy #кепка #kixbox

Попробуйте ввести еще раз 😉
`
    return "", err
  }

  if utf8.RuneCountInString(comment) > 100 {
    err = `Комментарий может содержать до 100 символов 👀
Попробуйте ввести еще раз 😉
`
    return "", err
  }

  return comment, ""
}

func parseTrackingURL(fields string) (url string, err string) {
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

func parseTrackingSizes(fields string, session *models.Session) (values []string, err string) {
  sizesSlice := strings.Split(fields, ",")

  if len(sizesSlice) == 0 {
    storedSizes := makeCutSizeValuesString(session.Tracking.Sizes.Values)

    err = fmt.Sprintf(`Не удалось найти список размеров для товара 😟

Пример корректного ввода 💬
%s

Попробуйте еще раз 😉
`, storedSizes)

    return nil, err
  }

  for _, value := range sizesSlice {
    value = strings.ReplaceAll(value, " ", "")

    values = append(values, value)
  }

  return values, ""
}

func makeProductSizes(product models.Product) (values []string) {
  values = lo.Map(product.Options, func(option models.ProductOption, _ int) string {
    return option.Size.Base.Value
  })
  return values
}

func makeTrackingSizesText(values []string, session *models.Session) (text string) {
  sizes := strings.Join(values, ", ")
  sizes = strings.TrimSpace(sizes)

  text += fmt.Sprintf(`<b>Введенные вами размеры 📋</b>
%s
`, sizes)

  if warn := validateTrackingSizes(values, session); warn != "" {
    text += warn
  }

  text += `
Если все верно, нажмите далее
Или введите актуальные размеры заново 😉`

  return text
}

func validateTrackingSizes(values []string, session *models.Session) (warn string) {
  storedSizes := makeProductSizes(session.Tracking.ParsedProduct)

  valuesSet := set.NewSet(values...)
  valuesSet.RemoveAll(storedSizes...)

  if valuesSet.IsEmpty() {
    return ""
  }

  notFoundSizes := strings.Join(valuesSet.ToSlice(), ", ")
  notFoundSizes = strings.TrimSpace(notFoundSizes)

  warn = fmt.Sprintf(
    `
Среди них есть отсутствующие в размерной сетке 👀
%s
`, notFoundSizes)

  return warn
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

func (b *Transport) insertIssue(ctx context.Context, issue *models.Issue) error {
  _, err := b.deps.Mongodb.Insert(ctx, mongodb.InsertParams{
    CommonParams: mongodb.CommonParams{
      Database:   "outfit",
      Collection: "issues",
    },
    Document: issue,
  })
  if err != nil {
    return fmt.Errorf("b.deps.Mongodb.Insert: %w", err)
  }

  return nil
}
