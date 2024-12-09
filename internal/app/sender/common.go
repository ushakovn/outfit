package sender

import (
  "context"
  "fmt"

  "github.com/ushakovn/outfit/internal/deps/storage/mongodb"
  "github.com/ushakovn/outfit/internal/models"
)

func (c *Sender) makeMessagesFilters() map[string]any {
  filters := map[string]any{
    "type":    models.ProductDiffSendableType,
    "sent_id": nil,
  }
  if c.config.ProductType != "" {
    filters["product.type"] = c.config.ProductType
  }
  return filters
}

func (c *Sender) updateSendableMessage(ctx context.Context, message *models.SendableMessage) error {
  _, err := c.deps.Mongodb.Update(ctx, mongodb.UpdateParams{
    GetParams: mongodb.GetParams{
      CommonParams: mongodb.CommonParams{
        Database:   "outfit",
        Collection: "messages",
        StructType: models.SendableMessage{},
      },
      Filters: map[string]any{
        "uuid": message.UUID,
      },
    },
    Document: message,
  })
  if err != nil {
    return fmt.Errorf("c.deps.Mongodb.Update: %w", err)
  }

  return nil
}
