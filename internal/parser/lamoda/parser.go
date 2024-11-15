package lamoda

import (
  "context"
  "encoding/json"
  "fmt"
  "regexp"
  "strings"

  set "github.com/deckarep/golang-set/v2"
  "github.com/ushakovn/outfit/internal/models"
  "github.com/ushakovn/outfit/pkg/money"
  "github.com/ushakovn/outfit/pkg/parser/xpath"
  "github.com/ushakovn/outfit/pkg/validator"
)

var (
  regexNuxt = regexp.MustCompile(`.*__NUXT__.*`)
)

type Parser struct {
  deps Dependencies
}

type Dependencies struct {
  Xpath *xpath.Parser
}

func NewParser(deps Dependencies) *Parser {
  return &Parser{deps: deps}
}

func (p *Parser) findProductJSON(ctx context.Context, url string) (*ParsedProduct, error) {
  content, err := p.findProductNodeContent(ctx, url)
  if err != nil {
    return nil, fmt.Errorf("p.findProductNodeContent: %w", err)
  }

  content, err = sanitizeProductNodeContent(content)
  if err != nil {
    return nil, fmt.Errorf("sanitizeProductNodeContent: %w", err)
  }

  parsed := new(ParsedProduct)

  if err = json.Unmarshal([]byte(content), parsed); err != nil {
    return nil, fmt.Errorf("product json unmarhsal: %w", err)
  }

  return parsed, nil
}

func (p *Parser) findProductNodeContent(ctx context.Context, url string) (string, error) {
  doc, err := p.deps.Xpath.GetHtmlDoc(ctx, url)
  if err != nil {
    return "", fmt.Errorf("p.deps.Xpath.GetHtmlDoc: %w", err)
  }

  scripts := p.deps.Xpath.CollectElements(doc, `//script`)

  var content string

  for _, script := range scripts {
    content, _ = p.deps.Xpath.ExtractNodeContent(script, xpath.ShiftToLastChild)

    if content = regexNuxt.FindString(content); content != "" {
      break
    }
  }

  if content == "" {
    return "", fmt.Errorf("product script node not found")
  }

  return content, nil
}

func sanitizeProductNodeContent(content string) (string, error) {
  for idx := 0; idx < 2; idx++ {
    _, after, ok := strings.Cut(content, "payload")
    if !ok {
      return "", fmt.Errorf("product json payload not found")
    }
    content = after
  }

  before, _, ok := strings.Cut(content, "settings")
  if !ok {
    return "", fmt.Errorf("product json payload not found")
  }
  content = before

  content = strings.TrimLeft(content, ": ")
  content = strings.TrimRight(content, ",. ")

  return content, nil
}

func (p *Parser) Parse(ctx context.Context, params models.ParseParams) (*models.Product, error) {
  if err := validator.URL(params.URL); err != nil {
    return nil, fmt.Errorf("invalid url: %s: %w", params.URL, err)
  }

  parsed, err := p.findProductJSON(ctx, params.URL)
  if err != nil {
    return nil, fmt.Errorf("p.findProductJSON: %w", err)
  }

  product := newProduct(params.URL, parsed)

  paramsSizesSet := makeSizesSet(params.Sizes.Values)
  priceOptions := makeProductPriceOptions(parsed, params)

  productSizes := make([]string, 0, len(parsed.Product.Sizes)*2)

  for _, size := range parsed.Product.Sizes {
    sizeString := fmt.Sprintf("%s %s", size.Title, size.SizeSystem)
    brandSizeString := fmt.Sprintf("%s %s", size.BrandTitle, size.BrandSizeSystem)

    if !paramsSizesSet.ContainsOne(sizeString) && !paramsSizesSet.Contains(brandSizeString) {
      continue
    }

    productOption := models.ProductOption{
      Size: models.ProductSizeOptions{
        Brand: models.ProductSize{
          System: size.BrandSizeSystem,
          Value:  size.BrandTitle,
        },
        Source: models.ProductSize{
          System: size.SizeSystem,
          Value:  size.Title,
        },
      },
      Stock: models.ProductStock{
        Quantity: size.StockQuantity,
      },
      Price: priceOptions,
    }

    product.Options = append(product.Options, productOption)

    productSizes = append(productSizes, sizeString, brandSizeString)
  }

  paramsSizesSet.RemoveAll(productSizes...)
  notFoundSizes := paramsSizesSet.ToSlice()

  for _, size := range notFoundSizes {
    product.Options = append(product.Options, models.ProductOption{
      Stock: models.ProductStock{
        Quantity: 0,
      },
      Size: models.ProductSizeOptions{
        EmbedNotFoundSize: &models.EmbedNotFoundSize{
          ValueString: size,
        },
      },
    })
  }

  return product, nil
}

func newProduct(url string, parsed *ParsedProduct) *models.Product {
  return &models.Product{
    URL:         url,
    Brand:       parsed.Product.Brand.Title,
    Category:    parsed.Product.Title,
    Description: parsed.Product.ModelTitle,
    Embed:       newEmbedProduct(parsed),
  }
}

func newEmbedProduct(parsed *ParsedProduct) *models.EmbedProduct {
  value, _ := json.Marshal(parsed)

  return &models.EmbedProduct{
    Value: value,
  }
}

func makeProductPriceOptions(parsed *ParsedProduct, params models.ParseParams) models.ProductPriceOptions {
  price := models.ProductPrice{
    IntValue:    parsed.Product.Price,
    StringValue: money.String(parsed.Product.Price),
  }

  options := models.ProductPriceOptions{
    Base:     price,
    Discount: price,
  }

  if params.HasDiscount() && parsed.IsDiscountApplicable() {
    multiplier := float64(int64(100)-params.Discount.Percent) * 0.01
    discount := int64(float64(parsed.Product.Price) * multiplier)

    options.Discount = models.ProductPrice{
      IntValue:    discount,
      StringValue: money.String(discount),
    }
  }

  return options
}

func makeSizesSet(values []string) set.Set[string] {
  if len(values) == 0 {
    return set.NewSet("00")
  }

  return set.NewSet(values...)
}
