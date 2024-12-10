package lime

import (
  "time"
)

type ParsedProduct struct {
  Id           int    `json:"id"`
  Name         string `json:"name"`
  Article      string `json:"article"`
  Code         string `json:"code"`
  Description  string `json:"description"`
  Composition  string `json:"composition"`
  Compositions []struct {
    Name  string `json:"name"`
    Value string `json:"value"`
  } `json:"compositions"`
  Care            string        `json:"care"`
  DescriptionText string        `json:"description_text"`
  Kind            string        `json:"kind"`
  Models          []ParsedModel `json:"models"`
  Measurements    []any         `json:"measurements"`
  HttpMeta        struct {
    Title         string `json:"title"`
    Description   string `json:"description"`
    Keywords      string `json:"keywords"`
    OgTitle       string `json:"og:title"`
    OgDescription string `json:"og:description"`
  } `json:"http_meta"`
}

type ParsedModel struct {
  Id          int    `json:"id"`
  Fit         any    `json:"fit"`
  Detail      any    `json:"detail"`
  Category    string `json:"category"`
  Subcategory string `json:"subcategory"`
  QualityType string `json:"quality_type"`
  Code        string `json:"code"`
  Collection  string `json:"collection"`
  Photo       struct {
    Id        int       `json:"id"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
    Active    bool      `json:"active"`
    ModelId   int       `json:"model_id"`
    Url       string    `json:"url"`
    Slot      int       `json:"slot"`
    Width     int       `json:"width"`
    Height    int       `json:"height"`
    Type      int       `json:"type"`
  } `json:"photo"`
  Name   string      `json:"name"`
  Skus   []ParsedSku `json:"skus"`
  Medias []struct {
    Id     int    `json:"id"`
    Slot   int    `json:"slot"`
    Width  int    `json:"width"`
    Height int    `json:"height"`
    Type   int    `json:"type"`
    Url    string `json:"url"`
  } `json:"medias"`
  HttpMeta struct {
    Title         string `json:"title"`
    Description   string `json:"description"`
    Keywords      string `json:"keywords"`
    OgTitle       string `json:"og:title"`
    OgDescription string `json:"og:description"`
    OgUrl         string `json:"og:url"`
    OgType        string `json:"og:type"`
    OgImage       string `json:"og:image"`
  } `json:"http_meta"`
  Badge []any `json:"badge"`
  Color struct {
    Id   int    `json:"id"`
    Hex  string `json:"hex"`
    Name string `json:"name"`
  } `json:"color"`
  Product struct {
    Id              int    `json:"id"`
    Name            string `json:"name"`
    Article         string `json:"article"`
    Code            string `json:"code"`
    Description     string `json:"description"`
    Composition     any    `json:"composition"`
    Care            string `json:"care"`
    DescriptionText string `json:"description_text"`
  } `json:"product"`
}

type ParsedSku struct {
  Id   int `json:"id"`
  Size struct {
    Id    int    `json:"id"`
    Unit  string `json:"unit"`
    Age   string `json:"age"`
    Value string `json:"value"`
  } `json:"size"`
  Stock struct {
    Online  int64 `json:"online"`
    Offline int64 `json:"offline"`
  } `json:"stock"`
  Price                   int64  `json:"price"`
  PriceFormatted          string `json:"price_formatted"`
  OldPrice                any    `json:"old_price"`
  OldPriceFormatted       string `json:"old_price_formatted"`
  FuturePriceFormatted    any    `json:"future_price_formatted"`
  DiscountFutureFormatted any    `json:"discount_future_formatted"`
  FuturePriceTooltipText  string `json:"future_price_tooltip_text"`
}

type ProductWithModel struct {
  Product ParsedProduct
  Model   ParsedModel
}
