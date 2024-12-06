package models

type Tracking struct {
  ChatId        int64            `bson:"chat_id" json:"chat_id"`
  URL           string           `bson:"url" json:"url"`
  Sizes         ParseSizesParams `bson:"sizes" json:"sizes"`
  ParsedProduct Product          `bson:"parsed_product" json:"parsed_product"`
  Flags         TrackingFlags    `bson:"flags" json:"flags"`
}

type TrackingFlags struct {
  WithOptional bool `bson:"with_optional" json:"with_optional"`
}
