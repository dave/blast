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

func New() blaster.Worker {
	return &Worker{}
}

type Worker struct {
	base     string
	print    bool
	rand     *rand.Rand
	min, max int
}

func (w *Worker) Start(ctx context.Context, raw map[string]interface{}) error {

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

func (w *Worker) Send(ctx context.Context, raw map[string]interface{}) (map[string]interface{}, error) {

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
	Base  string `mapstructure:"base"`
	Print bool   `mapstructure:"print"`
	Min   int    `mapstructure:"min"`
	Max   int    `mapstructure:"max"`
}

type payloadConfig struct {
	Method string `mapstructure:"method"`
	Path   string `mapstructure:"path"`
}
