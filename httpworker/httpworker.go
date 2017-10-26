package httpworker

import (
	"context"

	"net/http"

	"bytes"

	"github.com/dave/blast"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

func New() blast.Worker {
	return &Worker{}
}

type Worker struct{}

func (w *Worker) Send(ctx context.Context, raw map[string]interface{}) error {
	var payload def
	if err := mapstructure.Decode(&payload, raw); err != nil {
		return errors.WithStack(err)
	}
	request, err := http.NewRequest(payload.Method, payload.Url, bytes.NewBufferString(payload.Body))
	if err != nil {
		return errors.WithStack(err)
	}
	request = request.WithContext(ctx)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return errors.WithStack(err)
	}
	if response.StatusCode != 200 {
		return errors.New("Non 200 status code")
	}
	return nil
}

type def struct {
	Method string `mapstructure:"method"`
	Url    string `mapstructure:"url"`
	Body   string `mapstructure:"body"`
}
