package telegram

import (
  "context"
  "errors"
  "fmt"
  "strings"
  "time"

  telegram "github.com/go-telegram/bot"
  tgmodels "github.com/go-telegram/bot/models"
  "github.com/samber/lo"
  log "github.com/sirupsen/logrus"
  "github.com/ushakovn/outfit/internal/app/tracker"
  "github.com/ushakovn/outfit/internal/models"
)

func (b *Transport) handleStartMenu(ctx context.Context, bot *telegram.Bot, update *tgmodels.Update) {
  chatId, ok := findChatIdInUpdate(update)
  if !ok {
    log.
      WithField("update.message", update.Message).
      WithField("menu", models.StartMenu).
      Warn("chat_id not found")

    return
  }

  reply := newReplyKeyboard(models.StartMenu).
    Row().Button("Мои отслеживания ✉️", bot, telegram.MatchTypeExact, b.handleTrackingListMenu).
    Row().Button("Добавить отслеживание 📨", bot, telegram.MatchTypeExact, b.handleTrackingInsertMenu).
    Row().Button("Поддерживаемые магазины 👜", bot, telegram.MatchTypeExact, b.handleShopList).
    Row().Button("Обратная связь 📧", bot, telegram.MatchTypeExact, b.handleInsertIssueMenu)

  text := `<b>Бот создан для отслеживания товаров 💬</b>

<b>Он отсылает уведомления, когда:</b>
1. Цена на товар была снижена или появилась скидка на товар 📉
2. Распроданный товар снова появился в наличии 📦

<b>Опционально, бот может отсылать уведомления, когда</b>:
1. Цена на товар возросла 📈
2. Количество товара сократилось 📦

<b>Управление ботом происходит с помощью виртуальной клавиатуры 💡</b>`

  err := b.sendMessage(ctx, sendMessageParams{
    ChatId: chatId,
    Text:   text,
    Reply:  reply,
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.StartMenu).
      Errorf("b.sendMessage: %v", err)

    return
  }

  err = b.upsertSession(ctx, upsertSessionParams{
    ChatId: chatId,
    Menu:   models.StartMenu,
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.StartMenu).
      Errorf("b.upsertSession: %v", err)

    return
  }
}

func (b *Transport) handleStartSilentMenu(ctx context.Context, bot *telegram.Bot, update *tgmodels.Update) {
  chatId, ok := findChatIdInUpdate(update)
  if !ok {
    log.
      WithField("update.message", update.Message).
      WithField("menu", models.StartSilentMenu).
      Warn("chat_id not found")

    return
  }

  reply := newReplyKeyboard(models.StartSilentMenu).
    Row().Button("Помощь 💡", bot, telegram.MatchTypeExact, b.handleStartMenu).
    Row().Button("Мои отслеживания ✉️", bot, telegram.MatchTypeExact, b.handleTrackingListMenu).
    Row().Button("Добавить отслеживание 📨", bot, telegram.MatchTypeExact, b.handleTrackingInsertMenu)

  err := b.sendMessage(ctx, sendMessageParams{
    ChatId: chatId,
    Text:   `Вы вернулись в главное меню бота 💬`,
    Reply:  reply,
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.StartSilentMenu).
      Errorf("b.sendMessage: %v", err)

    return
  }

  err = b.upsertSession(ctx, upsertSessionParams{
    ChatId: chatId,
    Menu:   models.StartSilentMenu,
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.StartSilentMenu).
      Errorf("b.upsertSession: %v", err)

    return
  }
}

func (b *Transport) handleTrackingInsertMenu(ctx context.Context, bot *telegram.Bot, update *tgmodels.Update) {
  chatId, ok := findChatIdInUpdate(update)
  if !ok {
    log.
      WithField("update.message", update.Message).
      WithField("menu", models.TrackingInsertMenu).
      Warn("chat_id not found")

    return
  }

  reply := newReplyKeyboard(models.TrackingInsertMenu).
    Row().Button("Назад", bot, telegram.MatchTypeExact, b.handleStartSilentMenu)

  err := b.sendMessage(ctx, sendMessageParams{
    ChatId: chatId,
    Text:   `Введите ссылку на товар 📦`,
    Reply:  reply,
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingInsertMenu).
      Errorf("b.sendMessage: %v", err)

    return
  }

  err = b.upsertSession(ctx, upsertSessionParams{
    ChatId: chatId,
    Menu:   models.TrackingInsertMenu,
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingInsertMenu).
      Errorf("b.upsertSession: %v", err)

    return
  }
}

func (b *Transport) handleTrackingInputUrlMenu(ctx context.Context, bot *telegram.Bot, update *tgmodels.Update) {
  chatId, ok := findChatIdInUpdate(update)
  if !ok {
    log.
      WithField("update.message", update.Message).
      WithField("menu", models.TrackingInputUrlMenu).
      Warn("chat_id not found")

    return
  }

  reply := newReplyKeyboard(models.TrackingInputUrlMenu).
    Row().Button("Назад", bot, telegram.MatchTypeExact, b.handleStartSilentMenu)

  parsedUrl, errMessage := parseTrackingURL(update.Message.Text)

  if errMessage != "" {
    err := b.sendMessage(ctx, sendMessageParams{
      ChatId: chatId,
      Text:   errMessage,
      Reply:  reply,
    })
    if err != nil {
      log.
        WithField("chat_id", chatId).
        WithField("menu", models.TrackingInputUrlMenu).
        Errorf("b.sendMessage: %v", err)
    }
    return
  }

  tracking, err := b.findTracking(ctx, chatId, parsedUrl)
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingInputUrlMenu).
      Errorf("b.findTracking: %v", err)

    return
  }

  if tracking != nil {
    err = b.sendMessage(ctx, sendMessageParams{
      ChatId: chatId,
      Text: `Отслеживание по данному товару уже существует ✉️
Вы можете удалить его и создать новое с необходимыми параметрами 😉`,
      Reply: reply,
    })
    if err != nil {
      log.
        WithField("chat_id", chatId).
        WithField("menu", models.TrackingInputUrlMenu).
        Errorf("b.sendMessage: %v", err)
    }
    return
  }

  if err = b.checkProductURL(parsedUrl); err != nil {
    if errors.Is(err, tracker.ErrUnsupportedProductType) {
      err = b.sendMessage(ctx, sendMessageParams{
        ChatId: chatId,
        Text:   `Извините, бот пока не умеет работать с данным сайтом 😟`,
        Reply:  reply,
      })
      if err != nil {
        log.
          WithField("chat_id", chatId).
          WithField("menu", models.TrackingInputUrlMenu).
          Errorf("b.sendMessage: %v", err)
      }
      return
    }

    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingInputUrlMenu).
      Errorf("checkProductURL: %v", err)

    return
  }

  err = b.sendMessage(ctx, sendMessageParams{
    ChatId: chatId,
    Text:   `Сейчас бот проверит карточку товара и вернется 💬`,
    Reply:  reply,
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingInputUrlMenu).
      Errorf("b.sendMessage: %v", err)

    return
  }

  message, err := b.createMessage(ctx, parsedUrl)
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingInputUrlMenu).
      Errorf("b.createMessage: %v", err)

    b.sendErrorMessage(ctx, sendErrorMessageParams{
      ChatId: chatId,
      Text: `<b>Бот не смог получить данные 😟</b>

Убедитесь, что страница точно указывает на карточку товара
Если все верно, и ошибка повторится снова, обратитесь в поддержку 👨‍💻
`,
      Menu: models.TrackingInputUrlMenu,
    })

    return
  }

  err = b.sendMessage(ctx, sendMessageParams{
    ChatId: chatId,
    Text:   message.Text.Value,
    Reply:  reply,
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingInputUrlMenu).
      Errorf("b.sendMessage: %v", err)

    return
  }

  sizesValues := lo.Map(message.Product.Options, func(option models.ProductOption, _ int) string {
    return option.Size.Base.Value
  })
  sizesCount := len(message.Product.Options)

  // Если товар имеет one size размер.
  if sizesCount <= 1 {
    reply = newReplyKeyboard(models.TrackingInputUrlMenu).
      Row().Button("Далее", bot, telegram.MatchTypeExact, b.handleTrackingInputFlagMenu).
      Row().Button("Назад", bot, telegram.MatchTypeExact, b.handleStartSilentMenu)

    err = b.sendMessage(ctx, sendMessageParams{
      ChatId: chatId,
      Text: `<b>Проверьте полученные от бота данные</b>
Нажмите далее, если все хорошо 😉`,
      Reply: reply,
    })
    if err != nil {
      log.
        WithField("chat_id", chatId).
        WithField("menu", models.TrackingInputUrlMenu).
        Errorf("b.sendMessage: %v", err)
    }

    // Если товар имеет нормальную размерную сетку.
  } else {
    text := `<b>Проверьте полученные от бота данные</b>

Если все хорошо, выберите необходимые размеры из списка

Доступные размеры: 
`
    sizesString := strings.Join(sizesValues, ", ")
    text += strings.TrimSpace(sizesString)

    text += `

Размеры необходимо вводить через запятую, в точности так, как указано в списке

Кстати, вы можете ввести размер, которого нет в списке, если точно знаете, что такой существует и может появиться в наличии на сайте 😉

Пример корректного ввода 💬
`

    text += makeCutSizeValuesString(sizesValues)

    err = b.sendMessage(ctx, sendMessageParams{
      ChatId: chatId,
      Text:   text,
      Reply:  reply,
    })
    if err != nil {
      log.
        WithField("chat_id", chatId).
        WithField("menu", models.TrackingInputUrlMenu).
        Errorf("b.sendMessage: %v", err)

      return
    }
  }

  err = b.upsertSession(ctx, upsertSessionParams{
    ChatId: chatId,
    Menu:   models.TrackingInputUrlMenu,
    Tracking: &models.Tracking{
      ChatId: chatId,
      URL:    parsedUrl,
      Sizes: models.ParseSizesParams{
        Values: sizesValues,
      },
      ParsedProduct: message.Product,
      Timestamps: models.TrackingTimestamps{
        CreatedAt: time.Now(),
      },
    },
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingInputUrlMenu).
      Errorf("b.upsertSession: %v", err)

    return
  }
}

func (b *Transport) handleTrackingInputSizesMenu(ctx context.Context, bot *telegram.Bot, update *tgmodels.Update) {
  chatId, ok := findChatIdInUpdate(update)
  if !ok {
    log.
      WithField("update.message", update.Message).
      WithField("menu", models.TrackingInputSizesMenu).
      Warn("chat_id not found")

    return
  }

  session, err := b.findSession(ctx, chatId)
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingInputSizesMenu).
      Errorf("b.findSession: %v", err)

    return
  }

  reply := newReplyKeyboard(models.TrackingInputSizesMenu).
    Row().Button("Назад", bot, telegram.MatchTypeExact, b.handleStartSilentMenu)

  sizesValues, errMessage := parseTrackingSizes(update.Message.Text, session)

  if errMessage != "" {
    err = b.sendMessage(ctx, sendMessageParams{
      ChatId: chatId,
      Text:   errMessage,
      Reply:  reply,
    })
    if err != nil {
      log.
        WithField("chat_id", chatId).
        WithField("menu", models.TrackingInputSizesMenu).
        Errorf("b.sendMessage: %v", err)
    }
    return
  }

  setTrackingSizes(session.Tracking, sizesValues)

  err = b.upsertSession(ctx, upsertSessionParams{
    ChatId:   chatId,
    Menu:     models.TrackingInputSizesMenu,
    Tracking: session.Tracking,
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingInputSizesMenu).
      Errorf("b.upsertSession: %v", err)

    return
  }

  reply = newReplyKeyboard(models.TrackingInputSizesMenu).
    Row().Button("Далее", bot, telegram.MatchTypeExact, b.handleTrackingInputFlagMenu).
    Row().Button("Назад", bot, telegram.MatchTypeExact, b.handleStartSilentMenu)

  sizesString := strings.Join(sizesValues, ", ")
  sizesString = strings.TrimSpace(sizesString)

  text := fmt.Sprintf(`Введенные вами размеры: %s
`, sizesString)

  text += `Если все верно, нажмите далее 😉`

  err = b.sendMessage(ctx, sendMessageParams{
    ChatId: chatId,
    Text:   text,
    Reply:  reply,
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingInputSizesMenu).
      Errorf("b.sendMessage: %v", err)

    return
  }
}

func (b *Transport) handleTrackingInputFlagMenu(ctx context.Context, bot *telegram.Bot, update *tgmodels.Update) {
  chatId, ok := findChatIdInUpdate(update)
  if !ok {
    log.
      WithField("update.message", update.Message).
      WithField("menu", models.TrackingInputFlagMenu).
      Warn("chat_id not found")

    return
  }

  session, err := b.findSession(ctx, chatId)
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingInputFlagMenu).
      Errorf("b.findSession: %v", err)

    return
  }

  reply := newReplyKeyboard(models.TrackingInputFlagMenu).
    Row().Button("Включить️", bot, telegram.MatchTypeExact, b.handleTrackingFlagOnMenu).
    Row().Button("Пропустить️", bot, telegram.MatchTypeExact, b.handleTrackingFlagOffMenu).
    Row().Button("Назад", bot, telegram.MatchTypeExact, b.handleStartSilentMenu)

  err = b.sendMessage(ctx, sendMessageParams{
    ChatId: chatId,
    Text: `<b>Бот отсылает уведомления, когда:</b>
1. Цена на товар была снижена или появилась скидка на товар 📉
2. Распроданный товар снова появился в наличии 📦

<b>Опционально, бот может отсылать уведомления, когда</b>:
1. Цена на товар возросла 📈
2. Количество товара сократилось 📦

Включить опциональные уведомления?`,
    Reply: reply,
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingInputFlagMenu).
      Errorf("b.sendMessage: %v", err)

    return
  }

  err = b.upsertSession(ctx, upsertSessionParams{
    ChatId:   chatId,
    Menu:     models.TrackingInputFlagMenu,
    Tracking: session.Tracking,
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingInputFlagMenu).
      Errorf("b.upsertSession: %v", err)

    return
  }
}

func (b *Transport) handleTrackingCommentMenu(ctx context.Context, bot *telegram.Bot, update *tgmodels.Update) {
  chatId, ok := findChatIdInUpdate(update)
  if !ok {
    log.
      WithField("update.message", update.Message).
      WithField("menu", models.TrackingCommentMenu).
      Warn("chat_id not found")

    return
  }

  session, err := b.findSession(ctx, chatId)
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingCommentMenu).
      Errorf("b.findSession: %v", err)

    return
  }

  reply := newReplyKeyboard(models.TrackingCommentMenu).
    Row().Button("Подтвердить 📨", bot, telegram.MatchTypeExact, b.handleTrackingInsertConfirmMenu).
    Row().Button("Назад", bot, telegram.MatchTypeExact, b.handleStartSilentMenu)

  err = b.sendMessage(ctx, sendMessageParams{
    ChatId: chatId,
    Text: `Пример ввода 💬

Кепка Stussy черная 
#stussy #кепка #kixbox

Длина комментария может быть до 100 символов`,
    Reply: reply,
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingCommentMenu).
      Errorf("b.sendMessage: %v", err)
  }

  err = b.upsertSession(ctx, upsertSessionParams{
    ChatId:   chatId,
    Menu:     models.TrackingCommentMenu,
    Tracking: session.Tracking,
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingCommentMenu).
      Errorf("b.upsertSession: %v", err)

    return
  }
}

func (b *Transport) handleTrackingInputCommentMenu(ctx context.Context, bot *telegram.Bot, update *tgmodels.Update) {
  chatId, ok := findChatIdInUpdate(update)
  if !ok {
    log.
      WithField("update.message", update.Message).
      WithField("menu", models.TrackingInputCommentMenu).
      Warn("chat_id not found")

    return
  }

  session, err := b.findSession(ctx, chatId)
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingInputCommentMenu).
      Errorf("b.findSession: %v", err)

    return
  }

  reply := newReplyKeyboard(models.TrackingCommentMenu).
    Row().Button("Подтвердить 📨", bot, telegram.MatchTypeExact, b.handleTrackingInsertConfirmMenu).
    Row().Button("Назад", bot, telegram.MatchTypeExact, b.handleStartSilentMenu)

  parsedComment, errMessage := parseTrackingComment(update.Message.Text)

  if errMessage != "" {
    err = b.sendMessage(ctx, sendMessageParams{
      ChatId: chatId,
      Text:   errMessage,
      Reply:  reply,
    })
    if err != nil {
      log.
        WithField("chat_id", chatId).
        WithField("menu", models.TrackingInputCommentMenu).
        Errorf("b.sendMessage: %v", err)
    }
    return
  }

  session.Tracking.Comment = parsedComment

  err = b.upsertSession(ctx, upsertSessionParams{
    ChatId:   chatId,
    Menu:     models.TrackingInputCommentMenu,
    Tracking: session.Tracking,
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingInputCommentMenu).
      Errorf("b.upsertSession: %v", err)

    return
  }

  err = b.sendMessage(ctx, sendMessageParams{
    ChatId: chatId,
    Text: `Комментарий успешно сохранен 😉
Осталось подтвердить отслеживание 📨`,
    Reply: reply,
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingCommentMenu).
      Errorf("b.sendMessage: %v", err)
  }
}

func (b *Transport) handleTrackingFlagOnMenu(ctx context.Context, bot *telegram.Bot, update *tgmodels.Update) {
  chatId, ok := findChatIdInUpdate(update)
  if !ok {
    log.
      WithField("update.message", update.Message).
      WithField("menu", models.TrackingFlagConfirmMenu).
      Warn("chat_id not found")

    return
  }

  session, err := b.findSession(ctx, chatId)
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingFlagConfirmMenu).
      Errorf("b.findSession: %v", err)

    return
  }

  setTrackingFlag(session.Tracking, true)

  err = b.upsertSession(ctx, upsertSessionParams{
    ChatId:   chatId,
    Menu:     models.TrackingFlagConfirmMenu,
    Tracking: session.Tracking,
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingFlagConfirmMenu).
      Errorf("b.upsertSession: %v", err)

    return
  }

  reply := newReplyKeyboard(models.TrackingFlagConfirmMenu).
    Row().Button("Подтвердить 📨", bot, telegram.MatchTypeExact, b.handleTrackingInsertConfirmMenu).
    Row().Button("Комментарий 💬", bot, telegram.MatchTypeExact, b.handleTrackingCommentMenu).
    Row().Button("Назад", bot, telegram.MatchTypeExact, b.handleStartSilentMenu)

  err = b.sendMessage(ctx, sendMessageParams{
    ChatId: chatId,
    Text: `Опциональные уведомления включены 

Далее, вы можете оставить комментарий к вашему отслеживанию 💡
Он будет отображаться при просмотре списка отслеживаний и получении уведомлений по товару 💬

Если комментарий не требуется, просто подтвердите отслеживание 📨`,
    Reply: reply,
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingFlagConfirmMenu).
      Errorf("b.sendMessage: %v", err)
  }
}

func (b *Transport) handleTrackingFlagOffMenu(ctx context.Context, bot *telegram.Bot, update *tgmodels.Update) {
  chatId, ok := findChatIdInUpdate(update)
  if !ok {
    log.
      WithField("update.message", update.Message).
      WithField("menu", models.TrackingFlagConfirmMenu).
      Warn("chat_id not found")

    return
  }

  session, err := b.findSession(ctx, chatId)
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingFlagConfirmMenu).
      Errorf("b.findSession: %v", err)

    return
  }

  err = b.upsertSession(ctx, upsertSessionParams{
    ChatId:   chatId,
    Menu:     models.TrackingFlagConfirmMenu,
    Tracking: session.Tracking,
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingFlagConfirmMenu).
      Errorf("b.upsertSession: %v", err)

    return
  }

  reply := newReplyKeyboard(models.TrackingFlagConfirmMenu).
    Row().Button("Подтвердить 📨", bot, telegram.MatchTypeExact, b.handleTrackingInsertConfirmMenu).
    Row().Button("Комментарий 💬", bot, telegram.MatchTypeExact, b.handleTrackingCommentMenu).
    Row().Button("Назад", bot, telegram.MatchTypeExact, b.handleStartSilentMenu)

  err = b.sendMessage(ctx, sendMessageParams{
    ChatId: chatId,
    Text: `Опциональные уведомления отключены 

Далее, вы можете оставить комментарий к вашему отслеживанию 💡
Он будет отображаться при просмотре списка отслеживаний и получении уведомлений по товару 💬 

Если комментарий не требуется, просто подтвердите отслеживание 📨`,
    Reply: reply,
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingFlagConfirmMenu).
      Errorf("b.sendMessage: %v", err)
  }
}

func (b *Transport) handleTrackingInsertConfirmMenu(ctx context.Context, bot *telegram.Bot, update *tgmodels.Update) {
  chatId, ok := findChatIdInUpdate(update)
  if !ok {
    log.
      WithField("update.message", update.Message).
      WithField("menu", models.TrackingInsertConfirmMenu).
      Warn("chat_id not found")

    return
  }

  session, err := b.findSession(ctx, chatId)
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingInsertConfirmMenu).
      Errorf("b.findSession: %v", err)

    return
  }

  if session.Tracking == nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingInsertConfirmMenu).
      WithField("session.tracking", session.Tracking).
      Warn("message skipped")

    return
  }

  err = b.insertTracking(ctx, *session.Tracking)
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingInsertConfirmMenu).
      Errorf("b.insertTracking: %v", err)

    return
  }

  reply := newReplyKeyboard(models.TrackingInsertConfirmMenu).
    Row().Button("Помощь 💡", bot, telegram.MatchTypeExact, b.handleStartMenu).
    Row().Button("Мои отслеживания ✉️", bot, telegram.MatchTypeExact, b.handleTrackingListMenu).
    Row().Button("Добавить отслеживание 📨", bot, telegram.MatchTypeExact, b.handleTrackingInsertMenu)

  err = b.sendMessage(ctx, sendMessageParams{
    ChatId: chatId,
    Text: `Отслеживание для товара создано 😉
Мы пришлем уведомление, как только получим новости по товару 📦`,
    Reply: reply,
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingInsertConfirmMenu).
      Errorf("b.sendMessage: %v", err)

    return
  }

  err = b.upsertSession(ctx, upsertSessionParams{
    ChatId: chatId,
    Menu:   models.TrackingInsertConfirmMenu,
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingInsertConfirmMenu).
      Errorf("b.upsertSession: %v", err)

    return
  }
}

func (b *Transport) handleTrackingListMenu(ctx context.Context, bot *telegram.Bot, update *tgmodels.Update) {
  chatId, ok := findChatIdInUpdate(update)
  if !ok {
    log.
      WithField("update.message", update.Message).
      WithField("menu", models.TrackingListMenu).
      Warn("chat_id not found")

    return
  }

  list, err := b.listTrackings(ctx, chatId)
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingListMenu).
      Errorf("b.listTrackings: %v", err)

    return
  }

  for index, tracking := range list {
    key := chatSelectedTracking{
      ChatId: chatId,
      Index:  index,
    }
    b.deps.cache.TrackingURLs[key] = tracking.URL
  }

  if len(list) > 0 {
    slider := b.newTrackingSlider(trackingSliderParams{
      ChatId:    chatId,
      Bot:       bot,
      Trackings: list,
    })

    if _, err = slider.Show(ctx, bot, chatId); err != nil {
      log.
        WithField("chat_id", chatId).
        WithField("menu", models.TrackingListMenu).
        Errorf("telegram.Slider.Show: %v", err)

      return
    }
  } else {
    reply := newReplyKeyboard(models.TrackingListMenu).
      Row().Button("Помощь 💡", bot, telegram.MatchTypeExact, b.handleStartMenu).
      Row().Button("Мои отслеживания ✉️", bot, telegram.MatchTypeExact, b.handleTrackingListMenu).
      Row().Button("Добавить отслеживание 📨", bot, telegram.MatchTypeExact, b.handleTrackingInsertMenu)

    err = b.sendMessage(ctx, sendMessageParams{
      ChatId: chatId,
      Text:   `У вас пока нет отслеживаний 👀`,
      Reply:  reply,
    })
    if err != nil {
      log.
        WithField("chat_id", chatId).
        WithField("menu", models.TrackingListMenu).
        Errorf("b.sendMessage: %v", err)

      return
    }
  }

  err = b.upsertSession(ctx, upsertSessionParams{
    ChatId: chatId,
    Menu:   models.TrackingListMenu,
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingListMenu).
      Errorf("b.upsertSession: %v", err)

    return
  }
}

func (b *Transport) handleTrackingDeleteMenu(ctx context.Context, bot *telegram.Bot, message tgmodels.MaybeInaccessibleMessage, index int) {
  chatId, ok := findChatIdInMaybeInaccessible(message)
  if !ok {
    log.
      WithField("inaccessible_message", message).
      WithField("menu", models.TrackingDeleteMenu).
      Warn("chat_id not found")

    return
  }

  url, ok := b.deps.cache.TrackingURLs[chatSelectedTracking{
    ChatId: chatId,
    Index:  index,
  }]
  if !ok {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingDeleteMenu).
      WithField("tracking_index", index).
      Errorf("tracking url not found in cache")

    return
  }

  tracking, err := b.findTracking(ctx, chatId, url)
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingDeleteMenu).
      Errorf("b.findTracking: %v", err)

    return
  }

  reply := newReplyKeyboard(models.TrackingDeleteMenu).
    Row().Button("Подтвердить", bot, telegram.MatchTypeExact, b.handleTrackingDeleteConfirmMenu).
    Row().Button("Назад", bot, telegram.MatchTypeExact, b.handleStartSilentMenu)

  err = b.sendMessage(ctx, sendMessageParams{
    ChatId: chatId,
    Text:   `Вы уверены, что хотите удалить отслеживание? 🗑️`,
    Reply:  reply,
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingDeleteMenu).
      Errorf("b.sendMessage: %v", err)

    return
  }

  err = b.upsertSession(ctx, upsertSessionParams{
    ChatId:   chatId,
    Menu:     models.TrackingDeleteMenu,
    Tracking: tracking,
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingDeleteMenu).
      Errorf("b.upsertSession: %v", err)

    return
  }
}

func (b *Transport) handleTrackingSilentMenu(ctx context.Context, bot *telegram.Bot, message tgmodels.MaybeInaccessibleMessage) {
  chatId, ok := findChatIdInMaybeInaccessible(message)
  if !ok {
    log.
      WithField("inaccessible_message", message).
      WithField("menu", models.StartSilentMenu).
      Warn("chat_id not found")

    return
  }

  reply := newReplyKeyboard(models.StartSilentMenu).
    Row().Button("Помощь 💡", bot, telegram.MatchTypeExact, b.handleStartMenu).
    Row().Button("Мои отслеживания ✉️", bot, telegram.MatchTypeExact, b.handleTrackingListMenu).
    Row().Button("Добавить отслеживание 📨", bot, telegram.MatchTypeExact, b.handleTrackingInsertMenu)

  err := b.sendMessage(ctx, sendMessageParams{
    ChatId: chatId,
    Text:   `Вы вернулись в главное меню бота 💬`,
    Reply:  reply,
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.StartSilentMenu).
      Errorf("b.sendMessage: %v", err)

    return
  }

  err = b.upsertSession(ctx, upsertSessionParams{
    ChatId: chatId,
    Menu:   models.StartSilentMenu,
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.StartSilentMenu).
      Errorf("b.upsertSession: %v", err)

    return
  }
}

func (b *Transport) handleTrackingDeleteConfirmMenu(ctx context.Context, bot *telegram.Bot, update *tgmodels.Update) {
  chatId, ok := findChatIdInUpdate(update)
  if !ok {
    log.
      WithField("update.message", update.Message).
      WithField("menu", models.TrackingDeleteConfirmMenu).
      Warn("chat_id not found")

    return
  }

  session, err := b.findSession(ctx, chatId)
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingDeleteConfirmMenu).
      Errorf("b.findSession: %v", err)

    return
  }

  err = b.deleteTracking(ctx, session)
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingDeleteConfirmMenu).
      Errorf("b.deleteTracking: %v", err)

    return
  }

  reply := newReplyKeyboard(models.TrackingDeleteConfirmMenu).
    Row().Button("Помощь 💡", bot, telegram.MatchTypeExact, b.handleStartMenu).
    Row().Button("Мои отслеживания ✉️", bot, telegram.MatchTypeExact, b.handleTrackingListMenu).
    Row().Button("Добавить отслеживание 📨", bot, telegram.MatchTypeExact, b.handleTrackingInsertMenu)

  err = b.sendMessage(ctx, sendMessageParams{
    ChatId: chatId,
    Text:   `Отслеживание успешно удалено 😉`,
    Reply:  reply,
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingDeleteConfirmMenu).
      Errorf("b.sendMessage: %v", err)

    return
  }

  err = b.upsertSession(ctx, upsertSessionParams{
    ChatId: chatId,
    Menu:   models.TrackingDeleteConfirmMenu,
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingDeleteConfirmMenu).
      Errorf("b.upsertSession: %v", err)

    return
  }
}

func (b *Transport) handleShopList(ctx context.Context, bot *telegram.Bot, update *tgmodels.Update) {
  chatId, ok := findChatIdInUpdate(update)
  if !ok {
    log.
      WithField("update.message", update.Message).
      WithField("menu", models.TrackingDeleteConfirmMenu).
      Warn("chat_id not found")

    return
  }

  reply := newReplyKeyboard(models.ShopListMenu).
    Row().Button("Назад", bot, telegram.MatchTypeExact, b.handleStartSilentMenu)

  err := b.sendMessage(ctx, sendMessageParams{
    ChatId: chatId,
    Text: `Магазины, с которыми работает бот:
1. Lamoda
2. Lime
3. Kixbox
4. Ridestep
5. Октябрь Скейтшоп
Список постепенно будет пополняться 🤓`,
    Reply: reply,
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.TrackingDeleteConfirmMenu).
      Errorf("b.sendMessage: %v", err)

    return
  }

  err = b.upsertSession(ctx, upsertSessionParams{
    ChatId: chatId,
    Menu:   models.ShopListMenu,
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.ShopListMenu).
      Errorf("b.upsertSession: %v", err)

    return
  }
}

func (b *Transport) handleInsertIssueMenu(ctx context.Context, bot *telegram.Bot, update *tgmodels.Update) {
  chatId, ok := findChatIdInUpdate(update)
  if !ok {
    log.
      WithField("update.message", update.Message).
      WithField("menu", models.IssueInsertMenu).
      Warn("chat_id not found")

    return
  }

  reply := newReplyKeyboard(models.IssueInsertMenu).
    Row().Button("Улучшение 👨‍🔧", bot, telegram.MatchTypeExact, b.handleIssueInputStoryMenu).
    Row().Button("Баг 😟", bot, telegram.MatchTypeExact, b.handleIssueInputBugMenu).
    Row().Button("Назад", bot, telegram.MatchTypeExact, b.handleStartSilentMenu)

  err := b.sendMessage(ctx, sendMessageParams{
    ChatId: chatId,
    Text: `Выберите категорию обратной связи: 
баг или улучшение 😉`,
    Reply: reply,
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.IssueInsertMenu).
      Errorf("b.sendMessage: %v", err)

    return
  }
}

func (b *Transport) handleIssueInputTextMenu(ctx context.Context, bot *telegram.Bot, update *tgmodels.Update) {
  chatId, ok := findChatIdInUpdate(update)
  if !ok {
    log.
      WithField("update.message", update.Message).
      WithField("menu", models.IssueInputTextMenu).
      Warn("chat_id not found")

    return
  }

  session, err := b.findSession(ctx, chatId)
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.IssueInputTextMenu).
      Errorf("b.findSession: %v", err)

    return
  }

  session.Entities.Issue.Text = parseIssueText(update.Message.Text)

  err = b.upsertSession(ctx, upsertSessionParams{
    ChatId:   chatId,
    Menu:     models.IssueInputTextMenu,
    Entities: session.Entities,
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.IssueInputTextMenu).
      Errorf("b.upsertSession: %v", err)

    return
  }

  reply := newReplyKeyboard(models.IssueInputTextMenu).
    Row().Button("Подтвердить 📧", bot, telegram.MatchTypeExact, b.handleIssueInsertConfirmMenu).
    Row().Button("Назад", bot, telegram.MatchTypeExact, b.handleStartSilentMenu)

  err = b.sendMessage(ctx, sendMessageParams{
    ChatId: chatId,
    Text:   `Мы получили ваше сообщение, осталось его подтвердить 📧`,
    Reply:  reply,
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.IssueInsertMenu).
      Errorf("b.sendMessage: %v", err)

    return
  }
}

func (b *Transport) handleIssueInsertConfirmMenu(ctx context.Context, bot *telegram.Bot, update *tgmodels.Update) {
  chatId, ok := findChatIdInUpdate(update)
  if !ok {
    log.
      WithField("update.message", update.Message).
      WithField("menu", models.IssueInsertConfirmMenu).
      Warn("chat_id not found")

    return
  }

  session, err := b.findSession(ctx, chatId)
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.IssueInsertConfirmMenu).
      Errorf("b.findSession: %v", err)

    return
  }

  err = b.insertIssue(ctx, session.Entities.Issue)
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.IssueInsertConfirmMenu).
      Errorf("b.insertIssue: %v", err)

    return
  }

  reply := newReplyKeyboard(models.IssueInsertConfirmMenu).
    Row().Button("Назад", bot, telegram.MatchTypeExact, b.handleStartSilentMenu)

  err = b.sendMessage(ctx, sendMessageParams{
    ChatId: chatId,
    Text:   `Спасибо за вашу обратную связь 😉`,
    Reply:  reply,
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.IssueInsertConfirmMenu).
      Errorf("b.sendMessage: %v", err)

    return
  }
}

func (b *Transport) handleIssueInputBugMenu(ctx context.Context, bot *telegram.Bot, update *tgmodels.Update) {
  chatId, ok := findChatIdInUpdate(update)
  if !ok {
    log.
      WithField("update.message", update.Message).
      WithField("menu", models.IssueInputTypeMenu).
      Warn("chat_id not found")

    return
  }

  reply := newReplyKeyboard(models.IssueInputTypeMenu).
    Row().Button("Назад", bot, telegram.MatchTypeExact, b.handleStartSilentMenu)

  err := b.sendMessage(ctx, sendMessageParams{
    ChatId: chatId,
    Text:   `Опишите возникшую у вас проблему 💬`,
    Reply:  reply,
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.IssueInputTypeMenu).
      Errorf("b.sendMessage: %v", err)

    return
  }

  err = b.upsertSession(ctx, upsertSessionParams{
    ChatId: chatId,
    Menu:   models.IssueInputTypeMenu,
    Entities: &models.SessionEntities{
      Issue: &models.Issue{
        ChatId:    chatId,
        Type:      models.IssueTypeBug,
        CreatedAt: time.Now(),
      },
    },
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.IssueInputTypeMenu).
      Errorf("b.upsertSession: %v", err)

    return
  }
}

func (b *Transport) handleIssueInputStoryMenu(ctx context.Context, bot *telegram.Bot, update *tgmodels.Update) {
  chatId, ok := findChatIdInUpdate(update)
  if !ok {
    log.
      WithField("update.message", update.Message).
      WithField("menu", models.IssueInputTypeMenu).
      Warn("chat_id not found")

    return
  }

  reply := newReplyKeyboard(models.IssueInputTypeMenu).
    Row().Button("Назад", bot, telegram.MatchTypeExact, b.handleStartSilentMenu)

  err := b.sendMessage(ctx, sendMessageParams{
    ChatId: chatId,
    Text:   `Опишите улучшения бота, которые вам хотелось бы видеть 💬`,
    Reply:  reply,
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.IssueInputTypeMenu).
      Errorf("b.sendMessage: %v", err)

    return
  }

  err = b.upsertSession(ctx, upsertSessionParams{
    ChatId: chatId,
    Menu:   models.IssueInputTypeMenu,
    Entities: &models.SessionEntities{
      Issue: &models.Issue{
        ChatId:    chatId,
        Type:      models.IssueTypeStory,
        CreatedAt: time.Now(),
      },
    },
  })
  if err != nil {
    log.
      WithField("chat_id", chatId).
      WithField("menu", models.IssueInputTypeMenu).
      Errorf("b.upsertSession: %v", err)

    return
  }
}
