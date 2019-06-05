package main

import (
  log "github.com/sirupsen/logrus"
  "libs.altipla.consulting/errors"
)

func main() {
  if err := run(); err != nil {
    log.Fatal(errors.Stack(err))
  }
}

func run() error {
  return nil
}
