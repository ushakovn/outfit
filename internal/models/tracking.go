package models

import "time"

type Tracking struct {
  ChatId        int64              `bson:"chat_id" json:"chat_id"`
  URL           string             `bson:"url" json:"url"`
  Sizes         ParseSizesParams   `bson:"sizes" json:"sizes"`
  ParsedProduct Product            `bson:"parsed_product" json:"parsed_product"`
  Flags         TrackingFlags      `bson:"flags" json:"flags"`
  Timestamps    TrackingTimestamps `bson:"timestamps" json:"timestamps"`
}

type TrackingFlags struct {
  WithOptional bool `bson:"with_optional" json:"with_optional"`
}

type TrackingTimestamps struct {
  CreatedAt time.Time  `bson:"created_at" json:"created_at"`
  HandledAt *time.Time `bson:"handled_at" json:"handled_at"`
}
