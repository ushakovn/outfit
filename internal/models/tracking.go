package models

type Tracking struct {
  Telegram Telegram             `bson:"telegram" json:"telegram"`
  Comment  string               `bson:"comment" json:"comment"`
  URL      string               `bson:"url" json:"url"`
  Sizes    ParseSizesParams     `bson:"sizes" json:"sizes"`
  Discount *ParseDiscountParams `bson:"discount" json:"discount"`
  Product  *Product             `bson:"product" json:"product"`
}

type TrackingMessage struct {
  Telegram    Telegram    `bson:"telegram" json:"telegram"`
  TextValue   string      `bson:"text_value" json:"text_value"`
  Product     Product     `bson:"product" json:"product"`
  ProductDiff ProductDiff `bson:"product_diff" json:"product_diff"`
}

type Telegram struct {
  ChatID int64 `bson:"chat_id" json:"chat_id"`
}
