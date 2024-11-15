package validator

import "net/url"

func URL(value string) error {
  _, err := url.ParseRequestURI(value)
  return err
}
