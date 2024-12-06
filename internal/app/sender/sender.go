package sender

import (
  "context"
  "fmt"
  "strings"

  telegram "github.com/go-telegram/bot"
  tgmodels "github.com/go-telegram/bot/models"
  "github.com/ushakovn/outfit/internal/deps/storage/mongodb"
  "github.com/ushakovn/outfit/internal/models"
)

type Sender struct {
  config Config
  deps   Dependencies
}

type Config struct {
  IsCron      bool
  ProductType models.ProductType
}

type Dependencies struct {
  Telegram *telegram.Bot
  Mongodb  *mongodb.Client
}

func NewSenderCron(typ models.ProductType, deps Dependencies) *Sender {
  return &Sender{
    config: Config{
      IsCron:      true,
      ProductType: typ,
    },
    deps: deps,
  }
}

func (s *Sender) Start(ctx context.Context) error {
  if !s.config.IsCron {
    return fmt.Errorf("method called without cron flag")
  }
  if s.config.ProductType == "" {
    return fmt.Errorf("product type not specified")
  }

  err := s.deps.Mongodb.Scan(ctx, mongodb.ScanParams{
    CommonParams: mongodb.CommonParams{
      Database:   "outfit",
      Collection: "messages",
      StructType: models.SendableMessage{},
    },
    Filters: map[string]any{
      "type":         models.ProductDiffSendableType,
      "sent_id":      nil,
      "product.type": s.config.ProductType,
    },
    Callback: func(ctx context.Context, value any) error {
      message, ok := value.(*models.SendableMessage)
      if !ok {
        return fmt.Errorf("cast %v with type: %[1]T to: %T failed", value, new(models.SendableMessage))
      }
      return s.handleSendableMessage(ctx, message)
    },
  })
  if err != nil {
    return fmt.Errorf("s.deps.Mongodb.Scan: %w", err)
  }
  return nil
}

func (s *Sender) handleSendableMessage(ctx context.Context, message *models.SendableMessage) error {
  sent, err := s.deps.Telegram.SendMessage(ctx, &telegram.SendMessageParams{
    ChatID:    message.ChatId,
    Text:      strings.TrimSpace(message.Text.Value),
    ParseMode: tgmodels.ParseModeHTML,
  })
  if err != nil {
    return fmt.Errorf("s.deps.Telegram.SendMessage: %w", err)
  }
  message.SetAsSent(sent.ID)

  if err = s.updateSendableMessage(ctx, message); err != nil {
    return fmt.Errorf("s.updateSendableMessage: %w", err)
  }

  return nil
}

func (s *Sender) updateSendableMessage(ctx context.Context, message *models.SendableMessage) error {
  _, err := s.deps.Mongodb.Update(ctx, mongodb.UpdateParams{
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
    return fmt.Errorf("s.deps.Mongodb.Update: %w", err)
  }

  return nil
}
