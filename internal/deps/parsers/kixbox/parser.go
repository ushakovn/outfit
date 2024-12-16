package kixbox

import (
  "context"
  "encoding/json"
  "fmt"
  "regexp"
  "sort"
  "strings"

  set "github.com/deckarep/golang-set/v2"
  log "github.com/sirupsen/logrus"
  "github.com/spf13/cast"
  "github.com/ushakovn/outfit/internal/models"
  "github.com/ushakovn/outfit/pkg/ext"
  "github.com/ushakovn/outfit/pkg/money"
  "github.com/ushakovn/outfit/pkg/parser/xpath"
  "github.com/ushakovn/outfit/pkg/validator"
  "golang.org/x/net/html"
)

const (
  schemaContext        = "schema.org"
  schemaTypeProduct    = "Page"
  schemaTypeBreadcrumb = "BreadcrumbList"
)

var regexURL = regexp.MustCompile(`https?://(www\.)?kixbox\.ru/.+`)

type Parser struct {
  deps Dependencies
}

type Dependencies struct {
  Xpath *xpath.Parser
}

func NewParser(deps Dependencies) *Parser {
  return &Parser{deps: deps}
}

func validateURL(url string) error {
  if err := validator.URL(url); err != nil {
    return fmt.Errorf("url invalid: %w", err)
  }
  if !regexURL.MatchString(url) {
    return fmt.Errorf("url %s does not match regex %s", url, regexURL.String())
  }
  return nil
}

func (p *Parser) Parse(ctx context.Context, params models.ParseParams) (*models.Product, error) {
  log.
    WithFields(log.Fields{
      "params.url":   params.URL,
      "params.sizes": params.Sizes.Values,
    }).
    Debug("kixbox product parsing started")

  if err := validateURL(params.URL); err != nil {
    return nil, fmt.Errorf("invalid url: %s. error: %w", params.URL, err)
  }

  doc, err := p.fetchXpathDoc(ctx, params.URL)
  if err != nil {
    return nil, fmt.Errorf("p.fetchXpathDoc: %w", err)
  }

  parsed, err := findProductJSON(doc)
  if err != nil {
    return nil, fmt.Errorf("findProductJSON: %w", err)
  }
  product := makeProductFromParsed(params.URL, parsed)

  brand, err := findProductBrand(doc)
  if err != nil {
    return nil, fmt.Errorf("findProductBrand: %w", err)
  }
  product.Brand = brand

  sizeToStockMatching, err := findProductStocks(doc)
  if err != nil {
    return nil, fmt.Errorf("findProductStocks: %w", err)
  }
  skuToSizeMatching := findProductSizes(doc)

  paramsSizesSet := makeSizesSet(params.Sizes)
  productSizes := make([]string, 0, len(parsed.Offers))

  for _, parsedOffer := range parsed.Offers {
    sizeString, ok := findOfferSize(parsedOffer, skuToSizeMatching)

    if !ok || !matchSize(sizeString, paramsSizesSet) {
      log.
        WithFields(log.Fields{
          "params.url":   params.URL,
          "params.sizes": params.Sizes,
          "parsed.size":  sizeString,
        }).
        Debug("kixbox parsed size not match to params: size will be skipped")

      continue
    }

    stockQuantity, ok := findSizeStock(sizeString, sizeToStockMatching)
    if !ok {
      log.
        WithFields(log.Fields{
          "params.url":             params.URL,
          "params.sizes":           params.Sizes,
          "parsed.size":            sizeString,
          "parsed.stocks.matching": sizeToStockMatching,
        }).
        Debug("kixbox parsed size not match to parsed stocks: size will be skipped")

      continue
    }

    var productOption models.ProductOption

    productOption, err = makeProductOption(sizeString, stockQuantity, parsedOffer)
    if err != nil {
      log.
        WithFields(log.Fields{
          "params.url":   params.URL,
          "params.sizes": params.Sizes,
          "parsed.size":  sizeString,
        }).
        Errorf("kixbox make product option error: %v", err)

      continue
    }

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
      Debug("kixbox product has not parsed size: not found on site")

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
    Debug("kixbox product parsed successfully")

  return &product, nil
}

func findProductSizes(doc *xpath.HtmlDocument) SkuToSizeMatching {
  const path = `//form[contains(@action, 'cart')]//select[contains(@name, 'variant')]//option`

  nodes := xpath.CollectElements(doc, path)
  matching := make(SkuToSizeMatching, len(nodes))

  for _, node := range nodes {
    code, ok := xpath.GetAttribute(node, "value")
    if !ok {
      continue
    }
    code = strings.TrimSpace(code)

    content, ok := xpath.GetContent(node, xpath.ShiftToLastChild)
    if !ok {
      continue
    }

    parts := strings.Split(content, " / ")
    if len(parts) < 2 {
      continue
    }

    value := strings.ReplaceAll(parts[0], " ", "")
    value = strings.TrimSpace(value)

    matching[code] = value
  }

  return matching
}

func findProductJSON(doc *xpath.HtmlDocument) (*ParsedProduct, error) {
  content, err := findProductNodeContent(doc)
  if err != nil {
    return nil, fmt.Errorf("p.findProductNodeContent: %w", err)
  }

  parsed := new(ParsedProduct)

  if err = json.Unmarshal([]byte(content), parsed); err != nil {
    return nil, fmt.Errorf("product json unmarshal: %w", err)
  }

  return parsed, nil
}

func (p *Parser) fetchXpathDoc(ctx context.Context, url string) (*xpath.HtmlDocument, error) {
  doc, err := p.deps.Xpath.GetHtmlDoc(ctx, url)
  if err != nil {
    return nil, fmt.Errorf("p.deps.Xpath.GetHtmlDoc: %w", err)
  }

  return doc, nil
}

func findProductNodeContent(doc *xpath.HtmlDocument) (string, error) {
  const path = `//script[contains(@type, 'json')]`

  script, ok := xpath.FindElement(doc, path, func(node *html.Node) bool {
    content, ok := xpath.GetContent(node, xpath.ShiftToLastChild)

    return ok && strings.Contains(content, schemaContext) && strings.Contains(content, schemaTypeProduct)
  })
  if !ok {
    return "", fmt.Errorf("product script node not found")
  }

  content, _ := xpath.GetContent(script, xpath.ShiftToLastChild)
  content = sanitizeNodeContent(content)

  return content, nil
}

func findBreadcrumbsNodeContent(doc *xpath.HtmlDocument) (string, error) {
  const path = `//script[contains(@type, 'json')]`

  script, exist := xpath.FindElement(doc, path, func(node *html.Node) bool {
    content, ok := xpath.GetContent(node, xpath.ShiftToLastChild)

    return ok && strings.Contains(content, schemaContext) && strings.Contains(content, schemaTypeBreadcrumb)
  })

  if !exist {
    return "", fmt.Errorf("breadcrumbs script node not found")
  }

  content, _ := xpath.GetContent(script, xpath.ShiftToLastChild)
  content = sanitizeNodeContent(content)

  return content, nil
}

func findBreadcrumbsJSON(doc *xpath.HtmlDocument) (*ParsedBreadcrumbs, error) {
  content, err := findBreadcrumbsNodeContent(doc)
  if err != nil {
    return nil, fmt.Errorf("findBreadcrumbsNodeContent: %w", err)
  }

  parsed := new(ParsedBreadcrumbs)

  if err = json.Unmarshal([]byte(content), parsed); err != nil {
    return nil, fmt.Errorf("breadcrumbs json unmarshal: %w", err)
  }

  return parsed, nil
}

func findBrandInHeading(doc *xpath.HtmlDocument) (string, error) {
  const path = `//form[contains(@action, 'cart')]//h2[contains(@class, 'heading')]//a`

  brand, exist := xpath.FindElement(doc, path, func(node *html.Node) bool {
    _, ok := xpath.GetContent(node, xpath.ShiftToLastChild)
    return ok
  })

  if exist {
    content, _ := xpath.GetContent(brand, xpath.ShiftToLastChild)
    return content, nil
  }

  return "", nil
}

func findBrandInBreadcrumbs(doc *xpath.HtmlDocument) (string, error) {
  parsed, err := findBreadcrumbsJSON(doc)
  if err != nil {
    return "", fmt.Errorf("findBreadcrumbsJSON: %w", err)
  }

  elems := parsed.ItemListElement

  if len(elems) == 0 {
    return "", nil
  }

  sort.Slice(elems, func(i, j int) bool {
    return elems[i].Position > elems[j].Position
  })

  content := strings.TrimSpace(elems[0].Name)
  content = html.UnescapeString(content)

  return content, nil
}

func findProductBrand(doc *xpath.HtmlDocument) (string, error) {
  brand, err := findBrandInHeading(doc)
  if err != nil {
    return "", fmt.Errorf("findBrandInHeading: %w", err)
  }

  if brand != "" {
    return brand, nil
  }

  brand, err = findBrandInBreadcrumbs(doc)
  if err != nil {
    return "", fmt.Errorf("findBrandInBreadcrumbs: %w", err)
  }

  if brand != "" {
    return brand, nil
  }

  return "", fmt.Errorf("product brand not found")
}

func findProductStocks(doc *xpath.HtmlDocument) (SizeToStockMatching, error) {
  const path = `//div[contains(@class, 'stocks-data')]`

  node, exist := xpath.FindElement(doc, path, func(node *html.Node) bool {
    _, ok := xpath.GetAttributeContains(node, "stocks")
    return ok
  })

  if !exist {
    return nil, fmt.Errorf("product stocks node not found")
  }

  attr, _ := xpath.GetAttributeContains(node, "stocks")

  attr = html.UnescapeString(attr)
  attr = strings.ReplaceAll(attr, `'`, `"`)
  attr = strings.Trim(attr, "` ")

  parsed := make(ParsedProductStocks)

  if err := json.Unmarshal([]byte(attr), &parsed); err != nil {
    return nil, fmt.Errorf("product stocks json unmarshal: %w", err)
  }

  matching := make(SizeToStockMatching)

  for size, warehouses := range parsed {
    var total int64

    for _, count := range warehouses {
      total += cast.ToInt64(count)
    }
    matching[size] = total
  }

  return matching, nil
}

func sanitizeNodeContent(content string) string {
  content = strings.TrimSpace(content)
  content = html.UnescapeString(content)

  return content
}

func makeProductFromParsed(url string, parsed *ParsedProduct) models.Product {
  return models.Product{
    URL:         url,
    Type:        models.FindProductType(url),
    ImageURL:    makeProductImageURL(parsed),
    Category:    strings.TrimSpace(parsed.Name),
    Description: strings.TrimSpace(parsed.Description),
  }
}

func makeProductImageURL(parsed *ParsedProduct) string {
  url := strings.TrimSpace(parsed.Image)

  if !ext.IsImage(url) {
    return ""
  }
  return url
}

func makeProductOption(sizeString string, stockQuantity int64, parsedOffer ParsedProductOffer) (models.ProductOption, error) {
  price, err := cast.ToInt64E(parsedOffer.Price)
  if err != nil {
    return models.ProductOption{}, fmt.Errorf("offer.Price: %s. cast.ToInt64E: %w", parsedOffer.Price, err)
  }

  return models.ProductOption{
    URL: parsedOffer.Url,

    Stock: models.ProductStock{
      Quantity: stockQuantity,
    },
    Size: models.ProductSizeOptions{
      Base: models.ProductSize{
        System: "N/A",
        Value:  sizeString,
      },
    },
    Price: models.ProductPriceOptions{
      Base: models.ProductPrice{
        IntValue:    price,
        StringValue: money.String(price),
      },
      Discount: models.ProductPrice{
        IntValue:    price,
        StringValue: money.String(price),
      },
    },
  }, nil
}

func matchSize(sizeString string, sizesSet set.Set[string]) bool {
  return sizesSet.IsEmpty() || sizesSet.ContainsOne(sizeString)
}

func makeSizesSet(params models.ParseSizesParams) set.Set[string] {
  return set.NewSet(params.Values...)
}

func findOfferSize(parsedOffer ParsedProductOffer, matching SkuToSizeMatching) (string, bool) {
  size, ok := matching[parsedOffer.Sku]
  return size, ok
}

func findSizeStock(sizeString string, matching SizeToStockMatching) (int64, bool) {
  stock, ok := matching[sizeString]
  return stock, ok
}
