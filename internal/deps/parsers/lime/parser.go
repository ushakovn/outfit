package lime

import (
  "context"
  "encoding/json"
  "fmt"
  neturl "net/url"
  "regexp"
  "strings"

  set "github.com/deckarep/golang-set/v2"
  "github.com/go-resty/resty/v2"
  "github.com/samber/lo"
  log "github.com/sirupsen/logrus"
  "github.com/ushakovn/outfit/internal/models"
  "github.com/ushakovn/outfit/pkg/money"
  "github.com/ushakovn/outfit/pkg/validator"
)

const baseAPIURL = "https://lime-shop.com/api/v2/product/"

var regexURL = regexp.MustCompile(`https?://(www\.)?lime-shop\.com/.+`)

type Parser struct {
  deps Dependencies
}

type Dependencies struct {
  Client *resty.Client
}

func NewParser(deps Dependencies) *Parser {
  return &Parser{deps: deps}
}

func findProductCode(url string) (code string, err error) {
  // Пример: https://lime-shop.com/ru_ru/product/21261_0428_887-temno_seryi_melanz.
  // Код товара: 21261_0428_887.

  _, slug, _ := strings.Cut(url, "/product/")
  parts := strings.Split(slug, "/")

  if len(parts) < 1 {
    return "", fmt.Errorf("product code not found in url: %s", url)
  }
  slug = strings.Trim(parts[0], "/ ")

  parts = strings.Split(slug, "-")

  if len(parts) < 1 {
    return "", fmt.Errorf("product code not found in url: %s", url)
  }
  code = strings.Trim(parts[0], "- ")

  return code, nil
}

func makeParsedProduct(url string, page *ParsedPage) (*ParsedProduct, error) {
  selectedColor, err := findProductColor(url)
  if err != nil {
    return nil, fmt.Errorf("findProductColor: %w", err)
  }

  foundModel, ok := lo.Find(page.Models, func(model ParsedModel) bool {
    modelColor := strings.TrimSpace(model.Code)

    return strings.EqualFold(selectedColor, modelColor)
  })
  if !ok {
    return nil, fmt.Errorf("product model with color: %s not found", selectedColor)
  }

  return &ParsedProduct{
    Page:  lo.FromPtr(page),
    Model: foundModel,
  }, nil
}

func findProductColor(url string) (color string, err error) {
  // Пример: https://lime-shop.com/ru_ru/product/21261_0428_887-temno_seryi_melanz.
  // Цвет товара: temno_seryi_melanz.

  _, slug, _ := strings.Cut(url, "/product/")
  parts := strings.Split(slug, "-")

  if len(parts) < 1 {
    return "", fmt.Errorf("product color not found in url: %s", url)
  }

  color = strings.Trim(parts[len(parts)-1], "- ")
  color, _, _ = strings.Cut(color, "?")

  color = strings.TrimSpace(color)

  return color, nil
}

func (p *Parser) findProductJSON(ctx context.Context, url string) (*ParsedProduct, error) {
  code, err := findProductCode(url)
  if err != nil {
    return nil, fmt.Errorf("findProductCode: %w", err)
  }

  endpoint, err := neturl.JoinPath(baseAPIURL, code)
  if err != nil {
    return nil, fmt.Errorf("neturl.JoinPath: %w", err)
  }

  resp, err := p.deps.Client.R().SetContext(ctx).Get(endpoint)
  if err != nil {
    return nil, fmt.Errorf("resty.Client.Get: %w", err)
  }

  body := resp.Body()
  parsed := new(ParsedPage)

  if err = json.Unmarshal(body, parsed); err != nil {
    return nil, fmt.Errorf("page unmarshal json: %w", err)
  }

  found, err := makeParsedProduct(url, parsed)
  if err != nil {
    return nil, fmt.Errorf("makeParsedProduct: %w", err)
  }

  return found, nil
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
    Debug("lime product parsing started")

  if err := validateURL(params.URL); err != nil {
    return nil, fmt.Errorf("invalid url: %s. error: %w", params.URL, err)
  }

  parsed, err := p.findProductJSON(ctx, params.URL)
  if err != nil {
    return nil, fmt.Errorf("p.findProductJSON: %w", err)
  }

  product := makeProductFromParsed(params.URL, parsed)

  paramsSizesSet := makeSizesSet(params.Sizes)
  productSizes := make([]string, 0, len(parsed.Model.Skus))

  for _, parsedSku := range parsed.Model.Skus {
    sizeString := strings.TrimSpace(parsedSku.Size.Value)

    if !matchSize(sizeString, paramsSizesSet) {
      log.
        WithFields(log.Fields{
          "params.url":   params.URL,
          "params.sizes": params.Sizes,
          "parsed.size":  sizeString,
        }).
        Debug("lime parsed size not match to params: size will be skipped")

      continue
    }

    productOption := makeProductOption(params.URL, parsedSku)
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
      Debug("lime product has not parsed size: not found on site")

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
    Debug("lime product parsed successfully")

  return &product, nil
}

func makeProductFromParsed(url string, parsed *ParsedProduct) models.Product {
  return models.Product{
    URL:         url,
    Type:        models.FindProductType(url),
    ImageURL:    strings.TrimSpace(parsed.Model.Photo.Url),
    Brand:       "LIME",
    Category:    strings.TrimSpace(parsed.Page.Name),
    Description: strings.TrimSpace(parsed.Page.Description),
  }
}

func makeProductOption(url string, sku ParsedSku) models.ProductOption {
  return models.ProductOption{
    URL: url,
    Stock: models.ProductStock{
      Quantity: sku.Stock.Online,
    },
    Size: models.ProductSizeOptions{
      Base: models.ProductSize{
        System: sku.Size.Unit,
        Value:  sku.Size.Value,
      },
    },
    Price: models.ProductPriceOptions{
      Base: models.ProductPrice{
        IntValue:    sku.Price,
        StringValue: money.String(sku.Price),
      },
      Discount: models.ProductPrice{
        IntValue:    sku.Price,
        StringValue: money.String(sku.Price),
      },
    },
  }
}

func matchSize(sizeString string, sizesSet set.Set[string]) bool {
  return sizesSet.IsEmpty() || sizesSet.ContainsOne(sizeString)
}

func makeSizesSet(params models.ParseSizesParams) set.Set[string] {
  return set.NewSet(params.Values...)
}
