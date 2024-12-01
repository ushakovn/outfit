package mongodb

import (
  "context"
  "errors"
  "fmt"
  "reflect"

  log "github.com/sirupsen/logrus"
  "go.mongodb.org/mongo-driver/bson"
  "go.mongodb.org/mongo-driver/mongo"
  "go.mongodb.org/mongo-driver/mongo/options"
  "go.mongodb.org/mongo-driver/mongo/writeconcern"
)

type CommonParams struct {
  Database   string
  Collection string
  StructType any
}

type ScanParams struct {
  CommonParams

  Filters  map[string]any
  Callback func(ctx context.Context, value any) error
}

func (p *ScanParams) toFilters() bson.D {
  return makeBsonDFilters(p.Filters)
}

func (c *Client) Scan(ctx context.Context, params ScanParams) error {
  filters := params.toFilters()

  cursor, err := c.client.
    Database(params.Database).
    Collection(params.Collection).
    Find(ctx, filters)

  if err != nil {
    return fmt.Errorf("c.client.Database.Collection.Find: %w", err)
  }

  defer func() {
    if err = cursor.Close(ctx); err != nil {
      log.Error("mongodb.Telegram: cursor.Close: %v", err)
    }
  }()

  for cursor.Next(ctx) {
    doc := any(make(map[string]any))

    if params.StructType != nil {
      typ := reflect.TypeOf(params.StructType)
      doc = reflect.New(typ).Interface()
    }

    if err = cursor.Decode(doc); err != nil {
      return fmt.Errorf("cursor.Decode: %T: %w", doc, err)
    }

    if err = params.Callback(ctx, doc); err != nil {
      return fmt.Errorf("params.Callback: %T: %w", doc, err)
    }
  }

  return nil
}

type UpdateParams struct {
  GetParams

  Document any
}

func (p *UpdateParams) toFilters() bson.D {
  return makeBsonDFilters(p.GetParams.Filters)
}

func (p *UpdateParams) toUpdates() bson.D {
  return makeBsonDUpdates(p.Document)
}

func (c *Client) Upsert(ctx context.Context, params UpdateParams) (id any, err error) {
  res, err := c.Get(ctx, params.GetParams)
  if err != nil {
    if errors.Is(err, ErrNotFound) {

      res, err = c.Insert(ctx, InsertParams{
        CommonParams: params.CommonParams,
        Document:     params.Document,
      })
      if err != nil {
        return nil, fmt.Errorf("c.Insert: %w", err)
      }

      return res, nil
    }

    return nil, fmt.Errorf("c.Get: %w", err)
  }

  res, err = c.Update(ctx, UpdateParams{
    GetParams: params.GetParams,
    Document:  params.Document,
  })
  if err != nil {
    return nil, fmt.Errorf("c.Insert: %w", err)
  }

  return res, nil
}

func (c *Client) Update(ctx context.Context, params UpdateParams) (id any, err error) {
  filters := params.toFilters()
  updates := params.toUpdates()

  res, err := c.client.
    Database(params.Database).
    Collection(params.Collection).
    UpdateOne(ctx, filters, updates)

  if err != nil {
    return nil, fmt.Errorf("c.client.Database.Collection.UpdateOne: %w", err)
  }

  return res.UpsertedID, nil

}

type InsertParams struct {
  CommonParams

  Document any
}

func (c *Client) Insert(ctx context.Context, params InsertParams) (id any, err error) {
  res, err := c.client.
    Database(params.Database).
    Collection(params.Collection).
    InsertOne(ctx, params.Document)

  if err != nil {
    return nil, fmt.Errorf("c.client.Database.Collection.InsertOne: %w", err)
  }

  return res.InsertedID, nil
}

type GetParams struct {
  CommonParams

  Filters map[string]any
}

func (c *Client) Get(ctx context.Context, params GetParams) (any, error) {
  out, err := c.Find(ctx, FindParams{
    CommonParams: params.CommonParams,
    Filters:      params.Filters,
    Limit:        1,
  })
  if err != nil {
    return nil, fmt.Errorf("c.Find: %w", err)
  }

  if len(out) == 0 {
    return nil, ErrNotFound
  }

  return out[0], nil
}

type FindParams struct {
  CommonParams

  Filters map[string]any
  Limit   int64
}

func (p *FindParams) toFilters() bson.D {
  return makeBsonDFilters(p.Filters)
}

func (p *FindParams) toOptions() *options.FindOptions {
  opts := options.Find()

  if p.Limit != 0 {
    opts.SetLimit(p.Limit)
  }
  return opts
}

func (c *Client) Find(ctx context.Context, params FindParams) ([]any, error) {
  filters := params.toFilters()
  opts := params.toOptions()

  cursor, err := c.client.
    Database(params.Database).
    Collection(params.Collection).
    Find(ctx, filters, opts)

  if err != nil {
    return nil, fmt.Errorf("c.client.Database.Collection.Find: %w", err)
  }

  defer func() {
    if err = cursor.Close(ctx); err != nil {
      log.Error("mongodb.Telegram: cursor.Close: %v", err)
    }
  }()

  out := make([]any, 0, params.Limit)

  for cursor.Next(ctx) {
    doc := any(make(map[string]any))

    if params.StructType != nil {
      typ := reflect.TypeOf(params.StructType)
      doc = reflect.New(typ).Interface()
    }

    if err = cursor.Decode(doc); err != nil {
      return nil, fmt.Errorf("cursor.Decode: %T: %w", doc, err)
    }

    out = append(out, doc)
  }

  return out, nil
}

type DeleteParams struct {
  CommonParams

  Filters map[string]any
}

func (p *DeleteParams) toFilters() bson.D {
  return makeBsonDFilters(p.Filters)
}

func (c *Client) Delete(ctx context.Context, params DeleteParams) (count int64, err error) {
  filters := params.toFilters()

  res, err := c.client.
    Database(params.Database).
    Collection(params.Collection).
    DeleteMany(ctx, filters)

  if err != nil {
    return 0, fmt.Errorf("c.client.Database.Collection.Delete: %w", err)
  }

  return res.DeletedCount, nil
}

// WithTransaction работает только с replica set.
func (c *Client) WithTransaction(ctx context.Context, callback func(txCtx context.Context) error) error {
  writeConcern := writeconcern.Majority()

  txOptions := options.
    Transaction().
    SetWriteConcern(writeConcern)

  session, err := c.client.StartSession()
  if err != nil {
    return fmt.Errorf("c.client.StartSession: %w", err)
  }
  defer session.EndSession(ctx)

  wrappedCallback := func(sessionCtx mongo.SessionContext) (any, error) {
    err = callback(sessionCtx)

    return nil, err
  }

  _, err = session.WithTransaction(ctx, wrappedCallback, txOptions)
  if err != nil {
    return fmt.Errorf("session.WithTransaction: %w", err)
  }

  return nil
}
