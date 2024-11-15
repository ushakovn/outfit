package telegram

import (
  "context"
  "errors"
  "fmt"
  "regexp"
  "strings"
  "time"

  telegram "github.com/go-telegram/bot"
  tgmodels "github.com/go-telegram/bot/models"
  tgreply "github.com/go-telegram/ui/keyboard/reply"
  "github.com/samber/lo"
  log "github.com/sirupsen/logrus"
  "github.com/spf13/cast"
  "github.com/ushakovn/outfit/internal/models"
  "github.com/ushakovn/outfit/internal/provider/mongodb"
  "github.com/ushakovn/outfit/internal/tracker"
  "github.com/ushakovn/outfit/pkg/validator"
)

type Bot struct {
  deps Dependencies
}

type Dependencies struct {
  Tracker  *tracker.Tracker
  Telegram *telegram.Bot
  Mongodb  *mongodb.Client
}

func NewBot(deps Dependencies) *Bot {
  return &Bot{deps: deps}
}

func (b *Bot) Start(ctx context.Context) {
  b.registerHandlers(ctx)

  go b.deps.Telegram.Start(ctx)
}

func (b *Bot) registerHandlers(ctx context.Context) {
  b.deps.Telegram.RegisterHandler(telegram.HandlerTypeMessageText, "/start",
    telegram.MatchTypeExact, b.handleStartMenu)

  b.deps.Telegram.RegisterHandlerMatchFunc(
    func(update *tgmodels.Update) bool {
      chatID, ok := findChatID(update)
      if !ok {
        return false
      }

      session, err := b.findSession(ctx, models.Telegram{
        ChatID: chatID,
      })
      if err != nil {
        log.Errorf("telegram.handleTrackingFieldsMenu: findSession: %v", err)
        return false
      }

      return session.Message.Menu == models.TrackingInsertMenu

    },
    b.handleTrackingFieldsMenu,
  )
}

func (b *Bot) handleMockMenu(ctx context.Context, bot *telegram.Bot, update *tgmodels.Update) {
  // TODO: mock menu.
}

func (b *Bot) handleStartMenu(ctx context.Context, bot *telegram.Bot, update *tgmodels.Update) {
  text := telegram.EscapeMarkdown(`Данный бот создан для отслеживания товаров. 
  
  Бот отсылает уведомления если:
  1. Цена на товар была снижена или появилась скидка на товар.
  2. Распроданный товар снова появился в наличие.
  
  Управление ботом происходит с помощью виртуальной клавиатуры, под сообщением:
  1. Добавить отслеживание - добавляет новое отслеживание по вашему товару
  2. Мои отслеживания - выводит список товаров, для которых подключено отслеживание
  3. Удалить отслеживание - удаляет созданное вами ранее отслеживание
  
  Воспользуйтесь виртуальной клавиатурой, чтобы продолжить.
  `)

  chatID, ok := findChatID(update)
  if !ok {
    return
  }

  reply := tgreply.New(tgreply.WithPrefix("start"), tgreply.IsOneTimeKeyboard(), tgreply.ResizableKeyboard()).
    Row().Button("Добавить отслеживание", bot, telegram.MatchTypeExact, b.handleTrackingInsertMenu).
    Row().Button("Мои отслеживания", bot, telegram.MatchTypeExact, b.handleMockMenu).
    Row().Button("Удалить отслеживание", bot, telegram.MatchTypeExact, b.handleMockMenu)

  _, err := bot.SendMessage(ctx, &telegram.SendMessageParams{
    ChatID:      chatID,
    Text:        text,
    ParseMode:   tgmodels.ParseModeMarkdown,
    ReplyMarkup: reply,
    LinkPreviewOptions: &tgmodels.LinkPreviewOptions{
      IsDisabled: lo.ToPtr(true),
    },
  })
  if err != nil {
    log.Errorf("telegram.handleStartMenu: bot.SendMessage: %v", err)
    return
  }

  err = b.upsertSession(ctx, models.Session{
    Telegram: models.Telegram{
      ChatID: chatID,
    },
    Message: models.SessionMessage{
      Menu:      models.StartMenu,
      CreatedAt: time.Now(),
    },
  })
  if err != nil {
    log.Errorf("telegram.handleStartMenu: b.upsertSession: %v", err)
  }
}

func (b *Bot) handleStartSilentMenu(ctx context.Context, bot *telegram.Bot, update *tgmodels.Update) {
  text := telegram.EscapeMarkdown(`Вы вернулись в главное меню бота. 
Если вам требуется подсказка, нажмите клавишу "Помощь", на виртуальной клавиатуре или наберите /start.
`)

  chatID, ok := findChatID(update)
  if !ok {
    return
  }

  reply := tgreply.New(tgreply.WithPrefix("start"), tgreply.IsPersistent()).
    Row().Button("Помощь", bot, telegram.MatchTypeExact, b.handleStartMenu).
    Row().Button("Добавить отслеживание", bot, telegram.MatchTypeExact, b.handleTrackingInsertMenu).
    Row().Button("Мои отслеживания", bot, telegram.MatchTypeExact, nil).
    Row().Button("Удалить отслеживание", bot, telegram.MatchTypeExact, nil)

  _, err := bot.SendMessage(ctx, &telegram.SendMessageParams{
    ChatID:      chatID,
    Text:        text,
    ParseMode:   tgmodels.ParseModeMarkdown,
    ReplyMarkup: reply,
    LinkPreviewOptions: &tgmodels.LinkPreviewOptions{
      IsDisabled: lo.ToPtr(true),
    },
  })
  if err != nil {
    log.Errorf("telegram.handleStartSilentMenu: bot.SendMessage: %v", err)
    return
  }

  err = b.upsertSession(ctx, models.Session{
    Telegram: models.Telegram{
      ChatID: chatID,
    },
    Message: models.SessionMessage{
      Menu:      models.StartSilentMenu,
      CreatedAt: time.Now(),
    },
  })
  if err != nil {
    log.Errorf("telegram.handleStartSilentMenu: b.upsertSession: %v", err)
  }
}

func (b *Bot) handleTrackingInsertMenu(ctx context.Context, bot *telegram.Bot, update *tgmodels.Update) {
  text := telegram.EscapeMarkdown(`Для добавления отслеживания введите следующие данные:
1. Ссылка на карточку товара
2. Интересующие размеры через запятую (в точности так, как указано на сайте)
3. Размер персональной скидки в процентах (если она есть)

Пример ввода данных:
1. https://www.lamoda.ru/p/rtlacv500501/clothes-carharttwip-dzhinsy/
2. 46/48, M, XL, 56/54
3. 7

Если выбранный вами товар one size или не имеет размерной сетки:
1. https://www.lamoda.ru/p/rtlacv500501/clothes-carharttwip-dzhinsy/
2. -
3. 7

Если у вас отсутствует персональная скидка:
1. https://www.lamoda.ru/p/rtlacv500501/clothes-carharttwip-dzhinsy/
2. 46/48, M, XL, 56/54
3. -

Бот проверит введенные вами данные и в тестовом режиме проверит карточку товара.
Вы получите ответное сообщение, в котором будет информация об отслеживаемом товаре.
Если все хорошо, "Подтвердите" отслеживание с помощью виртуальной клавиатуры.
Если что-то не так или вы передумали отслеживать товар, нажмите клавишу "Отмены".
 
Если вы передумали, нажмите клавишу "Назад", на виртуальной клавиатуре,
чтобы вернуться в главное меню бота.
`)

  chatID, ok := findChatID(update)
  if !ok {
    return
  }

  reply := tgreply.New(tgreply.WithPrefix("tracking"), tgreply.IsOneTimeKeyboard(), tgreply.ResizableKeyboard()).
    Row().Button("Назад", bot, telegram.MatchTypeExact, b.handleStartSilentMenu)

  _, err := bot.SendMessage(ctx, &telegram.SendMessageParams{
    ChatID:      chatID,
    Text:        text,
    ParseMode:   tgmodels.ParseModeMarkdown,
    ReplyMarkup: reply,
    LinkPreviewOptions: &tgmodels.LinkPreviewOptions{
      IsDisabled: lo.ToPtr(true),
    },
  })

  if err != nil {
    log.Errorf("telegram.handleTrackingInsertMenu: bot.SendMessage: %v", err)
  }

  err = b.upsertSession(ctx, models.Session{
    Telegram: models.Telegram{
      ChatID: chatID,
    },
    Message: models.SessionMessage{
      Menu:      models.TrackingInsertMenu,
      CreatedAt: time.Now(),
    },
  })
  if err != nil {
    log.Errorf("telegram.handleTrackingInsertMenu: b.upsertSession: %v", err)
  }
}

func (b *Bot) handleTrackingFieldsMenu(ctx context.Context, bot *telegram.Bot, update *tgmodels.Update) {
  chatID, ok := findChatID(update)
  if !ok {
    return
  }

  parseResult := parseTracking(chatID, update.Message.Text)

  if parseResult.ErrorMessage != "" {
    _, err := bot.SendMessage(ctx, &telegram.SendMessageParams{
      ChatID:    chatID,
      Text:      parseResult.ErrorMessage,
      ParseMode: tgmodels.ParseModeMarkdown,
      LinkPreviewOptions: &tgmodels.LinkPreviewOptions{
        IsDisabled: lo.ToPtr(true),
      },
    })
    if err != nil {
      log.Errorf("telegram.handleTrackingInsertMenu: bot.SendMessage: %v", err)
    }
    return
  }

  text := telegram.EscapeMarkdown(`Мы получили введенные вами данные. 
Сейчас бот проверит данные по карточке товара и вернется с результатом.
`)

  _, err := bot.SendMessage(ctx, &telegram.SendMessageParams{
    ChatID:    chatID,
    Text:      text,
    ParseMode: tgmodels.ParseModeMarkdown,
    LinkPreviewOptions: &tgmodels.LinkPreviewOptions{
      IsDisabled: lo.ToPtr(true),
    },
  })
  if err != nil {
    log.Errorf("telegram.handleTrackingInsertMenu: bot.SendMessage: %v", err)
    return
  }

  trackingMessage, err := b.deps.Tracker.CreateTrackingMessage(ctx, parseResult.Tracking)
  if err != nil {
    if errors.Is(err, tracker.ErrUnsupportedProductType) {

      _, err = bot.SendMessage(ctx, &telegram.SendMessageParams{
        ChatID:    chatID,
        Text:      telegram.EscapeMarkdown(`Извините, наш бот пока не умеет работать с данным сайтом.`),
        ParseMode: tgmodels.ParseModeMarkdown,
        LinkPreviewOptions: &tgmodels.LinkPreviewOptions{
          IsDisabled: lo.ToPtr(true),
        },
      })
      if err != nil {
        log.Errorf("telegram.handleTrackingInsertMenu: bot.SendMessage: %v", err)
      }
      return
    }

    log.Errorf("telegram.handleTrackingInsertMenu: b.deps.Tracker.CreateTrackingMessage: %v", err)
    return
  }

  parseResult.Tracking.Product = &trackingMessage.Product

  _, err = bot.SendMessage(ctx, &telegram.SendMessageParams{
    ChatID:    chatID,
    Text:      telegram.EscapeMarkdown(trackingMessage.TextValue),
    ParseMode: tgmodels.ParseModeMarkdown,
    LinkPreviewOptions: &tgmodels.LinkPreviewOptions{
      IsDisabled: lo.ToPtr(true),
    },
  })
  if err != nil {
    log.Errorf("telegram.handleTrackingInsertMenu: bot.SendMessage: %v", err)
    return
  }

  text = telegram.EscapeMarkdown(`Проверьте полученные от бота данные.
Если все хорошо, подтвердите отслеживание нажав "Подтвердить" на виртуальной клавиатуре.
Если вы передумали или хотите вернуться в главное меню - нажмите "Отменить".
`)

  reply := tgreply.New(tgreply.WithPrefix("tracking"), tgreply.IsOneTimeKeyboard(), tgreply.ResizableKeyboard()).
    Row().Button("Подтвердить", bot, telegram.MatchTypeExact, b.handleTrackingConfirmMenu).
    Row().Button("Отменить", bot, telegram.MatchTypeExact, b.handleStartSilentMenu)

  _, err = bot.SendMessage(ctx, &telegram.SendMessageParams{
    ChatID:      chatID,
    Text:        text,
    ParseMode:   tgmodels.ParseModeMarkdown,
    ReplyMarkup: reply,
    LinkPreviewOptions: &tgmodels.LinkPreviewOptions{
      IsDisabled: lo.ToPtr(true),
    },
  })
  if err != nil {
    log.Errorf("telegram.handleTrackingInsertMenu: bot.SendMessage: %v", err)
    return
  }

  err = b.upsertSession(ctx, models.Session{
    Telegram: models.Telegram{
      ChatID: chatID,
    },
    Message: models.SessionMessage{
      Menu:      models.TrackingFieldsMenu,
      CreatedAt: time.Now(),
    },
    Tracking: parseResult.Tracking,
  })
  if err != nil {
    log.Errorf("telegram.handleStartSilentMenu: b.upsertSession: %v", err)
  }
}

func (b *Bot) handleTrackingConfirmMenu(ctx context.Context, bot *telegram.Bot, update *tgmodels.Update) {
  chatID, ok := findChatID(update)
  if !ok {
    return
  }

  session, err := b.findSession(ctx, models.Telegram{
    ChatID: chatID,
  })
  if err != nil {
    log.Errorf("telegram.handleTrackingConfirmMenu: b.findSession: %v", err)
    return
  }

  if session.Message.Menu != models.TrackingFieldsMenu {
    return
  }
  if session.Tracking == nil {
    return
  }

  err = b.insertTracking(ctx, *session.Tracking)
  if err != nil {
    log.Errorf("telegram.handleTrackingConfirmMenu: b.insertTracking: %v", err)
    return
  }

  text := telegram.EscapeMarkdown(`Отслеживание для товара успешно создано.
Мы пришлем вам сообщение, как только получим новости по товару!
`)

  reply := tgreply.New(tgreply.WithPrefix("start"), tgreply.IsOneTimeKeyboard(), tgreply.ResizableKeyboard()).
    Row().Button("Помощь", bot, telegram.MatchTypeExact, b.handleStartMenu).
    Row().Button("Добавить отслеживание", bot, telegram.MatchTypeExact, b.handleTrackingInsertMenu).
    Row().Button("Мои отслеживания", bot, telegram.MatchTypeExact, nil).
    Row().Button("Удалить отслеживание", bot, telegram.MatchTypeExact, nil)

  _, err = bot.SendMessage(ctx, &telegram.SendMessageParams{
    ChatID:      session.Telegram.ChatID,
    Text:        text,
    ParseMode:   tgmodels.ParseModeMarkdown,
    ReplyMarkup: reply,
    LinkPreviewOptions: &tgmodels.LinkPreviewOptions{
      IsDisabled: lo.ToPtr(true),
    },
  })
  if err != nil {
    log.Errorf("telegram.handleTrackingConfirmMenu: bot.SendMessage: %v", err)
    return
  }

  err = b.upsertSession(ctx, models.Session{
    Telegram: models.Telegram{
      ChatID: chatID,
    },
    Message: models.SessionMessage{
      Menu:      models.TrackingConfirmMenu,
      CreatedAt: time.Now(),
    },
    Tracking: session.Tracking,
  })
  if err != nil {
    log.Errorf("telegram.handleTrackingConfirmMenu: b.upsertSession: %v", err)
  }
}

type parseTrackingResult struct {
  Tracking     *models.Tracking
  ErrorMessage string
}

func parseTracking(chatID int64, fields string) (res parseTrackingResult) {
  urlString := regexp.MustCompile(`1\..+\s?2\.`).FindString(fields)
  urlString = strings.Trim(urlString, "1.")
  urlString = strings.Trim(urlString, "2.")
  urlString = strings.TrimSpace(urlString)

  if urlString == "" {
    res.ErrorMessage = telegram.EscapeMarkdown(`Не удалось найти ссылку на товар в сообщении.

Пример ввода данных:
1. https://www.lamoda.ru/p/rtlacv500501/clothes-carharttwip-dzhinsy/
2. 46/48, M, XL, 56/54
3. 7

 Попробуйте еще раз.
`)

    return res
  }

  err := validator.URL(urlString)
  if err != nil {
    res.ErrorMessage = telegram.EscapeMarkdown(`Кажется, введенная вами ссылка имеет неверный формат.
Пример корректной ссылки: https://www.lamoda.ru/p/rtlacv500501/clothes-carharttwip-dzhinsy/

 Попробуйте еще раз.
`)

    return res
  }

  sizesString := regexp.MustCompile(`2\..+\s?3\.`).FindString(fields)
  sizesString = strings.Trim(sizesString, "2.")
  sizesString = strings.Trim(sizesString, "3.")
  sizesString = strings.TrimSpace(sizesString)

  if sizesString == "" {
    res.ErrorMessage = telegram.EscapeMarkdown(`Не удалось найти список размеров для товара.

Пример ввода данных:
1. https://www.lamoda.ru/p/rtlacv500501/clothes-carharttwip-dzhinsy/
2. 46/48, M, XL, 56/54
3. 7

 Попробуйте еще раз.
`)

    return res
  }

  sizesSlice := strings.Split(sizesString, ",")
  if len(sizesSlice) == 0 {
    res.ErrorMessage = telegram.EscapeMarkdown(`Не удалось найти список размеров для товара.

Пример ввода данных:
1. https://www.lamoda.ru/p/rtlacv500501/clothes-carharttwip-dzhinsy/
2. 46/48, M, XL, 56/54
3. 7

 Попробуйте еще раз.
`)

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
    res.ErrorMessage = telegram.EscapeMarkdown(`Не удалось найти поле с указанием персональной скидки.

Пример ввода данных:
1. https://www.lamoda.ru/p/rtlacv500501/clothes-carharttwip-dzhinsy/
2. 46/48, M, XL, 56/54
3. 7

Если у вас отсутствует персональная скидка:
1. https://www.lamoda.ru/p/rtlacv500501/clothes-carharttwip-dzhinsy/
2. 46/48, M, XL, 56/54
3. -

 Попробуйте еще раз.
`)

    return res
  }

  var discountParams *models.ParseDiscountParams

  if discountString != "-" {
    discountString = strings.Trim(discountString, "%s")

    discountInt, castErr := cast.ToInt64E(discountString)
    if castErr != nil {
      res.ErrorMessage = telegram.EscapeMarkdown(`Кажется, число, указанное в размере скидки некорректное. 

Пример ввода данных:
1. https://www.lamoda.ru/p/rtlacv500501/clothes-carharttwip-dzhinsy/
2. 46/48, M, XL, 56/54
3. 7

 Попробуйте еще раз.
`)

      return res
    }

    discountParams = &models.ParseDiscountParams{
      Percent: discountInt,
    }
  }

  return parseTrackingResult{
    Tracking: &models.Tracking{
      Telegram: models.Telegram{
        ChatID: chatID,
      },
      URL:      urlString,
      Sizes:    sizesParams,
      Discount: discountParams,
    },
  }
}

func (b *Bot) findSession(ctx context.Context, telegram models.Telegram) (*models.Session, error) {
  res, err := b.deps.Mongodb.Get(ctx, mongodb.GetParams{
    CommonParams: mongodb.CommonParams{
      Database:   "outfit",
      Collection: "sessions",
      StructType: models.Session{},
    },
    Filters: map[string]any{
      "telegram.chat_id": telegram.ChatID,
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

func (b *Bot) insertTracking(ctx context.Context, tracking models.Tracking) error {
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

func (b *Bot) upsertSession(ctx context.Context, session models.Session) error {
  _, err := b.deps.Mongodb.Upsert(ctx, mongodb.UpdateParams{
    GetParams: mongodb.GetParams{
      CommonParams: mongodb.CommonParams{
        Database:   "outfit",
        Collection: "sessions",
        StructType: session,
      },
      Filters: map[string]any{
        "telegram.chat_id": session.Telegram.ChatID,
      },
    },
    Document: session,
  })
  if err != nil {
    return fmt.Errorf("b.deps.Mongodb.Upsert: %w", err)
  }

  return nil
}

func findChatID(update *tgmodels.Update) (int64, bool) {
  if update != nil && update.Message != nil && update.Message.Chat.ID != 0 {
    return update.Message.Chat.ID, true
  }

  return 0, false
}
