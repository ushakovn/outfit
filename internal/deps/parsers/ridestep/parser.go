package ridestep

import (
  "context"
  "encoding/json"
  "errors"
  "fmt"
  "regexp"
  "strings"

  set "github.com/deckarep/golang-set/v2"
  log "github.com/sirupsen/logrus"
  "github.com/spf13/cast"
  "github.com/ushakovn/outfit/internal/models"
  "github.com/ushakovn/outfit/pkg/ext"
  "github.com/ushakovn/outfit/pkg/money"
  "github.com/ushakovn/outfit/pkg/parser/xpath"
  "github.com/ushakovn/outfit/pkg/stringer"
  "github.com/ushakovn/outfit/pkg/validator"
  "golang.org/x/net/html"
)

var (
  errNotFound = errors.New("not found")

  regexURL          = regexp.MustCompile(`https?://(www\.)?ridestep\.ru/.+`)
  regexQuotedString = regexp.MustCompile(`'.+'`)
  regexProductName  = regexp.MustCompile(`currentProductName.+;`)
  regexProductImage = regexp.MustCompile(`currentProductImage.+;`)
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

func (p *Parser) findProduct(ctx context.Context, url string) (*ParsedProduct, error) {
  doc, err := p.fetchXpathDoc(ctx, url)
  if err != nil {
    return nil, fmt.Errorf("fetchXpathDoc: %w", err)
  }

  skus, err := p.findProductSkus(doc)
  if err != nil {
    return nil, fmt.Errorf("p.findProductSkus: %w", err)
  }

  view, err := p.findProductProductView(doc)
  if err != nil {
    return nil, fmt.Errorf("p.findProductProductView: %w", err)
  }

  tracking, err := p.findProductTracking(doc)
  if err != nil {
    return nil, fmt.Errorf("p.findProductTracking: %w", err)
  }

  parsed := &ParsedProduct{
    Skus:  skus,
    Name:  strings.TrimSpace(view.Name),
    Brand: strings.TrimSpace(view.Brand),
    Image: tracking.Image,
  }

  return parsed, nil
}

func (p *Parser) findProductTracking(doc *xpath.HtmlDocument) (*ParsedProductTracking, error) {
  const path = `//script[contains(text(), 'document.current')]`

  node, exist := xpath.FindElement(doc, path, func(node *html.Node) bool {
    _, ok := xpath.GetContent(node, xpath.ShiftToLastChild)
    return ok
  })

  if !exist {
    return nil, errNotFound
  }

  content, _ := xpath.GetContent(node, xpath.ShiftToLastChild)

  name := regexProductName.FindString(content)
  name = regexQuotedString.FindString(name)
  name = strings.Trim(name, "' ")

  image := regexProductImage.FindString(content)
  image = regexQuotedString.FindString(image)
  image = strings.Trim(image, "' ")

  if !ext.IsImage(image) {
    image = ""
  }

  return &ParsedProductTracking{
    Name:  name,
    Image: image,
  }, nil
}

func (p *Parser) findProductProductView(doc *xpath.HtmlDocument) (*ParsedProductView, error) {
  const path = `//script[contains(text(), 'productView')]`

  node, exist := xpath.FindElement(doc, path, func(node *html.Node) bool {
    _, ok := xpath.GetContent(node, xpath.ShiftToLastChild)
    return ok
  })

  if !exist {
    return nil, errNotFound
  }

  content, _ := xpath.GetContent(node, xpath.ShiftToLastChild)

  _, content, _ = strings.Cut(content, "productView")
  content, _, _ = strings.Cut(content, ";")

  content = strings.TrimLeft(content, "(")
  content = strings.TrimRight(content, ");")

  content = sanitizeNodeContent(content)

  parsed := new(ParsedProductView)

  if err := json.Unmarshal([]byte(content), parsed); err != nil {
    return nil, fmt.Errorf("product view unmarshal json: %w", err)
  }

  return parsed, nil
}

func (p *Parser) findProductSkus(doc *xpath.HtmlDocument) ([]ParsedProductSku, error) {
  const path = `//script[contains(text(), '#cart-form') and contains(text(), 'product')]`

  node, exist := xpath.FindElement(doc, path, func(node *html.Node) bool {
    _, ok := xpath.GetContent(node, xpath.ShiftToLastChild)
    return ok
  })

  if !exist {
    return nil, errNotFound
  }

  content, _ := xpath.GetContent(node, xpath.ShiftToLastChild)

  _, content, _ = strings.Cut(content, "skus:")
  content, _, _ = strings.Cut(content, "services:")

  content = strings.TrimRight(content, ",")
  content = sanitizeNodeContent(content)

  parsed := make(map[string]ParsedProductSku)

  if err := json.Unmarshal([]byte(content), &parsed); err != nil {
    return nil, fmt.Errorf("product skus unmarshal json: %w", err)
  }

  skus := make([]ParsedProductSku, 0, len(parsed))

  for _, sku := range parsed {
    skus = append(skus, sku)
  }

  return skus, nil
}

func sanitizeNodeContent(content string) string {
  content = strings.TrimSpace(content)
  content = html.UnescapeString(content)

  return content
}

func (p *Parser) fetchXpathDoc(ctx context.Context, url string) (*xpath.HtmlDocument, error) {
  doc, err := p.deps.Xpath.GetHtmlDoc(ctx, url)
  if err != nil {
    return nil, fmt.Errorf("p.deps.Xpath.GetHtmlDoc: %w", err)
  }

  return doc, nil
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
    Debug("ridestep product parsing started")

  if err := validateURL(params.URL); err != nil {
    return nil, fmt.Errorf("invalid url: %s. error: %w", params.URL, err)
  }

  parsed, err := p.findProduct(ctx, params.URL)
  if err != nil {
    return nil, fmt.Errorf("p.findProduct: %w", err)
  }

  product := makeProductFromParsed(params.URL, parsed)

  paramsSizesSet := makeSizesSet(params.Sizes)
  productSizes := make([]string, 0, len(parsed.Skus))

  for _, parsedSku := range parsed.Skus {
    sizeString := strings.TrimSpace(parsedSku.Name)

    if !matchSize(sizeString, paramsSizesSet) {
      log.
        WithFields(log.Fields{
          "params.url":   params.URL,
          "params.sizes": params.Sizes,
          "parsed.size":  sizeString,
        }).
        Debug("ridestep parsed size not match to params: size will be skipped")

      continue
    }

    var option models.ProductOption

    option, err = makeProductOption(params.URL, parsedSku)
    if err != nil {
      return nil, fmt.Errorf("makeProductOption: %w", err)
    }

    product.Options = append(product.Options, option)

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
      Debug("ridestep product has not parsed size: not found on site")

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
    Debug("ridestep product parsed successfully")

  return &product, nil
}

func sanitizeProductName(brand, name string) string {
  name = strings.ReplaceAll(name, brand, "")
  name = stringer.ReplaceRepeatSeparators(name, " ")
  name = strings.TrimSpace(name)

  return name
}

func makeProductFromParsed(url string, parsed *ParsedProduct) models.Product {
  return models.Product{
    URL:      url,
    Type:     models.FindProductType(url),
    ImageURL: parsed.Image,
    Brand:    parsed.Brand,
    Category: sanitizeProductName(parsed.Brand, parsed.Name),
  }
}

func makeProductOption(url string, sku ParsedProductSku) (option models.ProductOption, err error) {
  quantity, err := cast.ToInt64E(sku.Available)
  if err != nil {
    err = fmt.Errorf("sku.Available: %s cast.ToInt64E: %w", sku.Available, err)

    return
  }

  price, err := cast.ToInt64E(sku.Price)
  if err != nil {
    err = fmt.Errorf("sku.Price: %s cast.ToInt64E: %w", sku.Available, err)

    return
  }

  option = models.ProductOption{
    URL: url,
    Stock: models.ProductStock{
      Quantity: quantity,
    },
    Size: models.ProductSizeOptions{
      Base: models.ProductSize{
        System: "N/A",
        Value:  strings.TrimSpace(sku.Name),
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
  }

  return
}

func matchSize(sizeString string, sizesSet set.Set[string]) bool {
  return sizesSet.IsEmpty() || sizesSet.ContainsOne(sizeString)
}

func makeSizesSet(params models.ParseSizesParams) set.Set[string] {
  return set.NewSet(params.Values...)
}
