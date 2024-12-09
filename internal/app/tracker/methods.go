package tracker

import (
  "context"
  "fmt"

  log "github.com/sirupsen/logrus"
  "github.com/ushakovn/outfit/internal/deps/storage/mongodb"
  "github.com/ushakovn/outfit/internal/models"
  "github.com/ushakovn/outfit/pkg/worker"
)

func (c *Tracker) Start(ctx context.Context) error {
  if !c.config.IsCron {
    return fmt.Errorf("method called without cron flag")
  }

  log.
    WithField("product_type", c.config.ProductType).
    Info("tracker cron starting")

  pool := worker.NewPool(ctx, worker.DefaultCount)

  err := c.deps.Mongodb.Scan(ctx, mongodb.ScanParams{
    CommonParams: mongodb.CommonParams{
      Database:   "outfit",
      Collection: "trackings",
      StructType: models.Tracking{},
    },
    Filters: c.makeTrackingFilters(),

    Callback: func(ctx context.Context, value any) error {
      tracking, ok := value.(*models.Tracking)
      if !ok {
        log.
          WithField("tracking.value", value).
          Errorf("cast tracking %v with type: %[1]T to: %T failed", value, new(models.Tracking))

        return nil
      }

      log.
        WithFields(log.Fields{
          "tracking.url":     tracking.URL,
          "tracking.chat_id": tracking.ChatId,
        }).
        Info("scanned tracking from mongodb collection")

      pool.Push(func(ctx context.Context) error {
        if err := c.handleTracking(ctx, tracking); err != nil {
          log.
            WithFields(log.Fields{
              "tracking.url":     tracking.URL,
              "tracking.chat_id": tracking.ChatId,
            }).
            Errorf("tracking handle failed: %v", err)

          return nil
        }

        log.
          WithFields(log.Fields{
            "tracking.url":     tracking.URL,
            "tracking.chat_id": tracking.ChatId,
          }).
          Info("tracking handled successfully")

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
    Info("tracker cron completed successfully")

  return nil
}

func (c *Tracker) CheckProductURL(url string) error {
  if _, err := c.findParser(url); err != nil {
    return fmt.Errorf("c.findParser: %w", err)
  }
  return nil
}

type CreateMessageParams struct {
  ChatId   int64
  URL      string
  Sizes    models.ParseSizesParams
  Discount *models.ParseDiscountParams
}

func (c *Tracker) CreateMessage(ctx context.Context, params CreateMessageParams) (*models.SendableMessage, error) {
  parser, err := c.findParser(params.URL)
  if err != nil {
    return nil, fmt.Errorf("c.findParser: %w", err)
  }

  parsed, err := parser.Parse(ctx, models.ParseParams{
    URL:      params.URL,
    Sizes:    params.Sizes,
    Discount: params.Discount,
  })
  if err != nil {
    return nil, fmt.Errorf("parser.Parse: %T: %w", parser, err)
  }

  result := models.Sendable(params.ChatId).
    SetProductPtr(parsed).
    BuildProductMessage()

  return &result.Message, nil
}

func (c *Tracker) handleTracking(ctx context.Context, tracking *models.Tracking) error {
  parser, err := c.findParser(tracking.ParsedProduct.URL)
  if err != nil {
    return fmt.Errorf("c.findParser: %w", err)
  }

  parsed, err := parser.Parse(ctx, models.ParseParams{
    URL:   tracking.URL,
    Sizes: tracking.Sizes,
  })
  if err != nil {
    return fmt.Errorf("parser.Parse: %T: %w", parser, err)
  }

  diff := models.NewProductDiff(tracking.ParsedProduct, *parsed)

  result := models.Sendable(tracking.ChatId).
    SetTrackingPtr(tracking).
    SetProductPtr(parsed).
    SetProductDiffPtr(diff).
    BuildProductDiffMessage()

  if !result.IsValid {
    return nil
  }

  if err = c.insertMessageIfNotExist(ctx, result.Message); err != nil {
    return fmt.Errorf("c.insertMessageIfNotExist: %w", err)
  }

  setTrackingUpdates(tracking, parsed)

  if err = c.upsertTracking(ctx, tracking); err != nil {
    return fmt.Errorf("c.upsertTracking: %w", err)
  }

  return nil
}
