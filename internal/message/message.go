package message

import (
  "fmt"
  "strings"

  "github.com/ushakovn/outfit/internal/models"
)

type Builder struct {
  product  models.Product
  diff     models.ProductDiff
  tracking models.Tracking
}

func Do() Builder {
  return Builder{}
}

func (b Builder) SetProduct(product models.Product) Builder {
  b.product = product
  return b
}

func (b Builder) SetProductDiff(diff models.ProductDiff) Builder {
  b.diff = diff
  return b
}

func (b Builder) SetTracking(tracking models.Tracking) Builder {
  b.tracking = tracking
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

func (b Builder) SetTrackingPtr(tracking *models.Tracking) Builder {
  b.tracking = *tracking
  return b
}

type BuildResult struct {
  Message    models.TrackingMessage
  IsSendable bool
}

func (b Builder) BuildDiffMessage() BuildResult {
  var (
    isSendable   bool
    isInStock    bool
    isPriceLower bool
  )

  text := fmt.Sprintf(`<b>Оповещение по товару 🤠:</b>
%s %s %s
(Ссылка: %s)

`, b.product.Brand, b.product.Category, b.product.Description,
    b.product.URL)

  for _, option := range b.diff.Options {

    if option.Stock.IsComeToInStock {
      isSendable = true

      text += fmt.Sprintf(`<b>Размер: %s %s появился в наличие 📦!</b>
Доступен в количестве: %d шт.
`,
        option.Size.Brand.Value, option.Size.Brand.System,
        option.Stock.Quantity,
      )

    } else if option.Stock.IsAvailable {
      isInStock = true

      text += fmt.Sprintf(`Размер: %s %s в наличие.
Доступен в количестве: %d шт.
`,
        option.Size.Brand.Value, option.Size.Brand.System,
        option.Stock.Quantity)
    }

    if option.Price.IsLower {
      isPriceLower = true

      text += fmt.Sprintf(`<b>Цена на размер была снижена 📉!</b>
Новая цена: %s.
(Старая цена: %s. Разница: %s)`, option.Price.New, option.Price.Old, option.Price.Diff)

    } else {

      text += fmt.Sprintf(`Цена на размер: %s.`, option.Price.New)
    }

    text += "\n\n"

    if isInStock && isPriceLower {
      isSendable = true
    }
  }

  text = strings.TrimSpace(text)

  return BuildResult{
    Message: models.TrackingMessage{
      Product:     b.product,
      ProductDiff: b.diff,
      TextValue:   text,
    },
    IsSendable: isSendable,
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

  text = strings.TrimSpace(text)

  return BuildResult{
    Message: models.TrackingMessage{
      Product:   b.product,
      TextValue: text,
    },
    IsSendable: true,
  }
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
  text = strings.TrimSpace(text)

  return BuildResult{
    Message: models.TrackingMessage{
      Product:   b.product,
      TextValue: text,
    },
    IsSendable: true,
  }
}
