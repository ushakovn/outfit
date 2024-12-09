package lamoda

import (
  "context"
  "encoding/json"
  "fmt"
  neturl "net/url"
  "regexp"
  "strings"

  set "github.com/deckarep/golang-set/v2"
  log "github.com/sirupsen/logrus"
  "github.com/ushakovn/outfit/internal/models"
  "github.com/ushakovn/outfit/pkg/ext"
  "github.com/ushakovn/outfit/pkg/money"
  "github.com/ushakovn/outfit/pkg/parser/xpath"
  "github.com/ushakovn/outfit/pkg/validator"
  "golang.org/x/net/html"
)

const (
  baseURL    = "https://lamoda.ru"
  baseCdnURL = "https://a.lmcdn.ru/product"
)

var (
  regexURL  = regexp.MustCompile(`https?://(www\.)?lamoda.ru/.+`)
  regexNUXT = regexp.MustCompile(`.*__NUXT__.*`)
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
    return nil, fmt.Errorf("product json unmarshal: %w", err)
  }

  return parsed, nil
}

func (p *Parser) findProductNodeContent(ctx context.Context, url string) (string, error) {
  doc, err := p.deps.Xpath.GetHtmlDoc(ctx, url)
  if err != nil {
    return "", fmt.Errorf("p.deps.Xpath.GetHtmlDoc: %w", err)
  }

  script, exist := xpath.FindElement(doc, `//script`, func(node *html.Node) bool {
    content, _ := xpath.RenderContent(node, xpath.ShiftToLastChild)

    return regexNUXT.MatchString(content)
  })

  if !exist {
    return "", fmt.Errorf("product script node not found")
  }

  content, err := xpath.RenderContent(script, xpath.ShiftToLastChild)
  if err != nil {
    return "", fmt.Errorf("xpath.RenderContent: %w", err)
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
  if !regexURL.MatchString(url) {
    return fmt.Errorf("expected: %s", baseURL)
  }
  return nil
}

func (p *Parser) Parse(ctx context.Context, params models.ParseParams) (*models.Product, error) {
  log.
    WithFields(log.Fields{
      "params.url":   params.URL,
      "params.sizes": params.Sizes.Values,
    }).
    Debug("lamoda product parsing started")

  if err := validateURL(params.URL); err != nil {
    return nil, fmt.Errorf("invalid url: %s. error: %w", params.URL, err)
  }

  parsed, err := p.findProductJSON(ctx, params.URL)
  if err != nil {
    return nil, fmt.Errorf("p.findProductJSON: %w", err)
  }

  product := makeProductFromParsed(params.URL, parsed)
  priceOptions := makeProductPriceOptions(parsed, params)

  paramsSizesSet := makeSizesSet(params.Sizes)
  productSizes := make([]string, 0, len(parsed.Product.Sizes))

  for _, parsedSize := range parsed.Product.Sizes {
    sizeString := strings.TrimSpace(parsedSize.BrandTitle)

    if !matchSize(sizeString, paramsSizesSet) {
      log.
        WithFields(log.Fields{
          "params.url":   params.URL,
          "params.sizes": params.Sizes,
          "parsed.size":  sizeString,
        }).
        Debug("lamoda parsed size not match to params: size will be skipped")

      continue
    }

    productOption := makeProductOption(params.URL, parsedSize, priceOptions)
    product.Options = append(product.Options, productOption)

    productSizes = append(productSizes, sizeString)
  }

  paramsSizesSet.RemoveAll(productSizes...)
  notFoundSizes := paramsSizesSet.ToSlice()

  for _, size := range notFoundSizes {
    log.
      WithFields(log.Fields{
        "params.url":  params.URL,
        "params.size": size,
      }).
      Debug("lamoda product has not parsed size: not found on site")

    notFoundSize := models.ProductSize{
      System: "N/A",
      Value:  size,
    }
    productOption := models.ProductOption{
      Size: models.ProductSizeOptions{
        NotFoundSize: &notFoundSize,
      },
    }
    product.Options = append(product.Options, productOption)
  }

  product.SetParsedAt()

  log.
    WithFields(log.Fields{
      "params.url":   params.URL,
      "params.sizes": params.Sizes.Values,
    }).
    Debug("lamoda product parsed successfully")

  return &product, nil
}

func makeProductFromParsed(url string, parsed *ParsedProduct) models.Product {
  title := strings.TrimSpace(parsed.Product.Title)
  model := strings.TrimSpace(parsed.Product.ModelTitle)

  return models.Product{
    URL:      url,
    Type:     models.FindProductType(url),
    ImageURL: makeProductImageURL(parsed),
    Brand:    strings.TrimSpace(parsed.Product.Brand.Title),
    Category: fmt.Sprintf("%s %s", title, model),
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

func makeProductOption(url string, parsed ParsedProductSize, price models.ProductPriceOptions) models.ProductOption {
  return models.ProductOption{
    URL: fmt.Sprintf("%s?sku=%s", url, parsed.Sku),

    Stock: models.ProductStock{
      Quantity: parsed.StockQuantity,
    },
    Size: models.ProductSizeOptions{
      Base: models.ProductSize{
        System: parsed.BrandSizeSystem,
        Value:  parsed.BrandTitle,
      },
    },
    Price: price,
  }
}

func makeProductImageURL(parsed *ParsedProduct) string {
  thumbnail := strings.TrimSpace(parsed.Product.Thumbnail)

  if !ext.IsImage(thumbnail) {
    return ""
  }

  url, err := neturl.JoinPath(baseCdnURL, thumbnail)
  if err != nil {
    return ""
  }

  return url
}

func matchSize(sizeString string, sizesSet set.Set[string]) bool {
  return sizesSet.IsEmpty() || sizesSet.ContainsOne(sizeString)
}

func makeSizesSet(params models.ParseSizesParams) set.Set[string] {
  return set.NewSet(params.Values...)
}
