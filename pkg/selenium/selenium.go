package selenium

import (
  "fmt"

  "github.com/tebeka/selenium"
  "github.com/tebeka/selenium/chrome"
)

type Config struct {
  Path string
  Port int
}

type Chrome struct {
  svc    *selenium.Service
  driver selenium.WebDriver
}

func NewChrome(config Config, opts ...selenium.ServiceOption) (*Chrome, error) {
  svc, err := selenium.NewChromeDriverService(config.Path, config.Port, opts...)
  if err != nil {
    return nil, fmt.Errorf("selenium.NewChromeDriverService: %w", err)
  }

  var (
    caps selenium.Capabilities
    crm  chrome.Capabilities
  )

  caps.AddChrome(crm)

  driver, err := selenium.NewRemote(caps, "")
  if err != nil {
    return nil, fmt.Errorf("selenium.NewRemote: %w", err)
  }

  if err = driver.MaximizeWindow(""); err != nil {
    return nil, fmt.Errorf("driver.MaximizeWindow: %w", err)
  }

  return &Chrome{
    svc:    svc,
    driver: driver,
  }, nil
}

func (c *Chrome) Stop() error {
  return c.svc.Stop()
}

func (c *Chrome) WebDriver() selenium.WebDriver {
  return c.driver
}
