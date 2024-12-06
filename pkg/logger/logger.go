package logger

import (
  log "github.com/sirupsen/logrus"
  "github.com/ushakovn/outfit/pkg/env"
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
    f log.Formatter
    l log.Level
  )

  if env.IsProduction() {
    f = new(log.JSONFormatter)
    l = log.InfoLevel
  } else {
    f = new(log.TextFormatter)
    l = log.DebugLevel
  }

  log.SetFormatter(formatter{
    fields: fields,
    format: f,
  })

  log.SetLevel(l)
  log.SetReportCaller(true)
}
