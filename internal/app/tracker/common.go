package tracker

import (
  "context"
  "fmt"
  "time"

  "github.com/samber/lo"
  log "github.com/sirupsen/logrus"
  "github.com/ushakovn/outfit/internal/deps/storage/mongodb"
  "github.com/ushakovn/outfit/internal/models"
)

func (c *Tracker) makeTrackingFilters() map[string]any {
  filters := make(map[string]any)

  if c.config.ProductType != "" {
    filters["parsed_product.type"] = c.config.ProductType
  }
  return filters
}

func setTrackingUpdates(tracking *models.Tracking, product *models.Product) {
  tracking.ParsedProduct = lo.FromPtr(product)
  tracking.Timestamps.HandledAt = lo.ToPtr(time.Now())
}

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

  log.
    WithFields(log.Fields{
      "message.uuid":    message.UUID,
      "message.chat_id": message.ChatId,
    }).
    Info("new sendable message inserted to trackings mongodb collection")

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
  productType := models.FindProductType(productURL)

  parser, ok := c.deps.Parsers[productType]
  if !ok {
    return nil, fmt.Errorf("%w: not found parser for product type: %s. url: %s",
      ErrUnsupportedProductType, productType, productURL)
  }

  return parser, nil
}
