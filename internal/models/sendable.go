package models

import (
  "fmt"
  "strings"
  "time"

  "github.com/google/uuid"
  "github.com/samber/lo"
)

type SendableType string

const (
  TrackingSendableType    SendableType = "tracking"
  ProductSendableType     SendableType = "product"
  ProductDiffSendableType SendableType = "product_diff"
)

type SendableMessage struct {
  UUID        string       `bson:"uuid" json:"uuid"`
  ChatId      int64        `bson:"chat_id" json:"chat_id"`
  Type        SendableType `bson:"type" json:"type"`
  Text        SendableText `bson:"text" json:"text"`
  Product     Product      `bson:"product" json:"product"`
  ProductDiff *ProductDiff `bson:"product_diff" json:"product_diff"`
  SentId      *int         `bson:"sent_id" json:"sent_id"`
  SentAt      *time.Time   `bson:"sent_at" json:"sent_at"`
}

type SendableText struct {
  Value  string `bson:"value" json:"value"`
  SHA256 string `bson:"sha256" json:"sha256"`
}

func (s *SendableMessage) SetAsSent(id int) {
  s.SentId = lo.ToPtr(id)
  s.SentAt = lo.ToPtr(time.Now())
}

type BuildResult struct {
  Message SendableMessage
  IsValid bool
}

type Builder struct {
  chatId   int64
  product  Product
  diff     ProductDiff
  tracking Tracking
}

func Sendable(chatId int64) Builder {
  return Builder{chatId: chatId}
}

func (b Builder) SetProduct(product Product) Builder {
  b.product = product
  return b
}

func (b Builder) SetProductDiff(diff ProductDiff) Builder {
  b.diff = diff
  return b
}

func (b Builder) SetTracking(tracking Tracking) Builder {
  b.tracking = tracking
  return b
}

func (b Builder) SetProductPtr(product *Product) Builder {
  b.product = lo.FromPtr(product)
  return b
}

func (b Builder) SetProductDiffPtr(diff *ProductDiff) Builder {
  b.diff = lo.FromPtr(diff)
  return b
}

func (b Builder) SetTrackingPtr(tracking *Tracking) Builder {
  b.tracking = lo.FromPtr(tracking)
  return b
}

func (b Builder) BuildTrackingMessage() BuildResult {
  text := fmt.Sprintf(`Товар 📦:
%s %s %s
(Ссылка: %s)

`, b.tracking.ParsedProduct.Brand,
    b.tracking.ParsedProduct.Category,
    b.tracking.ParsedProduct.Description,
    b.tracking.ParsedProduct.URL)

  text += `Указанные размеры 📋:
`

  for index, label := range b.tracking.Sizes.Values {
    text += fmt.Sprintf("%d. %s", index+1, label)

    if index != len(b.tracking.Sizes.Values)-1 {
      text += "\n"
    }
  }

  return BuildResult{
    Message: SendableMessage{
      UUID:    uuid.NewString(),
      ChatId:  b.chatId,
      Type:    TrackingSendableType,
      Product: b.product,
      Text: SendableText{
        Value:  strings.TrimSpace(text),
        SHA256: "",
      },
    },
    IsValid: true,
  }
}

func (b Builder) BuildProductMessage() BuildResult {
  text := fmt.Sprintf(`Товар 📦:
%s %s %s
(Ссылка: %s)
`, b.product.Brand, b.product.Category, b.product.Description,
    b.product.URL)

  for index, option := range b.product.Options {

    if option.Stock.Quantity != 0 {
      text += fmt.Sprintf(`
%d. Размер: %s %s в наличие.
Кол-во: %d шт.`,
        index+1,
        option.Size.Brand.Value, option.Size.Brand.System,
        option.Stock.Quantity)
    }

    if option.Stock.Quantity == 0 && option.Size.EmbedNotFoundSize == nil {
      text += fmt.Sprintf(`
%d. Размер: %s %s отсутствует в наличие.`,
        index+1,
        option.Size.Brand.Value, option.Size.Brand.System)
    }

    if option.Size.EmbedNotFoundSize != nil {
      text += fmt.Sprintf(`
%d. Размер: %s не был найден на сайте.`,
        index+1,
        option.Size.EmbedNotFoundSize.StringValue)
    }

    if option.Size.EmbedNotFoundSize == nil {
      text += fmt.Sprintf(`
Цена: %s.
`,
        option.Price.Discount.StringValue)
    }
  }

  return BuildResult{
    Message: SendableMessage{
      UUID:    uuid.NewString(),
      ChatId:  b.chatId,
      Type:    ProductSendableType,
      Product: b.product,
      Text: SendableText{
        Value:  strings.TrimSpace(text),
        SHA256: "",
      },
    },
    IsValid: true,
  }
}

func (b Builder) BuildProductDiffMessage() BuildResult {
  res := BuildResult{
    Message: SendableMessage{
      UUID:        uuid.NewString(),
      ChatId:      b.chatId,
      Type:        ProductDiffSendableType,
      Product:     b.product,
      ProductDiff: &b.diff,
    },
  }

  text := fmt.Sprintf(`<b>Оповещение по товару 🏷️:</b>
%s %s %s
(Ссылка: %s).

`, b.product.Brand, b.product.Category, b.product.Description,
    b.product.URL)

  for _, option := range b.diff.Options {
    switch {
    // Упала цена, есть в наличие.
    case option.Price.IsLower && !option.Stock.IsComeToInStock && option.Stock.IsAvailable:
      res.IsValid = true

      text += fmt.Sprintf(`Цена на размер %s %s была снижена 📉!
Текущая цена: %s 
(Старая цена: %s, Разница: %s).
Доступен в количестве: %d шт.

`,
        option.Size.Brand.Value, option.Size.Brand.System,
        option.Price.New, option.Price.Old, option.Price.Diff,
        option.Stock.Quantity)

    // Упала цена, появился в наличие.
    case option.Price.IsLower && option.Stock.IsComeToInStock:
      res.IsValid = true

      text += fmt.Sprintf(`Размер: %s %s появился в наличие по сниженной цене 📦📉!
Текущая цена: %s 
(Старая цена: %s, Разница: %s).
Доступен в количестве: %d шт.

`,
        option.Size.Brand.Value, option.Size.Brand.System,
        option.Price.New, option.Price.Old, option.Price.Diff,
        option.Stock.Quantity)

    // Появился в наличие.
    case !option.Price.IsLower && option.Stock.IsComeToInStock:
      res.IsValid = true

      text += fmt.Sprintf(`Размер: %s %s появился в наличие 📦!
Текущая цена: %s.
Доступен в количестве: %d шт.

`,
        option.Size.Brand.Value, option.Size.Brand.System,
        option.Price.New,
        option.Stock.Quantity)
    }
  }

  res.Message.Text = SendableText{
    Value:  strings.TrimSpace(text),
    SHA256: "",
  }

  return res
}
