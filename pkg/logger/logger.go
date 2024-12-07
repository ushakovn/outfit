package logger

import (
  log "github.com/sirupsen/logrus"
  "github.com/ushakovn/boiler/pkg/env"
)

type formatter struct {
  format log.Formatter
  fields map[string]any
}

func (f formatter) Format(entry *log.Entry) ([]byte, error) {
  for k, v := range f.fields {
    if _, exists := entry.Data[k]; !exists {
      entry.Data[k] = v
    }
  }
  return f.format.Format(entry)
}

func Init() {
  InitWithFields(map[string]any{})
}

func InitWithFields(fields map[string]any) {
  var (
    format log.Formatter
    caller bool
  )

  switch env.AppEnv() {

  case env.ProductionEnv:
    format = new(log.JSONFormatter)
    caller = true

  default:
    format = new(log.TextFormatter)
    caller = false
  }

  log.SetFormatter(formatter{
    fields: fields,
    format: format,
  })
  log.SetLevel(log.InfoLevel)
  log.SetReportCaller(caller)
}
