package kixbox

type ParsedProduct struct {
  Context     string `json:"@context"`
  Type        string `json:"@type"`
  Name        string `json:"name"`
  Image       string `json:"image"`
  Description string `json:"description"`
  Brand       struct {
    Type string `json:"@type"`
    Name string `json:"name"`
  } `json:"brand"`
  Sku    string               `json:"sku"`
  Offers []ParsedProductOffer `json:"offers"`
}

type ParsedProductOffer struct {
  Type          string `json:"@type"`
  Url           string `json:"url"`
  PriceCurrency string `json:"priceCurrency"`
  Price         string `json:"price"`
  Sku           string `json:"sku"`
  Availability  string `json:"availability"`
}

type ParsedBreadcrumbs struct {
  Context         string `json:"@context"`
  Type            string `json:"@type"`
  ItemListElement []struct {
    Type     string `json:"@type"`
    Position int    `json:"position"`
    Name     string `json:"name"`
    Item     string `json:"item"`
  } `json:"itemListElement"`
}

type ParsedProductStocks map[string]map[string]string

type SizeToStockMatching map[string]int64

type SkuToSizeMatching map[string]string
