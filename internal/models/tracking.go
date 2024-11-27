package models

type Tracking struct {
	ChatId        int64                `bson:"chat_id" json:"chat_id"`
	URL           string               `bson:"url" json:"url"`
	Sizes         ParseSizesParams     `bson:"sizes" json:"sizes"`
	Discount      *ParseDiscountParams `bson:"discount" json:"discount"`
	ParsedProduct Product              `bson:"parsed_product" json:"parsed_product"`
}

type TrackingMessage struct {
	TextValue   string      `bson:"text_value" json:"text_value"`
	Product     Product     `bson:"product" json:"product"`
	ProductDiff ProductDiff `bson:"product_diff" json:"product_diff"`
}

type TrackingMessageParams struct {
	URL      string
	Sizes    ParseSizesParams
	Discount *ParseDiscountParams
}
