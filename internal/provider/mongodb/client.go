package mongodb

import (
  "context"
  "errors"
  "fmt"
  "net/http"
  "reflect"
  "strings"

  "github.com/go-playground/validator/v10"
  log "github.com/sirupsen/logrus"
  "go.mongodb.org/mongo-driver/bson"
  "go.mongodb.org/mongo-driver/mongo"
  "go.mongodb.org/mongo-driver/mongo/options"
)

var ErrNotFound = errors.New("document not found")

type Client struct {
  client *mongo.Client
}

type Config struct {
  Host           string          `validate:"required"`
  Port           string          `validate:"required"`
  Authentication *Authentication `validate:"required"`
}

type Authentication struct {
  User     string `validate:"required"`
  Password string `validate:"required"`
}

func (c *Config) Validate() error {
  return validator.New().Struct(c)
}

type Dependencies struct {
  Client *http.Client `validate:"required"`
}

func (c *Dependencies) Validate() error {
  return validator.New().Struct(c)
}

func (c *Config) ConnectionString() string {
  sb := strings.Builder{}

  write := func(s string) {
    sb.WriteString(s)
  }

  sb.WriteString("mongodb://")

  if c.Authentication != nil {
    write(c.Authentication.User)
    write(":")
    write(c.Authentication.Password)
    write("@")
  }

  write(c.Host)
  write(":")
  write(c.Port)

  return sb.String()
}

func NewClient(ctx context.Context, config Config, deps Dependencies) (*Client, error) {
  if err := deps.Validate(); err != nil {
    return nil, fmt.Errorf("invalid dependencies: %w", err)
  }
  if err := config.Validate(); err != nil {
    return nil, fmt.Errorf("invalid config: %w", err)
  }

  opts := options.
    Client().
    SetHTTPClient(deps.Client).
    ApplyURI(config.ConnectionString())

  client, err := mongo.Connect(ctx, opts)
  if err != nil {
    return nil, fmt.Errorf("mongo.Connect: %w", err)
  }

  if err = client.Ping(ctx, nil); err != nil {
    return nil, fmt.Errorf("client.Ping: %w", err)
  }

  return &Client{
    client: client,
  }, nil
}

type CommonParams struct {
  Database   string
  Collection string
  StructType any
}

type ScanParams struct {
  CommonParams

  Callback func(ctx context.Context, value any) error
}

func (c *Client) Scan(ctx context.Context, params ScanParams) error {
  cursor, err := c.client.
    Database(params.Database).
    Collection(params.Collection).
    Find(ctx, bson.D{})

  if err != nil {
    return fmt.Errorf("c.client.Database.Collection.Find: %w", err)
  }

  defer func() {
    if err = cursor.Close(ctx); err != nil {
      log.Error("mongodb.Telegram: cursor.Close: %v", err)
    }
  }()

  for cursor.Next(ctx) {
    typ := reflect.TypeOf(params.StructType)
    doc := reflect.New(typ).Interface()

    if err = cursor.Decode(&doc); err != nil {
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
    typ := reflect.TypeOf(params.StructType)
    doc := reflect.New(typ).Interface()

    if err = cursor.Decode(doc); err != nil {
      return nil, fmt.Errorf("cursor.Decode: %T: %w", doc, err)
    }

    out = append(out, doc)
  }

  return out, nil
}

func makeBsonDUpdates(value any) bson.D {
  updates := bson.D{}

  typ := reflect.TypeOf(value)

  values := reflect.ValueOf(value)

  for i := 1; i < typ.NumField(); i++ {
    field := typ.Field(i)

    val := values.Field(i)
    tag := field.Tag.Get("json")

    if !isZeroType(val) {
      update := bson.E{
        Key:   tag,
        Value: val.Interface(),
      }
      updates = append(updates, update)
    }
  }

  return bson.D{{
    Key:   "$set",
    Value: updates,
  }}
}

func makeBsonDFilters(kv map[string]any) bson.D {
  out := bson.D{}

  for key, value := range kv {
    out = append(out, bson.E{
      Key:   key,
      Value: value,
    })
  }

  return out
}

func isZeroType(value reflect.Value) bool {
  zero := reflect.Zero(value.Type()).Interface()

  switch value.Kind() {
  case reflect.Slice, reflect.Array, reflect.Chan, reflect.Map:
    return value.Len() == 0
  default:
    return reflect.DeepEqual(zero, value.Interface())
  }
}
