package tracker

import (
  "context"
  "errors"
  "fmt"

  "github.com/ushakovn/outfit/internal/message"
  "github.com/ushakovn/outfit/internal/models"
  "github.com/ushakovn/outfit/internal/provider/mongodb"
)

var ErrUnsupportedProductType = errors.New("unsupported product type")

type Tracker struct {
  config Config
  deps   Dependencies
}

type Config struct {
  IsCron      bool
  ProductType models.ProductType
}

type Dependencies struct {
  Mongodb *mongodb.Client
  Parsers map[models.ProductType]models.Parser
}

func NewTracker(config Config, deps Dependencies) *Tracker {
  return &Tracker{
    config: config,
    deps:   deps,
  }
}

func (c *Tracker) StartTrackingHandle(ctx context.Context) error {
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

func (c *Tracker) CreateTrackingMessage(ctx context.Context, params models.TrackingMessageParams) (*models.TrackingMessage, error) {
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

  result := message.Do().
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

  result := message.Do().
    SetProductPtr(parsed).
    SetProductDiffPtr(diff).
    BuildDiffMessage()

  if !result.IsSendable {
    return nil
  }

  if err = c.insertMessage(ctx, result.Message); err != nil {
    return fmt.Errorf("c.insertMessage: %w", err)
  }

  return nil
}

func (c *Tracker) insertMessage(ctx context.Context, message models.TrackingMessage) error {
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

func (c *Tracker) findParser(productURL string) (models.Parser, error) {
  productType := models.ProductTypeByURL(productURL)

  parser, ok := c.deps.Parsers[productType]
  if !ok {
    return nil, fmt.Errorf("%w: not found parser for product type: %s. url: %s",
      ErrUnsupportedProductType, productType, productURL)
  }

  return parser, nil
}
