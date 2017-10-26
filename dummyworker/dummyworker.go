package dummyworker

import (
	"context"

	"time"

	"math/rand"

	"fmt"

	"github.com/dave/blast"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

func New() blast.Worker {
	return &Worker{}
}

type Worker struct {
	base string
}

func (w *Worker) Start(ctx context.Context, raw map[string]interface{}) error {

	var config workerConfig
	if err := mapstructure.Decode(raw, &config); err != nil {
		return errors.WithStack(err)
	}

	w.base = config.Base

	fmt.Printf("Dummy worker: Initialising with %s\n", config.Base)
	return nil
}

func (w *Worker) Send(ctx context.Context, raw map[string]interface{}) (map[string]interface{}, error) {

	var payload payloadConfig
	if err := mapstructure.Decode(raw, &payload); err != nil {
		return nil, errors.WithStack(err)
	}

	fmt.Printf("Dummy worker: Sending payload %s %s%s\n", payload.Method, w.base, payload.Path)

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Dummy worker - wait a random time
	duration := 1000 + int(r.Float64()*1000.0)
	select {
	case <-time.After(time.Millisecond * time.Duration(duration)):
	case <-ctx.Done():
	}

	// Dummy worker - return an error sometimes
	errorrand := r.Float64()
	if errorrand > 0.95 {
		return map[string]interface{}{"code": 500}, errors.New("Error 500")
	} else if errorrand > 0.7 {
		return map[string]interface{}{"code": 404}, errors.New("Error 404")
	} else {
		return map[string]interface{}{"code": 200}, nil
	}
}

type workerConfig struct {
	Base string `mapstructure:"base"`
}

type payloadConfig struct {
	Method string `mapstructure:"method"`
	Path   string `mapstructure:"path"`
}
