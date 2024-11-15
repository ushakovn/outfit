package message

import (
  "fmt"
  "strings"

  "github.com/ushakovn/outfit/internal/models"
)

type Builder struct {
  tracking models.Tracking
  product  models.Product
  diff     models.ProductDiff
}

func Do() Builder {
  return Builder{}
}

func (b Builder) SetTracking(tracking models.Tracking) Builder {
  b.tracking = tracking
  return b
}

func (b Builder) SetProduct(product models.Product) Builder {
  b.product = product
  return b
}

func (b Builder) SetProductDiff(diff models.ProductDiff) Builder {
  b.diff = diff
  return b
}

func (b Builder) SetTrackingPtr(tracking *models.Tracking) Builder {
  b.tracking = *tracking
  return b
}

func (b Builder) SetProductPtr(product *models.Product) Builder {
  b.product = *product
  return b
}

func (b Builder) SetProductDiffPtr(diff *models.ProductDiff) Builder {
  b.diff = *diff
  return b
}

type BuildResult struct {
  Message    models.TrackingMessage
  IsSendable bool
}

func (b Builder) BuildDiffTrackingMessage() BuildResult {
  var (
    isSendable   bool
    isInStock    bool
    isPriceLower bool
  )

  text := fmt.Sprintf(`Оповещение по товару:
%s %s %s
(Ссылка: %s)

`, b.product.Brand, b.product.Category, b.product.Description,
    b.product.URL)

  for _, option := range b.diff.Options {

    if option.Stock.IsComeToInStock {
      isSendable = true

      text += fmt.Sprintf(`
Размер: %s %s (%s %s) появился в наличие!
Доступен в количестве: %d шт.

`,
        option.Size.Source.Value, option.Size.Source.System,
        option.Size.Brand.Value, option.Size.Brand.System,
        option.Stock.Quantity,
      )

    } else if option.Stock.IsAvailable {
      isInStock = true

      text += fmt.Sprintf(`
Размер: %s %s (%s %s) в наличие.
Доступен в количестве: %d шт.

`,
        option.Size.Source.Value, option.Size.Source.System,
        option.Size.Brand.Value, option.Size.Brand.System,
        option.Stock.Quantity)
    }

    if option.Price.IsLower {
      isPriceLower = true

      text += fmt.Sprintf(`
Цена на размер была снижена!
Новая цена учетом всех скидок: %s.
(Старая цена: %s. Разница: %s)

`, option.Price.New, option.Price.Old, option.Price.Diff)

    } else {

      text += fmt.Sprintf(`
Цена на размер с учетом всех скидок: %s.
(Старая цена: %s. Разница: %s)

`, option.Price.New, option.Price.Old, option.Price.Diff)
    }

    if isInStock && isPriceLower {
      isSendable = true
    }
  }

  text = strings.TrimSpace(text)

  return BuildResult{
    Message: models.TrackingMessage{
      Telegram:    b.tracking.Telegram,
      Product:     b.product,
      ProductDiff: b.diff,
      TextValue:   text,
    },
    IsSendable: isSendable,
  }
}

func (b Builder) BuildProductTrackingMessage() BuildResult {
  text := fmt.Sprintf(`Товар:
%s %s %s
(Ссылка: %s)
`, b.product.Brand, b.product.Category, b.product.Description,
    b.product.URL)

  for _, option := range b.product.Options {

    if option.Stock.Quantity != 0 {

      text += fmt.Sprintf(`
Размер: %s %s (%s %s) в наличие.
Доступен в количестве: %d шт.`,
        option.Size.Source.Value, option.Size.Source.System,
        option.Size.Brand.Value, option.Size.Brand.System,
        option.Stock.Quantity)

    }

    if option.Stock.Quantity == 0 && option.Size.EmbedNotFoundSize == nil {

      text += fmt.Sprintf(`
Размер: %s %s (%s %s) отсутствует в наличие.`,
        option.Size.Source.Value, option.Size.Source.System,
        option.Size.Brand.Value, option.Size.Brand.System)
    }

    if option.Size.EmbedNotFoundSize != nil {

      text += fmt.Sprintf(`
Размер: %s не был найден на сайте.`,
        option.Size.EmbedNotFoundSize.ValueString)
    }

    if option.Size.EmbedNotFoundSize == nil {

      text += fmt.Sprintf(`
Цена на размер с учетом всех скидок: %s.
`, option.Price.Discount.StringValue)
    }
  }

  text = strings.TrimSpace(text)

  return BuildResult{
    Message: models.TrackingMessage{
      Telegram:  b.tracking.Telegram,
      Product:   b.product,
      TextValue: text,
    },
    IsSendable: true,
  }
}
