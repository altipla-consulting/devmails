package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"
	"libs.altipla.consulting/errors"
	"libs.altipla.consulting/mjml"
	"libs.altipla.consulting/templates"
)

var (
	srcFolder    = flag.String("src", "src", "Source folder")
	outputFolder = flag.String("output", "output", "Output folder")
	dataFolder   = flag.String("data", "data", "Data folder")
	watch        = flag.Bool("watch", true, "Watch changes and reload every time")
)

func main() {
	if err := run(); err != nil {
		log.Fatal(errors.Stack(err))
	}
}

func run() error {
	flag.Parse()

	var files []string
	fn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.Trace(err)
		}

		if info.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".mjml" {
			log.WithField("path", path).Warning("Ignoring file with unknown extension")
			return nil
		}

		rel, err := filepath.Rel(*srcFolder, path)
		if err != nil {
			return errors.Trace(err)
		}
		files = append(files, rel[:len(rel)-len(filepath.Ext(rel))])

		return nil
	}
	if err := filepath.Walk(*srcFolder, fn); err != nil {
		return errors.Trace(err)
	}

	if len(files) == 0 {
		return errors.Errorf("no files to transform in the source folder")
	}

	if err := generate(files); err != nil {
		return errors.Trace(err)
	}

	return nil
}

func generate(files []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, file := range files {
		log.WithField("path", filepath.Join(*srcFolder, file+".mjml")).Info("Generate template")

		rawData, err := ioutil.ReadFile(filepath.Join(*dataFolder, file+".json"))
		if err != nil && !os.IsNotExist(err) {
			return errors.Trace(err)
		} else if err != nil {
			rawData = []byte("{}")
		}

		var data interface{}
		if err := json.Unmarshal(rawData, &data); err != nil {
			return errors.Trace(err)
		}

		tmpl, err := templates.Load(filepath.Join(*srcFolder, file+".mjml"))
		if err != nil {
			return errors.Trace(err)
		}

		var buf bytes.Buffer
		if err := tmpl.ExecuteTemplate(&buf, filepath.Base(file)+".mjml", data); err != nil {
			return errors.Trace(err)
		}

		output, err := mjml.Render(ctx, buf.String())
		if err != nil {
			return errors.Trace(err)
		}

		destFilename := filepath.Join(*outputFolder, file+".html")
		if err := os.MkdirAll(filepath.Dir(destFilename), 0700); err != nil {
			return errors.Trace(err)
		}
		if err := ioutil.WriteFile(destFilename, []byte(output), 0600); err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}
