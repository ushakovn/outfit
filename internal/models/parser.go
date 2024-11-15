package models

import (
  "context"

  "github.com/go-playground/validator/v10"
)

type ParseParams struct {
  URL      string               `bson:"url" json:"url" validate:"required"`
  Sizes    ParseSizesParams     `bson:"sizes" json:"sizes"`
  Discount *ParseDiscountParams `bson:"discount" json:"discount"`
}

type ParseSizesParams struct {
  Values []string `bson:"values" json:"values" validate:"required"`
}

type ParseDiscountParams struct {
  Percent int64 `bson:"percent" json:"percent" validate:"required"`
}

func (p *ParseParams) Validate() error {
  return validator.New().Struct(p)
}

func (p *ParseParams) HasDiscount() bool {
  return p.Discount != nil && p.Discount.Percent > 0
}

type Parser interface {
  Parse(ctx context.Context, params ParseParams) (*Product, error)
}
