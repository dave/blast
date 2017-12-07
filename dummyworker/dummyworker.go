// Package dummyworker implements a worker for testing and examples.
package dummyworker

import (
	"context"

	"time"

	"math/rand"

	"fmt"

	"errors"

	"github.com/dave/blast/blaster"
	"github.com/mitchellh/mapstructure"
)

// New returns a new dummy worker
func New() blaster.Worker {
	// notest
	return &Worker{}
}

// Worker is the worker type
type Worker struct {
	base     string
	print    bool
	rand     *rand.Rand
	min, max int
}

// Start satisfies the blaster.Starter interface
func (w *Worker) Start(ctx context.Context, raw map[string]interface{}) error {

	// notest

	var config workerConfig
	if err := mapstructure.Decode(raw, &config); err != nil {
		return err
	}

	w.base = config.Base
	w.print = config.Print
	w.min = config.Min
	w.max = config.Max
	w.rand = rand.New(rand.NewSource(time.Now().UnixNano()))

	if w.print {
		fmt.Printf("Dummy worker: Initialising with %s\n", config.Base)
	}
	return nil
}

// Send satisfies the blaster.Worker interface
func (w *Worker) Send(ctx context.Context, raw map[string]interface{}) (map[string]interface{}, error) {

	// notest

	var payload payloadConfig
	if err := mapstructure.Decode(raw, &payload); err != nil {
		return map[string]interface{}{"status": "Error decoding payload"}, err
	}

	if w.print {
		fmt.Printf("Dummy worker: Sending payload %s %s%s\n", payload.Method, w.base, payload.Path)
	}

	// Dummy worker - wait a random time
	duration := w.min + int(w.rand.Float64()*float64(w.max-w.min))

	select {
	case <-time.After(time.Millisecond * time.Duration(duration)):
		// Dummy worker - success!
	case <-ctx.Done():
		// Dummy worker - interrupted by context
		err := ctx.Err()
		var status string
		switch err {
		case nil:
			status = "Unknown"
			err = errors.New("Context done")
		case context.DeadlineExceeded:
			status = "Timeout"
		case context.Canceled:
			status = "Cancelled"
		default:
			status = err.Error()
		}
		return map[string]interface{}{"status": status}, err
	}

	// Dummy worker - return an error sometimes
	errorrand := w.rand.Float64()
	if errorrand > 0.99 {
		return map[string]interface{}{"status": 500}, errors.New("Error 500")
	} else if errorrand > 0.96 {
		return map[string]interface{}{"status": 404}, errors.New("Error 404")
	} else {
		return map[string]interface{}{"status": 200}, nil
	}
}

type workerConfig struct {
	// Base sets the base of the http request e.g. `http://foo.com`
	Base string `mapstructure:"base"`
	// Print causes the worker to print debug messages
	Print bool `mapstructure:"print"`
	// Min is the minimum bound of the random wait, in ms
	Min int `mapstructure:"min"`
	// Max is the maximum bound of the random wait, in ms
	Max int `mapstructure:"max"`
}

type payloadConfig struct {
	// Method sets the http method e.g. `GET`, `POST` etc.
	Method string `mapstructure:"method"`
	// Path sets the path of the http request e.g. `/foo`
	Path string `mapstructure:"path"`
}
