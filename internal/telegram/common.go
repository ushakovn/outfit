package telegram

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	telegram "github.com/go-telegram/bot"
	tgmodels "github.com/go-telegram/bot/models"
	tgreply "github.com/go-telegram/ui/keyboard/reply"
	tgslider "github.com/go-telegram/ui/slider"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"github.com/ushakovn/outfit/internal/message"
	"github.com/ushakovn/outfit/internal/models"
	"github.com/ushakovn/outfit/internal/provider/mongodb"
	"github.com/ushakovn/outfit/pkg/validator"
)

func (b *Bot) findSession(ctx context.Context, chatID int64) (*models.Session, error) {
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

type sendMessageParams struct {
	ChatId int64
	Text   string
	Reply  tgmodels.ReplyMarkup
}

func (b *Bot) sendMessage(ctx context.Context, params sendMessageParams) error {
	_, err := b.deps.Telegram.SendMessage(ctx, &telegram.SendMessageParams{
		ChatID:      params.ChatId,
		Text:        telegram.EscapeMarkdown(params.Text),
		ParseMode:   tgmodels.ParseModeMarkdown,
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

func (b *Bot) upsertSession(ctx context.Context, params upsertSessionParams) error {
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

func (b *Bot) findTracking(ctx context.Context, chatId int64, url string) (*models.Tracking, error) {
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
		return nil, fmt.Errorf("b.deps.Mongodb.Get: %w", err)
	}

	tracking, ok := res.(*models.Tracking)
	if !ok {
		return nil, fmt.Errorf("cast %v with type: %[1]T to: %T failed", res, new(models.Tracking))
	}

	return tracking, nil
}

func (b *Bot) listTrackings(ctx context.Context, chatID int64) ([]*models.Tracking, error) {
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

type parsedTrackingFields struct {
	ErrorMessage string
	Comment      string
	URL          string
	Sizes        models.ParseSizesParams
	Discount     *models.ParseDiscountParams
}

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

func newTracking(chatID int64, fields parsedTrackingFields, msg models.TrackingMessage) *models.Tracking {
	return &models.Tracking{
		ChatId:        chatID,
		URL:           fields.URL,
		Sizes:         fields.Sizes,
		Discount:      fields.Discount,
		ParsedProduct: msg.Product,
	}
}

func newReplyKeyboard(prefix string) *tgreply.ReplyKeyboard {
	return tgreply.New(
		tgreply.WithPrefix(prefix),
		tgreply.IsOneTimeKeyboard(),
		tgreply.ResizableKeyboard(),
	)
}

type trackingSliderParams struct {
	Bot       *telegram.Bot
	Trackings []*models.Tracking
}

func (b *Bot) newTrackingSlider(params trackingSliderParams) *tgslider.Slider {
	slides := make([]tgslider.Slide, 0, len(params.Trackings))

	for _, tracking := range params.Trackings {
		res := message.Do().
			SetProduct(tracking.ParsedProduct).
			BuildProductMessage()

		if !res.IsSendable {
			continue
		}

		slides = append(slides, tgslider.Slide{
			Text:  telegram.EscapeMarkdown(res.Message.TextValue),
			Photo: photo,
		})
	}

	return tgslider.New(params.Bot, slides,
		tgslider.OnError(func(err error) {
			log.Errorf("telegram.TrackingSlider: %v", err)
		}),
		tgslider.WithPrefix("tracking"),
		tgslider.OnSelect("Удалить", false, b.handleTrackingSelectDeleteMenu),
		tgslider.OnCancel("Назад", false, b.handleTrackingSelectSilentMenu),
	)
}

var photo = `https://w7.pngwing.com/pngs/566/160/png-transparent-golang-hd-logo-thumbnail.png`
