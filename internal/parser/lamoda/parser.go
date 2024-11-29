package lamoda

import (
  "context"
  "encoding/json"
  "fmt"
  neturl "net/url"
  "regexp"
  "strings"

  set "github.com/deckarep/golang-set/v2"
  "github.com/ushakovn/outfit/internal/models"
  "github.com/ushakovn/outfit/pkg/extension"
  "github.com/ushakovn/outfit/pkg/money"
  "github.com/ushakovn/outfit/pkg/parser/xpath"
  "github.com/ushakovn/outfit/pkg/validator"
)

const (
  baseURL    = "https://lamoda.ru"
  baseCdnURL = "https://a.lmcdn.ru/product"
)

var (
  regexBaseURL      = regexp.MustCompile(`https?://(www\.)?lamoda.ru/.+`)
  regexProductJSVar = regexp.MustCompile(`.*__NUXT__.*`)
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

    if content = regexProductJSVar.FindString(content); content != "" {
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

func validateURL(url string) error {
  if err := validator.URL(url); err != nil {
    return err
  }
  if !regexBaseURL.MatchString(url) {
    return fmt.Errorf("expected: %s", baseURL)
  }
  return nil
}

func (p *Parser) Parse(ctx context.Context, params models.ParseParams) (*models.Product, error) {
  if err := validateURL(params.URL); err != nil {
    return nil, fmt.Errorf("invalid url: %s. error: %w", params.URL, err)
  }

  parsed, err := p.findProductJSON(ctx, params.URL)
  if err != nil {
    return nil, fmt.Errorf("p.findProductJSON: %w", err)
  }

  product := makeProductFromParsed(params.URL, parsed)

  paramsSizesSet := makeSizesSet(params.Sizes.Values)
  priceOptions := makeProductPriceOptions(parsed, params)

  productSizes := make([]string, 0, len(parsed.Product.Sizes)*2)

  for _, parsedSize := range parsed.Product.Sizes {
    sizeString := makeSizeString(parsedSize)

    if !matchSize(sizeString, paramsSizesSet) {
      continue
    }

    productOption := makeProductOption(parsedSize, priceOptions)
    product.Options = append(product.Options, productOption)

    productSizes = append(productSizes, sizeString)
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
          StringValue: size,
        },
      },
    })
  }

  return &product, nil
}

func makeProductFromParsed(url string, parsed *ParsedProduct) models.Product {
  return models.Product{
    URL:         url,
    Type:        models.ProductTypeByURL(url),
    ImageURL:    makeProductImageURL(parsed),
    Brand:       parsed.Product.Brand.Title,
    Category:    parsed.Product.Title,
    Description: parsed.Product.ModelTitle,
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

func makeProductOption(parsed *ParsedProductSize, price models.ProductPriceOptions) models.ProductOption {
  return models.ProductOption{
    Size: models.ProductSizeOptions{
      Brand: models.ProductSize{
        System: parsed.BrandSizeSystem,
        Value:  parsed.BrandTitle,
      },
    },
    Stock: models.ProductStock{
      Quantity: parsed.StockQuantity,
    },
    Price: price,
  }
}

func makeProductImageURL(parsed *ParsedProduct) string {
  thumbnail := strings.TrimSpace(parsed.Product.Thumbnail)

  if !extension.IsImage(thumbnail) {
    return ""
  }

  url, err := neturl.JoinPath(baseCdnURL, thumbnail)
  if err != nil {
    return ""
  }

  return url
}

func makeSizeString(parsed *ParsedProductSize) string {
  sizeString := parsed.BrandTitle

  if parsed.BrandSizeSystem != "" {
    sizeString = fmt.Sprintf("%s %s", sizeString, parsed.BrandSizeSystem)
  }

  return sizeString
}

func matchSize(sizeString string, sizesSet set.Set[string]) bool {
  return sizesSet.IsEmpty() || sizesSet.ContainsOne(sizeString)
}

func makeSizesSet(values []string) set.Set[string] {
  return set.NewSet(values...)
}
