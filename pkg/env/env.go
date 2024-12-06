package env

import "os"

type Env = string

const (
  DEV  Env = "DEV"
  PROD Env = "PROD"
)

func IsProduction() bool {
  return os.Getenv("ENV") == PROD
}
