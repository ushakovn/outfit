package telegram

import (
	"context"
	"errors"

	telegram "github.com/go-telegram/bot"
	tgmodels "github.com/go-telegram/bot/models"
	log "github.com/sirupsen/logrus"
	"github.com/ushakovn/outfit/internal/models"
	"github.com/ushakovn/outfit/internal/provider/mongodb"
	"github.com/ushakovn/outfit/internal/tracker"
)

func (b *Bot) handleStartMenu(ctx context.Context, bot *telegram.Bot, update *tgmodels.Update) {
	chatId, ok := findChatIdInUpdate(update)
	if !ok {
		log.Warnf("telegram.handleStartMenu: findChatIdInUpdate: chat not found")
		return
	}

	reply := newReplyKeyboard("start").
		Row().Button("Добавить отслеживание", bot, telegram.MatchTypeExact, b.handleTrackingInsertMenu).
		Row().Button("Мои отслеживания", bot, telegram.MatchTypeExact, nil).
		Row().Button("Удалить отслеживание", bot, telegram.MatchTypeExact, nil)

	text := `Данный бот создан для отслеживания товаров. 
  
  Бот отсылает уведомления если:
  1. Цена на товар была снижена или появилась скидка на товар.
  2. Распроданный товар снова появился в наличие.
  
  Управление ботом происходит с помощью виртуальной клавиатуры, под сообщением:
  1. Добавить отслеживание - добавляет новое отслеживание по вашему товару
  2. Мои отслеживания - выводит список товаров, для которых подключено отслеживание
  3. Удалить отслеживание - удаляет созданное вами ранее отслеживание
  
  Воспользуйтесь виртуальной клавиатурой, чтобы продолжить.`

	err := b.sendMessage(ctx, sendMessageParams{
		ChatId: chatId,
		Text:   text,
		Reply:  reply,
	})
	if err != nil {
		log.Errorf("telegram.handleStartMenu: b.sendMessage: %v", err)
		return
	}

	err = b.upsertSession(ctx, upsertSessionParams{
		ChatId: chatId,
		Menu:   models.StartMenu,
	})
	if err != nil {
		log.Errorf("telegram.handleStartMenu: b.upsertSession: %v", err)
		return
	}
}

func (b *Bot) handleStartSilentMenu(ctx context.Context, bot *telegram.Bot, update *tgmodels.Update) {
	chatId, ok := findChatIdInUpdate(update)
	if !ok {
		log.Warnf("telegram.handleStartSilentMenu: findChatIdInUpdate: chat not found")
		return
	}

	reply := newReplyKeyboard("start").
		Row().Button("Помощь", bot, telegram.MatchTypeExact, b.handleStartMenu).
		Row().Button("Добавить отслеживание", bot, telegram.MatchTypeExact, b.handleTrackingInsertMenu).
		Row().Button("Мои отслеживания", bot, telegram.MatchTypeExact, nil).
		Row().Button("Удалить отслеживание", bot, telegram.MatchTypeExact, nil)

	err := b.sendMessage(ctx, sendMessageParams{
		ChatId: chatId,
		Text: `Вы вернулись в главное меню бота. 
Если вам требуется подсказка, нажмите клавишу "Помощь", на виртуальной клавиатуре или наберите /start.`,
		Reply: reply,
	})
	if err != nil {
		log.Errorf("telegram.handleStartSilentMenu: b.sendMessage: %v", err)
		return
	}

	err = b.upsertSession(ctx, upsertSessionParams{
		ChatId: chatId,
		Menu:   models.StartSilentMenu,
	})
	if err != nil {
		log.Errorf("telegram.handleStartSilentMenu: b.upsertSession: %v", err)
		return
	}
}

func (b *Bot) handleTrackingInsertMenu(ctx context.Context, bot *telegram.Bot, update *tgmodels.Update) {
	chatId, ok := findChatIdInUpdate(update)
	if !ok {
		log.Warnf("telegram.handleTrackingInsertMenu: findChatIdInUpdate: chat not found")
		return
	}

	text := `Для добавления отслеживания введите следующие данные:
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
чтобы вернуться в главное меню бота.`

	reply := newReplyKeyboard("tracking").
		Row().Button("Назад", bot, telegram.MatchTypeExact, b.handleStartSilentMenu)

	err := b.sendMessage(ctx, sendMessageParams{
		ChatId: chatId,
		Text:   text,
		Reply:  reply,
	})
	if err != nil {
		log.Errorf("telegram.handleTrackingInsertMenu: b.sendMessage: %v", err)
		return
	}

	err = b.upsertSession(ctx, upsertSessionParams{
		ChatId: chatId,
		Menu:   models.TrackingInsertMenu,
	})
	if err != nil {
		log.Errorf("telegram.handleTrackingInsertMenu: b.upsertSession: %v", err)
		return
	}
}

func (b *Bot) handleTrackingFieldsMenu(ctx context.Context, bot *telegram.Bot, update *tgmodels.Update) {
	chatId, ok := findChatIdInUpdate(update)
	if !ok {
		log.Warnf("telegram.handleTrackingFieldsMenu: findChatIdInUpdate: chat not found")
		return
	}

	parsedFields := parseTrackingFields(update.Message.Text)

	reply := newReplyKeyboard("tracking").
		Row().Button("Назад", bot, telegram.MatchTypeExact, b.handleStartSilentMenu)

	if parsedFields.ErrorMessage != "" {
		err := b.sendMessage(ctx, sendMessageParams{
			ChatId: chatId,
			Text:   parsedFields.ErrorMessage,
			Reply:  reply,
		})
		if err != nil {
			log.Errorf("telegram.handleTrackingFieldsMenu: b.sendMessage: %v", err)
		}
		log.Warnf("telegram.handleTrackingFieldsMenu: parseTrackingFields: %v", parsedFields.ErrorMessage)
		return
	}

	err := b.sendMessage(ctx, sendMessageParams{
		ChatId: chatId,
		Text: `Мы получили введенные вами данные. 
Сейчас бот проверит данные по карточке товара и вернется с результатом.`,
		Reply: reply,
	})
	if err != nil {
		log.Errorf("telegram.handleTrackingFieldsMenu: b.sendMessage: %v", err)
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
				Text:   `Извините, наш бот пока не умеет работать с данным сайтом.`,
				Reply:  reply,
			})
			if err != nil {
				log.Errorf("telegram.handleTrackingFieldsMenu: b.sendMessage: %v", err)
			}
			return
		}
		log.Errorf("telegram.handleTrackingFieldsMenu: b.deps.Tracker.CreateTrackingMessage: %v", err)
		return
	}

	err = b.sendMessage(ctx, sendMessageParams{
		ChatId: chatId,
		Text:   trackingMsg.TextValue,
		Reply:  reply,
	})
	if err != nil {
		log.Errorf("telegram.handleTrackingFieldsMenu: b.sendMessage: %v", err)
		return
	}

	reply = newReplyKeyboard("tracking").
		Row().Button("Подтвердить", bot, telegram.MatchTypeExact, b.handleTrackingConfirmMenu).
		Row().Button("Отменить", bot, telegram.MatchTypeExact, b.handleStartSilentMenu)

	err = b.sendMessage(ctx, sendMessageParams{
		ChatId: chatId,
		Text: `Проверьте полученные от бота данные.
Если все хорошо, подтвердите отслеживание нажав "Подтвердить" на виртуальной клавиатуре.
Если вы передумали или хотите вернуться в главное меню - нажмите "Отменить".`,
		Reply: reply,
	})
	if err != nil {
		log.Errorf("telegram.handleTrackingFieldsMenu: b.sendMessage: %v", err)
		return
	}

	err = b.upsertSession(ctx, upsertSessionParams{
		ChatId:   chatId,
		Menu:     models.TrackingFieldsMenu,
		Tracking: newTracking(chatId, parsedFields, *trackingMsg),
	})
	if err != nil {
		log.Errorf("telegram.handleTrackingFieldsMenu: b.upsertSession: %v", err)
		return
	}
}

func (b *Bot) handleTrackingConfirmMenu(ctx context.Context, bot *telegram.Bot, update *tgmodels.Update) {
	chatId, ok := findChatIdInUpdate(update)
	if !ok {
		log.Warnf("telegram.handleTrackingConfirmMenu: findChatIdInUpdate: chat not found")
		return
	}

	session, err := b.findSession(ctx, chatId)
	if err != nil {
		log.Errorf("telegram.handleTrackingConfirmMenu: b.findSession: %v", err)
		return
	}

	if session.Message.Menu != models.TrackingFieldsMenu || session.Tracking == nil {
		log.Warnf("telegram.handleTrackingConfirmMenu: skip. session.Message.Menu: %v. session.TrackingMessage: %v",
			session.Message.Menu, session.Tracking)
		return
	}

	err = b.insertTracking(ctx, *session.Tracking)
	if err != nil {
		log.Errorf("telegram.handleTrackingConfirmMenu: b.insertTracking: %v", err)
		return
	}

	reply := newReplyKeyboard("tracking").
		Row().Button("Помощь", bot, telegram.MatchTypeExact, b.handleStartMenu).
		Row().Button("Добавить отслеживание", bot, telegram.MatchTypeExact, b.handleTrackingInsertMenu).
		Row().Button("Мои отслеживания", bot, telegram.MatchTypeExact, b.handleTrackingListMenu)

	err = b.sendMessage(ctx, sendMessageParams{
		ChatId: chatId,
		Text: `Отслеживание для товара успешно создано.
Мы пришлем вам сообщение, как только получим новости по товару!`,
		Reply: reply,
	})
	if err != nil {
		log.Errorf("telegram.handleTrackingConfirmMenu: b.sendMessage: %v", err)
		return
	}

	err = b.upsertSession(ctx, upsertSessionParams{
		ChatId: chatId,
		Menu:   models.TrackingConfirmMenu,
	})
	if err != nil {
		log.Errorf("telegram.handleTrackingConfirmMenu: b.upsertSession: %v", err)
		return
	}
}

func (b *Bot) handleTrackingListMenu(ctx context.Context, bot *telegram.Bot, update *tgmodels.Update) {
	chatId, ok := findChatIdInUpdate(update)
	if !ok {
		log.Warnf("telegram.handleTrackingListMenu: findChatIdInUpdate: chat not found")
		return
	}

	list, err := b.listTrackings(ctx, chatId)
	if err != nil {
		log.Errorf("telegram.handleTrackingListMenu: b.listTrackings: %v", err)
		return
	}

	for index, tracking := range list {
		key := chatSelectedTracking{
			ChatId: chatId,
			Index:  index,
		}
		b.deps.Cache.TrackingURLs[key] = tracking.URL
	}

	slider := b.newTrackingSlider(trackingSliderParams{
		Bot:       bot,
		Trackings: list,
	})

	if _, err = slider.Show(ctx, bot, chatId); err != nil {
		log.Errorf("telegram.handleTrackingListMenu: telegram.Slider.Show: %v", err)
		return
	}

	err = b.upsertSession(ctx, upsertSessionParams{
		ChatId: chatId,
		Menu:   models.TrackingListMenu,
	})
	if err != nil {
		log.Errorf("telegram.handleTrackingListMenu: b.upsertSession: %v", err)
		return
	}
}

func (b *Bot) handleTrackingSelectDeleteMenu(ctx context.Context, bot *telegram.Bot, message tgmodels.MaybeInaccessibleMessage, index int) {
	chatId, ok := findChatIdInMaybeInaccessible(message)
	if !ok {
		log.Warnf("telegram.handleTrackingSelectDeleteMenu: findChatIdInMaybeInaccessible: chat not found")
		return
	}

	url, ok := b.deps.Cache.TrackingURLs[chatSelectedTracking{
		ChatId: chatId,
		Index:  index,
	}]
	if !ok {
		log.Errorf("telegram.handleTrackingSelectDeleteMenu: b.deps.Cache.TrackingURLs: not found")
		return
	}

	tracking, err := b.findTracking(ctx, chatId, url)
	if err != nil {
		log.Errorf("telegram.handleTrackingSelectDeleteMenu: b.findTracking: %v", err)
		return
	}

	reply := newReplyKeyboard("tracking").
		Row().Button("Назад", bot, telegram.MatchTypeExact, b.handleTrackingListMenu)

	err = b.sendMessage(ctx, sendMessageParams{
		ChatId: chatId,
		Text:   `Вы уверены, что хотите удалить отслеживание?`,
		Reply:  reply,
	})
	if err != nil {
		log.Errorf("telegram.handleTrackingSelectDeleteMenu: b.sendMessage: %v", err)
		return
	}

	err = b.upsertSession(ctx, upsertSessionParams{
		ChatId:   chatId,
		Menu:     models.TrackingSelectDeleteMenu,
		Tracking: tracking,
	})
	if err != nil {
		log.Errorf("telegram.handleTrackingListMenu: b.upsertSession: %v", err)
		return
	}
}

func (b *Bot) handleTrackingSelectListMenu(ctx context.Context, bot *telegram.Bot, message tgmodels.MaybeInaccessibleMessage) {
	chatId, ok := findChatIdInMaybeInaccessible(message)
	if !ok {
		log.Warnf("telegram.handleTrackingSelectListMenu: findChatIdInUpdate: chat not found")
		return
	}

	list, err := b.listTrackings(ctx, chatId)
	if err != nil {
		log.Errorf("telegram.handleTrackingSelectListMenu: b.listTrackings: %v", err)
		return
	}

	for index, tracking := range list {
		key := chatSelectedTracking{
			ChatId: chatId,
			Index:  index,
		}
		b.deps.Cache.TrackingURLs[key] = tracking.URL
	}

	slider := b.newTrackingSlider(trackingSliderParams{
		Bot:       bot,
		Trackings: list,
	})

	if _, err = slider.Show(ctx, bot, chatId); err != nil {
		log.Errorf("telegram.handleTrackingListMenu: telegram.Slider.Show: %v", err)
		return
	}

	err = b.upsertSession(ctx, upsertSessionParams{
		ChatId: chatId,
		Menu:   models.TrackingListMenu,
	})
	if err != nil {
		log.Errorf("telegram.handleTrackingListMenu: b.upsertSession: %v", err)
		return
	}
}

func (b *Bot) handleTrackingDeleteConfirmMenu(ctx context.Context, bot *telegram.Bot, update *tgmodels.Update) {
	chatId, ok := findChatIdInUpdate(update)
	if !ok {
		log.Warnf("telegram.handleTrackingDeleteConfirmMenu: findChatIdInUpdate: chat not found")
		return
	}

	session, err := b.findSession(ctx, chatId)
	if err != nil {
		log.Errorf("telegram.handleTrackingDeleteConfirmMenu: b.findSession: %v", err)
		return
	}

	_, err = b.deps.Mongodb.Delete(ctx, mongodb.DeleteParams{
		CommonParams: mongodb.CommonParams{
			Database:   "outfit",
			Collection: "tracking",
		},
		Filters: map[string]any{
			"chat_id": session.Tracking.ChatId,
			"url":     session.Tracking.URL,
		},
	})
	if err != nil {
		log.Errorf("telegram.handleTrackingDeleteConfirmMenu: b.deps.Mongodb.Delete: %v", err)
		return
	}

	reply := newReplyKeyboard("start").
		Row().Button("Помощь", bot, telegram.MatchTypeExact, b.handleStartMenu).
		Row().Button("Добавить отслеживание", bot, telegram.MatchTypeExact, b.handleTrackingInsertMenu).
		Row().Button("Мои отслеживания", bot, telegram.MatchTypeExact, b.handleTrackingListMenu)

	err = b.sendMessage(ctx, sendMessageParams{
		ChatId: chatId,
		Text:   `Отслеживание успешно удалено!`,
		Reply:  reply,
	})
	if err != nil {
		log.Errorf("telegram.handleTrackingSelectDeleteMenu: b.sendMessage: %v", err)
		return
	}

	err = b.upsertSession(ctx, upsertSessionParams{
		ChatId: chatId,
		Menu:   models.TrackingConfirmDeleteMenu,
	})
	if err != nil {
		log.Errorf("telegram.handleTrackingListMenu: b.upsertSession: %v", err)
		return
	}
}
