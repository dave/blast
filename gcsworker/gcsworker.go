package gcsworker

import (
	"context"

	"net/http"

	"bytes"

	"errors"

	"net/url"

	"github.com/dave/blast/blaster"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func New() blaster.Worker {
	return &Worker{}
}

type Worker struct {
	client *http.Client
}

func (w *Worker) Start(ctx context.Context, payload map[string]interface{}) error {
	src, err := google.DefaultTokenSource(ctx)
	if err != nil {
		return err
	}
	w.client = oauth2.NewClient(ctx, src)
	return nil
}

func (w *Worker) Send(ctx context.Context, raw map[string]interface{}) (map[string]interface{}, error) {

	var payload def
	if err := mapstructure.Decode(raw, &payload); err != nil {
		return map[string]interface{}{"status": "Error decoding payload"}, err
	}

	request, err := http.NewRequest(payload.Method, payload.Url, bytes.NewBufferString(payload.Body))
	if err != nil {
		return map[string]interface{}{"status": "Error creating request"}, err
	}

	request = request.WithContext(ctx)

	for k, v := range payload.Headers {
		request.Header.Add(k, v)
	}

	response, err := w.client.Do(request)
	if err != nil {
		var status interface{}
		ue, ok := err.(*url.Error)
		switch {
		case response != nil:
			status = response.StatusCode
		case ok && ue.Err == context.DeadlineExceeded:
			status = "Timeout"
		case ok && ue.Err == context.Canceled:
			status = "Cancelled"
		case ok:
			status = ue.Err.Error()
		default:
			status = err.Error()
		}
		return map[string]interface{}{"status": status}, err
	}
	if response.StatusCode != 200 {
		return map[string]interface{}{"status": response.StatusCode}, errors.New("non 200 status")
	}
	return map[string]interface{}{"status": 200}, nil
}

type def struct {
	Method  string            `mapstructure:"method"`
	Url     string            `mapstructure:"url"`
	Body    string            `mapstructure:"body"`
	Headers map[string]string `mapstructure:"headers"`
}
