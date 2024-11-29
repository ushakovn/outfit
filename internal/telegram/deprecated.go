package telegram

import (
  "context"
  "errors"
  "regexp"
  "strings"

  telegram "github.com/go-telegram/bot"
  tgmodels "github.com/go-telegram/bot/models"
  log "github.com/sirupsen/logrus"
  "github.com/spf13/cast"
  "github.com/ushakovn/outfit/internal/models"
  "github.com/ushakovn/outfit/internal/tracker"
  "github.com/ushakovn/outfit/pkg/validator"
)

// handleTrackingFieldsMenu
// Deprecated: do not use.
func (b *Bot) handleTrackingFieldsMenu(ctx context.Context, bot *telegram.Bot, update *tgmodels.Update) {
  chatId, ok := findChatIdInUpdate(update)
  if !ok {
    log.
      WithField("update.message", update.Message).
      WithField("menu", models.TrackingFieldsMenu).
      Warn("chat_id not found")

    return
  }

  parsedFields := parseTrackingFields(update.Message.Text)

  reply := newReplyKeyboard(models.TrackingFieldsMenu).
    Row().Button("Назад 👓", bot, telegram.MatchTypeExact, b.handleStartSilentMenu)

  if parsedFields.ErrorMessage != "" {
    err := b.sendMessage(ctx, sendMessageParams{
      ChatId: chatId,
      Text:   parsedFields.ErrorMessage,
      Reply:  reply,
    })
    if err != nil {
      log.
        WithField("chat_id", chatId).
        WithField("menu", models.TrackingFieldsMenu).
        Errorf("b.sendMessage: %v", err)
    }

    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingFieldsMenu).
      Warnf("parseTrackingFields: %v", parsedFields.ErrorMessage)

    return
  }

  err := b.sendMessage(ctx, sendMessageParams{
    ChatId: chatId,
    Text: `Мы получили введенные вами данные 📄.
Сейчас бот проверит карточку товара и вернется с результатом 📦.`,
    Reply: reply,
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingFieldsMenu).
      Errorf("b.sendMessage: %v", err)

    return
  }

  trackingMsg, err := b.deps.Tracker.CreateTrackingMessage(ctx, models.TrackingMessageParams{
    URL:      parsedFields.URL,
    Sizes:    parsedFields.Sizes,
    Discount: parsedFields.Discount,
  })
  if err != nil {
    if errors.Is(err, tracker.ErrUnsupportedProductType) {
      err = b.sendMessage(ctx, sendMessageParams{
        ChatId: chatId,
        Text:   `Извините, бот пока не умеет работать с данным сайтом 😟.`,
        Reply:  reply,
      })
      if err != nil {
        log.
          WithField("chat_id", chatId).
          WithField("menu", models.TrackingFieldsMenu).
          Errorf("b.sendMessage: %v", err)
      }
      return
    }

    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingFieldsMenu).
      Errorf("b.deps.Tracker.CreateTrackingMessage: %v", err)

    return
  }

  err = b.sendMessage(ctx, sendMessageParams{
    ChatId: chatId,
    Text:   trackingMsg.TextValue,
    Reply:  reply,
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingFieldsMenu).
      Errorf("b.sendMessage: %v", err)

    return
  }

  reply = newReplyKeyboard(models.TrackingFieldsMenu).
    Row().Button("Подтвердить 📨", bot, telegram.MatchTypeExact, b.handleTrackingInsertConfirmMenu).
    Row().Button("Отменить 🗑️", bot, telegram.MatchTypeExact, b.handleStartSilentMenu)

  err = b.sendMessage(ctx, sendMessageParams{
    ChatId: chatId,
    Text: `<b>Проверьте полученные от бота данные 📦:<b/>
  - Если все хорошо, подтвердите отслеживание нажав "Подтвердить 📨"
  - Если вы передумали или хотите вернуться в главное меню, нажмите "Отменить 🗑️"`,
    Reply: reply,
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingFieldsMenu).
      Errorf("b.sendMessage: %v", err)

    return
  }

  err = b.upsertSession(ctx, upsertSessionParams{
    ChatId:   chatId,
    Menu:     models.TrackingFieldsMenu,
    Tracking: newTracking(chatId, parsedFields, *trackingMsg),
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingFieldsMenu).
      Errorf("b.upsertSession: %v", err)

    return
  }
}

// parsedTrackingFields
// Deprecated: do not use
type parsedTrackingFields struct {
  ErrorMessage string
  Comment      string
  URL          string
  Sizes        models.ParseSizesParams
  Discount     *models.ParseDiscountParams
}

// parseTrackingFields
// Deprecated: do not use
func parseTrackingFields(fields string) (res parsedTrackingFields) {
  urlString := regexp.MustCompile(`1\..+\s?2\.`).FindString(fields)
  urlString = strings.Trim(urlString, "1.")
  urlString = strings.Trim(urlString, "2.")
  urlString = strings.TrimSpace(urlString)

  if urlString == "" {
    res.ErrorMessage = `Не удалось найти ссылку на товар в сообщении.

Пример ввода данных:
1. https://www.lamoda.ru/p/rtlacv500501/clothes-carharttwip-dzhinsy/
2. 46/48, M, XL, 56/54
3. 7

 Попробуйте еще раз.
`

    return res
  }

  err := validator.URL(urlString)
  if err != nil {
    res.ErrorMessage = `Кажется, введенная вами ссылка имеет неверный формат.
Пример корректной ссылки: https://www.lamoda.ru/p/rtlacv500501/clothes-carharttwip-dzhinsy/

 Попробуйте еще раз.
`

    return res
  }

  sizesString := regexp.MustCompile(`2\..+\s?3\.`).FindString(fields)
  sizesString = strings.Trim(sizesString, "2.")
  sizesString = strings.Trim(sizesString, "3.")
  sizesString = strings.TrimSpace(sizesString)

  if sizesString == "" {
    res.ErrorMessage = `Не удалось найти список размеров для товара.

Пример ввода данных:
1. https://www.lamoda.ru/p/rtlacv500501/clothes-carharttwip-dzhinsy/
2. 46/48, M, XL, 56/54
3. 7

 Попробуйте еще раз.
`

    return res
  }

  sizesSlice := strings.Split(sizesString, ",")
  if len(sizesSlice) == 0 {
    res.ErrorMessage = `Не удалось найти список размеров для товара.

Пример ввода данных:
1. https://www.lamoda.ru/p/rtlacv500501/clothes-carharttwip-dzhinsy/
2. 46/48, M, XL, 56/54
3. 7

 Попробуйте еще раз.
`

    return res
  }

  var sizesParams models.ParseSizesParams

  for _, size := range sizesSlice {
    if size == "-" {
      continue
    }
    size = strings.TrimSpace(size)

    sizesParams.Values = append(sizesParams.Values, size)
  }

  discountString := regexp.MustCompile(`3\..+\s?`).FindString(fields)
  discountString = strings.Trim(discountString, "3.")
  discountString = strings.TrimSpace(discountString)

  if discountString == "" {
    res.ErrorMessage = `Не удалось найти поле с указанием персональной скидки.

Пример ввода данных:
1. https://www.lamoda.ru/p/rtlacv500501/clothes-carharttwip-dzhinsy/
2. 46/48, M, XL, 56/54
3. 7

Если у вас отсутствует персональная скидка:
1. https://www.lamoda.ru/p/rtlacv500501/clothes-carharttwip-dzhinsy/
2. 46/48, M, XL, 56/54
3. -

 Попробуйте еще раз.
`

    return res
  }

  var discountParams *models.ParseDiscountParams

  if discountString != "-" {
    discountString = strings.Trim(discountString, "%s")

    discountInt, castErr := cast.ToInt64E(discountString)
    if castErr != nil {
      res.ErrorMessage = `Кажется, число, указанное в размере скидки некорректное. 

Пример ввода данных:
1. https://www.lamoda.ru/p/rtlacv500501/clothes-carharttwip-dzhinsy/
2. 46/48, M, XL, 56/54
3. 7

 Попробуйте еще раз.
`

      return res
    }

    discountParams = &models.ParseDiscountParams{
      Percent: discountInt,
    }
  }

  return parsedTrackingFields{
    URL:      urlString,
    Sizes:    sizesParams,
    Discount: discountParams,
  }
}
