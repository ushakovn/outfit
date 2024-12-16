package traektoria

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
  "github.com/spf13/cast"
  "github.com/ushakovn/outfit/internal/models"
  "github.com/ushakovn/outfit/pkg/ext"
  "github.com/ushakovn/outfit/pkg/money"
  "github.com/ushakovn/outfit/pkg/stringer"
  "github.com/ushakovn/outfit/pkg/validator"
  "golang.org/x/net/html"
)

// TODO: fix sku selection.

const (
  baseURL    = "https://www.traektoria.ru/"
  baseAPIURL = "https://www.traektoria.ru/slim/pages/product/%s?SKU=%s"
)

var regexURL = regexp.MustCompile(`https?://(www\.)?traektoria\.ru/.+`)

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
  // Пример: https://www.traektoria.ru/product/1639029_bryuki-carhartt-wip-cole-cargo-pant/?SKU=1645314.
  // Код товара: 1639029.

  _, slug, _ := strings.Cut(url, "/product/")
  parts := strings.Split(slug, "/")

  if len(parts) < 1 {
    return "", fmt.Errorf("product code not found in url: %s", url)
  }
  slug = strings.Trim(parts[0], "/ ")

  parts = strings.Split(slug, "_")

  if len(parts) < 1 {
    return "", fmt.Errorf("product code not found in url: %s", url)
  }
  code = strings.Trim(parts[0], "_ ")

  return code, nil
}

func findProductSkuCode(url string) (code string) {
  // Пример: https://www.traektoria.ru/product/1639029_bryuki-carhartt-wip-cole-cargo-pant/?SKU=1645314.
  // Sku товара: 1645314.

  parsed, err := neturl.Parse(url)
  if err != nil {
    return ""
  }

  code = parsed.Query().Get("SKU")
  code = strings.TrimSpace(code)

  return code
}

func findSkuByCode(skuCode string, page *ParsedPage) (*ParsedSku, bool) {
  for _, parsed := range page.Data.MAIN.Content.Model.SkuList {
    for _, size := range parsed.Sizes {
      id := cast.ToString(size.Id)

      if skuCode == id {
        return &parsed, true
      }
    }
  }

  return nil, false
}

func findSkuByColor(page *ParsedPage) (*ParsedSku, bool) {
  color := strings.TrimSpace(page.Data.MAIN.Content.SelectedSku.ColorTitle)

  for _, parsed := range page.Data.MAIN.Content.Model.SkuList {
    name := cast.ToString(parsed.Name)
    name = strings.TrimSpace(name)

    if strings.EqualFold(color, name) {
      return &parsed, true
    }
  }

  return nil, false
}

func makeParsedProduct(skuCode string, page *ParsedPage) (*ParsedProduct, error) {
  if sku, ok := findSkuByCode(skuCode, page); ok {
    return &ParsedProduct{
      Page: lo.FromPtr(page),
      Sku:  lo.FromPtr(sku),
    }, nil
  }

  if sku, ok := findSkuByColor(page); ok {
    return &ParsedProduct{
      Page: lo.FromPtr(page),
      Sku:  lo.FromPtr(sku),
    }, nil
  }

  return nil, fmt.Errorf("product sku not found")
}

func (p *Parser) findProductJSON(ctx context.Context, url string) (*ParsedProduct, error) {
  productCode, err := findProductCode(url)
  if err != nil {
    return nil, fmt.Errorf("findProductCode: %w", err)
  }
  skuCode := findProductSkuCode(url)

  endpoint := fmt.Sprintf(baseAPIURL, productCode, skuCode)

  resp, err := p.deps.Client.R().SetContext(ctx).Get(endpoint)
  if err != nil {
    return nil, fmt.Errorf("resty.Client.Get: %w", err)
  }

  body := resp.Body()
  parsed := new(ParsedPage)

  if err = json.Unmarshal(body, parsed); err != nil {
    return nil, fmt.Errorf("page unmarshal json: %w", err)
  }

  found, err := makeParsedProduct(skuCode, parsed)
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
    Debug("traektoria product parsing started")

  if err := validateURL(params.URL); err != nil {
    return nil, fmt.Errorf("invalid url: %s. error: %w", params.URL, err)
  }

  parsed, err := p.findProductJSON(ctx, params.URL)
  if err != nil {
    return nil, fmt.Errorf("p.findProductJSON: %w", err)
  }

  product := makeProductFromParsed(params.URL, parsed)

  paramsSizesSet := makeSizesSet(params.Sizes)
  productSizes := make([]string, 0, len(parsed.Sku.Sizes))

  for _, parsedSize := range parsed.Sku.Sizes {
    sizeString := strings.TrimSpace(parsedSize.SizeTitle)

    if !matchSize(sizeString, paramsSizesSet) {
      log.
        WithFields(log.Fields{
          "params.url":   params.URL,
          "params.sizes": params.Sizes,
          "parsed.size":  sizeString,
        }).
        Debug("traektoria parsed size not match to params: size will be skipped")

      continue
    }

    productOption := makeProductOption(params.URL, parsedSize)
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
      Debug("traektoria product has not parsed size: not found on site")

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
    Debug("traektoria product parsed successfully")

  return &product, nil
}

func makeProductFromParsed(url string, parsed *ParsedProduct) models.Product {
  brand := makeProductBrand(parsed)

  return models.Product{
    URL:         url,
    Type:        models.FindProductType(url),
    ImageURL:    makeProductImage(parsed),
    Brand:       brand,
    Category:    makeProductCategory(brand, parsed),
    Description: makeProductDescription(parsed),
  }
}

func makeProductBrand(parsed *ParsedProduct) string {
  brand := strings.TrimSpace(parsed.Page.Data.MAIN.Content.Model.Brand.Name)
  return brand
}

func makeProductDescription(parsed *ParsedProduct) string {
  desc := html.UnescapeString(parsed.Page.Data.MAIN.Content.Descriptions.Features)

  desc = stringer.StripTags(desc)
  desc = stringer.ReplaceRepeatSeparators(desc, " ")
  desc = stringer.SanitizeString(desc)

  return desc
}

func makeProductCategory(brand string, parsed *ParsedProduct) string {
  // Пример: "breadcrumb": [
  //  {
  //    "title": "Главная",
  //    "url": "/"
  //  },
  //  {
  //    "title": "Брюки Carhartt Wip Cole Cargo Pant",
  //    "url": ""
  //  } ]
  breadcrumbs := parsed.Page.Data.MAIN.Content.Breadcrumb

  if len(breadcrumbs) != 0 {
    crumb := breadcrumbs[len(breadcrumbs)-1]

    if crumb.Url == "" {
      category := strings.TrimSpace(crumb.Title)

      brand = strings.ToLower(brand)
      brand = strings.Title(brand)

      category = strings.ReplaceAll(category, brand, "")
      category = stringer.ReplaceRepeatSeparators(category, " ")

      return category
    }
  }

  // Пример:
  // "color_title": "PARK (RINSED)",
  // "size_title": "27",
  // "is_available": true,
  // "quantity_text": "",
  // "in_wishlist": false,
  // "name": "CARHARTT WIP COLE CARGO PANT A/S PARK (RINSED) 27",
  selected := parsed.Page.Data.MAIN.Content.SelectedSku

  category := strings.TrimSpace(selected.Name)

  brand = strings.ToUpper(brand)

  category = strings.ReplaceAll(category, brand, "")
  category = stringer.ReplaceRepeatSeparators(category, " ")

  category = strings.ReplaceAll(category, selected.SizeTitle, "")
  category = strings.ReplaceAll(category, selected.ColorTitle, "")

  category = strings.TrimSpace(category)

  return category
}

func makeProductImage(parsed *ParsedProduct) string {
  if len(parsed.Sku.PhotoList) == 0 {
    return ""
  }
  url := strings.TrimSpace(parsed.Sku.PhotoList[0].Url)

  url, err := neturl.JoinPath(baseURL, url)
  if err != nil {
    return ""
  }
  if !ext.IsImage(url) {
    return ""
  }
  return url
}

func makeProductOption(url string, size ParsedSize) models.ProductOption {
  return models.ProductOption{
    URL: url,
    Stock: models.ProductStock{
      Quantity: size.Quantity,
    },
    Size: models.ProductSizeOptions{
      Base: models.ProductSize{
        System: "N/A",
        Value:  strings.TrimSpace(size.SizeTitle),
      },
    },
    Price: models.ProductPriceOptions{
      Base: models.ProductPrice{
        IntValue:    size.BasePrice,
        StringValue: money.String(size.BasePrice),
      },
      Discount: models.ProductPrice{
        IntValue:    size.RetailPrice,
        StringValue: money.String(size.RetailPrice),
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
