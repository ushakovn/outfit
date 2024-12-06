package hasher

import (
  "crypto/sha256"
  "fmt"
)

func SHA256(value string) string {
  hash := sha256.New()
  hash.Write([]byte(value))

  return fmt.Sprintf("%x", hash.Sum(nil))
}
