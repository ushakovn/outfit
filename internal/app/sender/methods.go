package sender

import (
  "context"
  "fmt"
  "strings"

  telegram "github.com/go-telegram/bot"
  tgmodels "github.com/go-telegram/bot/models"
  log "github.com/sirupsen/logrus"
  "github.com/ushakovn/outfit/internal/deps/storage/mongodb"
  "github.com/ushakovn/outfit/internal/models"
  "github.com/ushakovn/outfit/pkg/worker"
)

func (c *Sender) Start(ctx context.Context) error {
  if !c.config.IsCron {
    return fmt.Errorf("method called without cron flag")
  }

  log.
    WithField("product_type", c.config.ProductType).
    Info("sender cron starting")

  pool := worker.NewPool(ctx, worker.DefaultCount)

  err := c.deps.Mongodb.Scan(ctx, mongodb.ScanParams{
    CommonParams: mongodb.CommonParams{
      Database:   "outfit",
      Collection: "messages",
      StructType: models.SendableMessage{},
    },
    Filters: c.makeMessagesFilters(),

    Callback: func(ctx context.Context, value any) error {
      message, ok := value.(*models.SendableMessage)
      if !ok {
        log.
          WithField("message.value", value).
          Errorf("cast message %v with type: %[1]T to: %T failed", value, new(models.SendableMessage))

        return nil
      }

      log.
        WithFields(log.Fields{
          "message.uuid":        message.UUID,
          "message.chat_id":     message.ChatId,
          "message.product.url": message.Product.URL,
        }).
        Info("scanned message from mongodb collection")

      pool.Push(func(ctx context.Context) error {
        if err := c.handleSendableMessage(ctx, message); err != nil {
          log.
            WithFields(log.Fields{
              "message.uuid":        message.UUID,
              "message.chat_id":     message.ChatId,
              "message.product.url": message.Product.URL,
            }).
            Errorf("sendable message handle failed: %v", err)

          return nil
        }

        log.
          WithFields(log.Fields{
            "message.uuid":        message.UUID,
            "message.chat_id":     message.ChatId,
            "message.product.url": message.Product.URL,
          }).
          Info("message handled successfully")

        return nil
      })

      return nil
    },
  })
  if err != nil {
    return fmt.Errorf("c.deps.Mongodb.Scan: %w", err)
  }

  pool.StopWait()

  log.
    WithField("product_type", c.config.ProductType).
    Info("sender cron completed successfully")

  return nil
}

func (c *Sender) handleSendableMessage(ctx context.Context, message *models.SendableMessage) error {
  sent, err := c.deps.Telegram.SendMessage(ctx, &telegram.SendMessageParams{
    ChatID:    message.ChatId,
    Text:      strings.TrimSpace(message.Text.Value),
    ParseMode: tgmodels.ParseModeHTML,
  })
  if err != nil {
    return fmt.Errorf("c.deps.Telegram.SendMessage: %w", err)
  }

  log.
    WithFields(log.Fields{
      "message.uuid":        message.UUID,
      "message.chat_id":     message.ChatId,
      "message.sent_id":     sent.ID,
      "message.product.url": message.Product.URL,
    }).
    Info("message sent to telegram chat")

  message.SetAsSent(sent.ID)

  if err = c.updateSendableMessage(ctx, message); err != nil {
    return fmt.Errorf("c.updateSendableMessage: %w", err)
  }

  return nil
}
