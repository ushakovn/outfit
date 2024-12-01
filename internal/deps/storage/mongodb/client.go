package mongodb

import (
  "context"
  "errors"
  "fmt"
  "net/http"
  "strings"

  "github.com/go-playground/validator/v10"
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
