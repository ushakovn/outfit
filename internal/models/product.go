package models

import (
  "encoding/json"
  "math"
  neturl "net/url"
  "strings"
  "time"

  "github.com/samber/lo"
  "github.com/ushakovn/outfit/pkg/money"
)

const (
  ProductTypeUnknown ProductType = "unknown"
  ProductTypeLamoda  ProductType = "lamoda"
  ProductTypeKixbox  ProductType = "kixbox"
  ProductTypeOktyabr ProductType = "oktyabr"
)

type ProductType = string

type Product struct {
  URL         string          `bson:"url" json:"url"`
  Type        ProductType     `bson:"type" json:"type"`
  ImageURL    string          `bson:"image_url" json:"image_url"`
  Brand       string          `bson:"brand" json:"brand"`
  Category    string          `bson:"category" json:"category"`
  Description string          `bson:"description" json:"description"`
  Options     []ProductOption `bson:"options" json:"options"`
  ParsedAt    time.Time       `bson:"parsed_at" json:"parsed_at"`
}

type ProductOption struct {
  URL   string              `bson:"url" json:"url"`
  Stock ProductStock        `bson:"stock" json:"stock"`
  Size  ProductSizeOptions  `bson:"size" json:"size"`
  Price ProductPriceOptions `bson:"price" json:"price"`
}

type ProductSize struct {
  System string `bson:"system" json:"system"`
  Value  string `bson:"value" json:"value"`
}

type ProductSizeOptions struct {
  Base         ProductSize  `bson:"base" json:"base"`
  NotFoundSize *ProductSize `bson:"not_found_size" json:"not_found_size"`
}

type ProductStock struct {
  Quantity int64 `bson:"quantity" json:"quantity"`
}

type ProductPrice struct {
  IntValue    int64  `bson:"int_value" json:"int_value"`
  StringValue string `bson:"string_value" json:"string_value"`
}

type ProductPriceOptions struct {
  Base     ProductPrice `bson:"price" json:"price"`
  Discount ProductPrice `bson:"discount" json:"discount"`
}

type EmbedProduct struct {
  Value json.RawMessage `bson:"value" json:"value"`
}

type ProductDiff struct {
  Options []ProductOptionDiff `bson:"options" json:"options"`
}

type ProductOptionDiff struct {
  Stock ProductStockDiff   `bson:"stock" json:"stock"`
  Size  ProductSizeOptions `bson:"size" json:"size"`
  Price ProductPriceDiff   `bson:"price" json:"price"`
}

type ProductStockDiff struct {
  OldQuantity     int64 `bson:"old_quantity" json:"old_quantity"`
  Quantity        int64 `bson:"quantity" json:"quantity"`
  IsSellUp        bool  `bson:"is_sell_up" json:"is_sell_up"`
  IsAvailable     bool  `bson:"is_available" json:"is_available"`
  IsComeToInStock bool  `bson:"is_come_to_in_stock" json:"is_come_to_in_stock"`
}

type ProductPriceOptionsDiff struct {
  Base     ProductPriceDiff `bson:"base" json:"base"`
  Discount ProductPriceDiff `bson:"discount" json:"discount"`
}

type ProductPriceDiff struct {
  IsLower  bool   `bson:"is_lower" json:"is_lower"`
  IsHigher bool   `bson:"is_higher" json:"is_higher"`
  New      string `bson:"new" json:"new"`
  Old      string `bson:"old" json:"old"`
  Diff     string `bson:"diff" json:"diff"`
}

func NewProductDiff(stored, parsed Product) *ProductDiff {
  storedOptionsBySize := lo.SliceToMap(stored.Options, func(opt ProductOption) (ProductSizeOptions, ProductOption) {
    return opt.Size, opt
  })

  optionsDiff := make([]ProductOptionDiff, 0, len(parsed.Options))

  for _, parsedOption := range parsed.Options {
    storedOption, ok := storedOptionsBySize[parsedOption.Size]
    if !ok {
      continue
    }

    stockDiff := ProductStockDiff{
      OldQuantity:     storedOption.Stock.Quantity,
      Quantity:        parsedOption.Stock.Quantity,
      IsSellUp:        parsedOption.Stock.Quantity <= 5 && parsedOption.Stock.Quantity < storedOption.Stock.Quantity,
      IsAvailable:     parsedOption.Stock.Quantity > 0,
      IsComeToInStock: parsedOption.Stock.Quantity > 0 && storedOption.Stock.Quantity <= 0,
    }

    priceDiffInt := storedOption.Price.Discount.IntValue - parsedOption.Price.Discount.IntValue
    priceDiffAbs := int64(math.Abs(float64(priceDiffInt)))

    priceDiff := ProductPriceDiff{
      IsLower:  priceDiffInt > 0,
      IsHigher: priceDiffInt < 0,
      New:      money.String(parsedOption.Price.Discount.IntValue),
      Old:      money.String(storedOption.Price.Discount.IntValue),
      Diff:     money.String(priceDiffAbs),
    }

    optionsDiff = append(optionsDiff, ProductOptionDiff{
      Stock: stockDiff,
      Price: priceDiff,
      Size:  parsedOption.Size,
    })
  }

  return &ProductDiff{
    Options: optionsDiff,
  }
}

func (p *Product) SetParsedAt() {
  p.ParsedAt = time.Now()
}

func FindProductType(url string) ProductType {
  parsed, _ := neturl.Parse(url)

  switch {
  case strings.Contains(parsed.Host, "lamoda"):
    return ProductTypeLamoda
  case strings.Contains(parsed.Host, "kixbox"):
    return ProductTypeKixbox
  case strings.Contains(parsed.Host, "oktyabr"):
    return ProductTypeOktyabr
  }

  return ProductTypeUnknown
}
