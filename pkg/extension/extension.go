package extension

import (
  "strings"

  set "github.com/deckarep/golang-set/v2"
)

var extImage = set.NewSet("jpg", "jpeg", "png", "svg")

func IsImage(filename string) bool {
  parts := strings.Split(filename, ".")

  if len(parts) < 2 {
    return false
  }
  ext := parts[len(parts)-1]

  return extImage.ContainsOne(ext)
}
