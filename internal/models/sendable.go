package models

import (
  "fmt"
  "strings"
  "time"
  "unicode/utf8"

  "github.com/google/uuid"
  "github.com/samber/lo"
  "github.com/ushakovn/outfit/pkg/hasher"
)

type SendableType string

const (
  TrackingSendableType    SendableType = "tracking"
  ProductSendableType     SendableType = "product"
  ProductDiffSendableType SendableType = "product_diff"
)

type SendableMessage struct {
  UUID        string             `bson:"uuid" json:"uuid"`
  ChatId      int64              `bson:"chat_id" json:"chat_id"`
  Type        SendableType       `bson:"type" json:"type"`
  Text        SendableText       `bson:"text" json:"text"`
  Product     Product            `bson:"product" json:"product"`
  ProductDiff *ProductDiff       `bson:"product_diff" json:"product_diff"`
  SentId      *int               `bson:"sent_id" json:"sent_id"`
  Timestamps  SendableTimestamps `bson:"timestamps" json:"timestamps"`
}

type SendableText struct {
  Value  string `bson:"value" json:"value"`
  SHA256 string `bson:"sha256" json:"sha256"`
}

type SendableTimestamps struct {
  CreatedAt time.Time  `bson:"created_at" json:"created_at"`
  SentAt    *time.Time `bson:"sent_at" json:"sent_at"`
}

func (s *SendableMessage) SetAsSent(id int) {
  s.SentId = lo.ToPtr(id)
  s.Timestamps.SentAt = lo.ToPtr(time.Now())
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
  text := fmt.Sprintf(`Отслеживаемый товар 📦

%s %s
%s

`, b.tracking.ParsedProduct.Brand,
    b.tracking.ParsedProduct.Category,
    b.tracking.ParsedProduct.URL)

  if len(b.tracking.Sizes.Values) != 0 {
    sizesString := strings.Join(b.tracking.Sizes.Values, ", ")
    sizesString = strings.TrimSpace(sizesString)

    text += fmt.Sprintf(`Указанные размеры: %s
`, sizesString)
  }

  if utf8.RuneCountInString(b.tracking.Comment) != 0 {
    text += fmt.Sprintf(`
Комментарий к отслеживанию 💬
%s
`, b.tracking.Comment)
  }

  return BuildResult{
    Message: SendableMessage{
      UUID:   uuid.NewString(),
      ChatId: b.chatId,
      Type:   TrackingSendableType,
      Text: SendableText{
        Value:  text,
        SHA256: hasher.SHA256(text),
      },
      Product: b.product,
      Timestamps: SendableTimestamps{
        CreatedAt: time.Now(),
      },
    },
    IsValid: true,
  }
}

func (b Builder) BuildProductMessage() BuildResult {
  text := fmt.Sprintf(`<b>Выбранный товар 📦</b>

%s %s
%s
`, b.product.Brand, b.product.Category,
    b.product.URL)

  for index, option := range b.product.Options {

    if option.Stock.Quantity != 0 {
      text += fmt.Sprintf(`
%d. Размер: %s в наличии
Кол-во: %d шт`,
        index+1,
        option.Size.Base.Value,
        option.Stock.Quantity)
    }

    if option.Stock.Quantity == 0 && option.Size.NotFoundSize == nil {
      text += fmt.Sprintf(`
%d. Размер: %s отсутствует в наличии`,
        index+1,
        option.Size.Base.Value)
    }

    if option.Size.NotFoundSize != nil {
      text += fmt.Sprintf(`
%d. Размер: %s не был найден на сайте`,
        index+1,
        option.Size.NotFoundSize.Value)
    }

    if option.Size.NotFoundSize == nil {
      text += fmt.Sprintf(`
Цена: %s
`,
        option.Price.Discount.StringValue)
    }
  }

  text = strings.TrimSpace(text)

  return BuildResult{
    Message: SendableMessage{
      UUID:    uuid.NewString(),
      ChatId:  b.chatId,
      Type:    ProductSendableType,
      Product: b.product,
      Text: SendableText{
        Value:  text,
        SHA256: hasher.SHA256(text),
      },
      Timestamps: SendableTimestamps{
        CreatedAt: time.Now(),
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
      Timestamps: SendableTimestamps{
        CreatedAt: time.Now(),
      },
    },
  }

  text := fmt.Sprintf(`<b>Оповещение по товару 📦</b>

%s %s
%s

`, b.product.Brand, b.product.Category,
    b.product.URL)

  for _, option := range b.diff.Options {
    switch {
    // Упала цена, есть в наличии.
    case option.Price.IsLower && !option.Stock.IsComeToInStock && option.Stock.IsAvailable:
      res.IsValid = true

      text += fmt.Sprintf(`Цена на размер %s снижена 📉
Текущая цена: %s 
Старая цена: %s
Разница: %s
Доступен в количестве: %d шт

`,
        option.Size.Base.Value,
        option.Price.New, option.Price.Old, option.Price.Diff,
        option.Stock.Quantity)

    // Упала цена, появился в наличии.
    case option.Price.IsLower && option.Stock.IsComeToInStock:
      res.IsValid = true

      text += fmt.Sprintf(`Размер: %s снова в наличии по сниженной цене 📦📉
Текущая цена: %s 
Старая цена: %s
Разница: %s
Доступен в количестве: %d шт

`,
        option.Size.Base.Value,
        option.Price.New, option.Price.Old, option.Price.Diff,
        option.Stock.Quantity)

    // Появился в наличии.
    case !option.Price.IsLower && option.Stock.IsComeToInStock:
      res.IsValid = true

      text += fmt.Sprintf(`Размер: %s снова в наличии 📦
Текущая цена: %s
Доступен в количестве: %d шт

`,
        option.Size.Base.Value,
        option.Price.New,
        option.Stock.Quantity)

    // Цена возросла, есть в наличии, флаг включен.
    case option.Price.IsHigher && option.Stock.IsAvailable && b.tracking.Flags.WithOptional:
      res.IsValid = true

      text += fmt.Sprintf(`Цена на размер: %s возросла 📈
Текущая цена: %s 
Старая цена: %s
Разница: %s
Доступен в количестве: %d шт

`,
        option.Size.Base.Value,
        option.Price.New, option.Price.Old, option.Price.Diff,
        option.Stock.Quantity)

    // Цена возросла, товар раскупают, флаг включен.
    case option.Price.IsHigher && option.Stock.IsAvailable && option.Stock.IsSellUp && b.tracking.Flags.WithOptional:
      res.IsValid = true

      text += fmt.Sprintf(`Цена на размер: %s возросла 📈
Текущая цена: %s
Старая цена: %s
Разница: %s
Количество товара уменьшилось c %d до %d 📉

`,
        option.Size.Base.Value,
        option.Price.New, option.Price.Old, option.Price.Diff,
        option.Stock.OldQuantity, option.Stock.Quantity)

      // Товар раскупают, флаг включен.
    case option.Stock.IsAvailable && option.Stock.IsSellUp && b.tracking.Flags.WithOptional:
      res.IsValid = true

      text += fmt.Sprintf(`Количество товара в размере %s уменьшилось c %d до %d 📉
Текущая цена: %s`,
        option.Size.Base.Value,
        option.Stock.OldQuantity, option.Stock.Quantity,
        option.Price.New)
    }
  }

  text = strings.TrimSpace(text)

  res.Message.Text = SendableText{
    Value:  text,
    SHA256: hasher.SHA256(text),
  }

  return res
}
