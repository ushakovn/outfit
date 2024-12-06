package tracker

import (
  "context"
  "fmt"

  "github.com/ushakovn/outfit/internal/deps/storage/mongodb"
  "github.com/ushakovn/outfit/internal/models"
)

func (c *Tracker) insertMessageIfNotExist(ctx context.Context, message models.SendableMessage) error {
  common := mongodb.CommonParams{
    Database:   "outfit",
    Collection: "messages",
    StructType: message,
  }

  res, err := c.deps.Mongodb.Find(ctx, mongodb.FindParams{
    CommonParams: common,
    Filters: map[string]any{
      "chat_id":     message.ChatId,
      "text.sha256": message.Text.SHA256,
    },
    Limit: 1,
  })
  if err != nil {
    return fmt.Errorf("c.deps.Mongodb.Find: %w", err)
  }

  if len(res) != 0 {
    return nil
  }

  _, err = c.deps.Mongodb.Insert(ctx, mongodb.InsertParams{
    CommonParams: common,
    Document:     message,
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
