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
  "go.uber.org/atomic"
)

type CommonParams struct {
  Database   string
  Collection string
  StructType any
}

type SortParams struct {
  Field string
  Order SortOrder
}

type SortOrder int

const (
  SortOrderAsc  SortOrder = 1
  SortOrderDesc SortOrder = -1
)

type ScanParams struct {
  CommonParams

  Filters map[string]any
  Sorting []SortParams

  Callback func(ctx context.Context, value any) error
}

func (p *ScanParams) toFilters() bson.D {
  return makeBsonDFilters(p.Filters)
}

func (p *ScanParams) toOptions() *options.FindOptions {
  opts := options.Find()

  if len(p.Sorting) != 0 {
    sort := makeBsonBsonDSort(p.Sorting)
    opts.SetSort(sort)
  }

  return opts
}

func (c *Client) Scan(ctx context.Context, params ScanParams) error {
  log.
    WithFields(log.Fields{
      "params.database":   params.Database,
      "params.collection": params.Collection,
      "params.filters":    params.Filters,
    }).
    Info("mongodb collection scan starting")

  counter := atomic.NewUint32(0)

  filters := params.toFilters()
  opts := params.toOptions()

  cursor, err := c.client.
    Database(params.Database).
    Collection(params.Collection).
    Find(ctx, filters, opts)

  if err != nil {
    return fmt.Errorf("mongodb.Database.Collection.Find: %w", err)
  }

  defer func() {
    if err = cursor.Close(ctx); err != nil {
      log.Error("mongodb.Client: cursor.Close: %v", err)
    }
  }()

  for cursor.Next(ctx) {
    counter.Inc()

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

  log.
    WithFields(log.Fields{
      "params.database":    params.Database,
      "params.collection":  params.Collection,
      "params.filters":     params.Filters,
      "mongodb.find.count": counter.Load(),
    }).
    Info("mongodb collection scan completed")

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
      log.
        WithFields(log.Fields{
          "params.database":   params.Database,
          "params.collection": params.Collection,
          "params.filters":    params.Filters,
        }).
        Debug("document not found in mongodb collection. new document will be inserted")

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

  log.
    WithFields(log.Fields{
      "params.database":   params.Database,
      "params.collection": params.Collection,
      "params.filters":    params.Filters,
    }).
    Debug("document found in mongodb collection. existed document will be updated")

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

  log.
    WithFields(log.Fields{
      "params.database":   params.Database,
      "params.collection": params.Collection,
      "params.filters":    params.Filters,
    }).
    Debug("document in mongodb collection updated successfully")

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

  log.
    WithFields(log.Fields{
      "params.database":   params.Database,
      "params.collection": params.Collection,
    }).
    Debug("document inserted to mongodb collection successfully")

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
    log.
      WithFields(log.Fields{
        "params.database":   params.Database,
        "params.collection": params.Collection,
        "params.filters":    params.Filters,
      }).
      Debug("document not found in mongodb collection")

    return nil, ErrNotFound
  }

  return out[0], nil
}

type FindParams struct {
  CommonParams

  Filters map[string]any
  Sorting []SortParams

  Limit int64
}

func (p *FindParams) toFilters() bson.D {
  return makeBsonDFilters(p.Filters)
}

func (p *FindParams) toOptions() *options.FindOptions {
  opts := options.Find()

  if p.Limit != 0 {
    opts.SetLimit(p.Limit)
  }

  if len(p.Sorting) != 0 {
    sort := makeBsonBsonDSort(p.Sorting)
    opts.SetSort(sort)
  }

  return opts
}

func (c *Client) Find(ctx context.Context, params FindParams) ([]any, error) {
  log.
    WithFields(log.Fields{
      "params.database":   params.Database,
      "params.collection": params.Collection,
      "params.filters":    params.Filters,
    }).
    Debug("mongodb collection find starting")

  counter := atomic.NewUint32(0)

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

    counter.Inc()
  }

  log.
    WithFields(log.Fields{
      "params.database":    params.Database,
      "params.collection":  params.Collection,
      "params.filters":     params.Filters,
      "mongodb.find.count": counter.Load(),
    }).
    Debug("mongodb collection find completed")

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

  log.
    WithFields(log.Fields{
      "params.database":   params.Database,
      "params.collection": params.Collection,
      "params.filters":    params.Filters,
    }).
    Debug("documents deleted from mongodb collection successfully")

  return res.DeletedCount, nil
}

type TextSearchParams struct {
  CommonParams

  Query string
  Limit int64
}

func (c *Client) TextSearch(ctx context.Context, params TextSearchParams) ([]any, error) {
  filters := map[string]any{
    "$text": bson.D{
      {
        Key:   "$search",
        Value: params.Query,
      },
    },
  }

  res, err := c.Find(ctx, FindParams{
    CommonParams: params.CommonParams,
    Filters:      filters,
    Limit:        params.Limit,
  })
  if err != nil {
    return nil, fmt.Errorf("c.Find: %w", err)
  }

  return res, nil
}

type CreateIndexParams struct {
  CommonParams

  Parts   []IndexPart
  Options *options.IndexOptions
}

type IndexPart struct {
  Field string
  Type  any
}

const (
  IndexTypeAsc  = 1
  IndexTypeDesc = -1
  IndexTypeText = "text"
)

func (p *CreateIndexParams) toIndexModel() mongo.IndexModel {
  keys := make(bson.D, 0, len(p.Parts))

  for _, part := range p.Parts {
    keys = append(keys, bson.E{
      Key:   part.Field,
      Value: part.Type,
    })
  }

  return mongo.IndexModel{
    Keys:    keys,
    Options: p.Options,
  }
}

func (c *Client) CreateIndex(ctx context.Context, params CreateIndexParams) (name string, err error) {
  model := params.toIndexModel()

  name, err = c.client.
    Database(params.Database).
    Collection(params.Collection).
    Indexes().
    CreateOne(ctx, model)

  if err != nil {
    return "", fmt.Errorf("c.client.Database.Collection.Indexes.CreateOne: %w", err)
  }

  return name, nil
}
