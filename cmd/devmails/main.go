package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/jaschaephraim/lrserver"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	if err := generate(ctx, files); err != nil {
		return errors.Trace(err)
	}

	if *watch {
		if err := runWatcher(ctx, cancel, files); err != nil {
			return errors.Trace(err)
		}
		fmt.Println()
		log.Println("Bye!")
	}

	return nil
}

func generate(ctx context.Context, files []string) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
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

		var output string

		tmpl, err := templates.Load(filepath.Join(*srcFolder, file+".mjml"))
		if err != nil {
			log.Warning(errors.Cause(err))
			output = `<h1>Error building template</h1><h3 style="color: tomato">` + errors.Cause(err).Error() + `</h3>`
		} else {
			var buf bytes.Buffer
			if err := tmpl.ExecuteTemplate(&buf, filepath.Base(file)+".mjml", data); err != nil {
				return errors.Trace(err)
			}

			output, err = mjml.Render(ctx, buf.String())
			if err != nil {
				return errors.Trace(err)
			}
		}

		output += `<script src="http://localhost:35700/livereload.js?snipver=1"></script>` + "\n"

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

func runWatcher(ctx context.Context, cancel context.CancelFunc, files []string) error {
	dataWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return errors.Trace(err)
	}
	srcWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return errors.Trace(err)
	}
	server := &http.Server{
		Addr: ":3000",
	}
	lr := lrserver.New(lrserver.DefaultName, 35700)
	lr.SetStatusLog(nil)
	lr.SetErrorLog(nil)

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		if err := lr.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return errors.Trace(err)
		}

		return nil
	})

	g.Go(func() error {
		http.Handle("/", http.FileServer(http.Dir(*outputFolder)))

		log.Println("Listening in localhost:3000...")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return errors.Trace(err)
		}

		return nil
	})

	g.Go(func() error {
		for {
			log.Info("Waiting for changes...")
			select {
			case ev := <-dataWatcher.Events:
				name, err := filepath.Rel(*dataFolder, ev.Name)
				if err != nil {
					return errors.Trace(err)
				}
				name = name[:len(name)-len(filepath.Ext(name))]

				if err := generate(ctx, []string{name}); err != nil {
					return errors.Trace(err)
				}

				lr.Reload(ev.Name)

			case ev := <-srcWatcher.Events:
				name, err := filepath.Rel(*srcFolder, ev.Name)
				if err != nil {
					return errors.Trace(err)
				}
				name = name[:len(name)-len(filepath.Ext(name))]

				if err := generate(ctx, []string{name}); err != nil {
					return errors.Trace(err)
				}

				lr.Reload(ev.Name)

			case err := <-dataWatcher.Errors:
				return errors.Trace(err)

			case err := <-srcWatcher.Errors:
				return errors.Trace(err)

			case <-ctx.Done():
			}

			if ctx.Err() != nil {
				return nil
			}
		}
	})

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	g.Go(func() error {
		select {
		case <-c:
			cancel()
			signal.Stop(c)
			if err := server.Close(); err != nil {
				return errors.Trace(err)
			}
			if err := lr.Close(); err != nil {
				return errors.Trace(err)
			}
			return nil
		}
	})

	for _, file := range files {
		if err := srcWatcher.Add(filepath.Join(*srcFolder, file+".mjml")); err != nil {
			return errors.Wrapf(err, "file: %s", file)
		}
		if err := dataWatcher.Add(filepath.Join(*dataFolder, file+".json")); err != nil && !os.IsNotExist(err) {
			return errors.Wrapf(err, "file: %s", file)
		}
	}

	return errors.Trace(g.Wait())
}
