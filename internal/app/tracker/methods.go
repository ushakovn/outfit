package tracker

import (
  "context"
  "fmt"

  "github.com/samber/lo"
  "github.com/ushakovn/outfit/internal/deps/storage/mongodb"
  "github.com/ushakovn/outfit/internal/models"
)

func (c *Tracker) Start(ctx context.Context) error {
  if !c.config.IsCron {
    return fmt.Errorf("method called without cron flag")
  }
  if c.config.ProductType == "" {
    return fmt.Errorf("product type not specified")
  }

  err := c.deps.Mongodb.Scan(ctx,
    mongodb.ScanParams{
      CommonParams: mongodb.CommonParams{
        Database:   "outfit",
        Collection: "trackings",
        StructType: models.Tracking{},
      },
      Callback: func(ctx context.Context, value any) error {
        tracking, ok := value.(*models.Tracking)
        if !ok {
          return fmt.Errorf("cast %v with type: %[1]T to: %T failed", value, new(models.Tracking))
        }
        return c.handleTracking(ctx, tracking)
      },
      Filters: map[string]any{
        "parsed_product.type": c.config.ProductType,
      },
    },
  )
  if err != nil {
    return fmt.Errorf("c.deps.Mongodb.Scan: %w", err)
  }

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
    return nil, fmt.Errorf("parser.Pars: %T: %w", parser, err)
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
    URL:      tracking.URL,
    Sizes:    tracking.Sizes,
    Discount: tracking.Discount,
  })
  if err != nil {
    return fmt.Errorf("parser.Pars: %T: %w", parser, err)
  }

  diff := models.NewProductDiff(tracking.ParsedProduct, *parsed)

  result := models.Sendable(tracking.ChatId).
    SetProductPtr(parsed).
    SetProductDiffPtr(diff).
    BuildProductDiffMessage()

  if !result.IsValid {
    return nil
  }

  if err = c.insertMessage(ctx, result.Message); err != nil {
    return fmt.Errorf("c.insertMessage: %w", err)
  }

  // Проставляем актуальный продукт
  tracking.ParsedProduct = lo.FromPtr(parsed)

  if err = c.upsertTracking(ctx, tracking); err != nil {
    return fmt.Errorf("c.upsertTracking: %w", err)
  }

  return nil
}
