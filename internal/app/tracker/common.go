package tracker

import (
  "context"
  "fmt"

  "github.com/ushakovn/outfit/internal/deps/storage/mongodb"
  "github.com/ushakovn/outfit/internal/models"
)

func (c *Tracker) insertMessage(ctx context.Context, message models.SendableMessage) error {
  _, err := c.deps.Mongodb.Insert(ctx, mongodb.InsertParams{
    CommonParams: mongodb.CommonParams{
      Database:   "outfit",
      Collection: "messages",
    },
    Document: message,
  })
  if err != nil {
    return fmt.Errorf("c.deps.Mongodb.Insert: %w", err)
  }

  return nil
}

func (c *Tracker) upsertTracking(ctx context.Context, tracking *models.Tracking) error {
  _, err := c.deps.Mongodb.Upsert(ctx, mongodb.UpdateParams{
    GetParams: mongodb.GetParams{
      CommonParams: mongodb.CommonParams{
        Database:   "outfit",
        Collection: "trackings",
        StructType: tracking,
      },
      Filters: map[string]any{
        "chat_id": tracking.ChatId,
        "url":     tracking.URL,
      },
    },
    Document: tracking,
  })
  if err != nil {
    return fmt.Errorf("c.deps.Mongodb.Upsert: %w", err)
  }

  return nil
}

func (c *Tracker) findParser(productURL string) (models.Parser, error) {
  productType := models.ProductTypeByURL(productURL)

  parser, ok := c.deps.Parsers[productType]
  if !ok {
    return nil, fmt.Errorf("%w: not found parser for product type: %s. url: %s",
      ErrUnsupportedProductType, productType, productURL)
  }

  return parser, nil
}
