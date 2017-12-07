// Package gcsworker implements an http worker with automatic Google Cloud authentication.
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

// New returns a new gcs worker
func New() blaster.Worker {
	return &Worker{}
}

// Worker is the worker type
type Worker struct {
	client *http.Client
}

// Start satisfies the blaster.Starter interface
func (w *Worker) Start(ctx context.Context, payload map[string]interface{}) error {
	// notest
	src, err := google.DefaultTokenSource(ctx)
	if err != nil {
		return err
	}
	w.client = oauth2.NewClient(ctx, src)
	return nil
}

// Send satisfies the blaster.Worker interface
func (w *Worker) Send(ctx context.Context, raw map[string]interface{}) (map[string]interface{}, error) {

	var payload def
	if err := mapstructure.Decode(raw, &payload); err != nil {
		return map[string]interface{}{"status": "Error decoding payload"}, err
	}

	request, err := http.NewRequest(payload.Method, payload.URL, bytes.NewBufferString(payload.Body))
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
			// notest
			status = response.StatusCode
		case ok && ue.Err == context.DeadlineExceeded:
			status = "Timeout"
		case ok && ue.Err == context.Canceled:
			status = "Cancelled"
		case ok:
			status = ue.Err.Error()
		default:
			// notest
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
	// Method sets the http method e.g. `GET`, `POST` etc.
	Method string `mapstructure:"method"`
	// Url sets the full URL of the http request
	URL string `mapstructure:"url"`
	// Body sets the full http body
	Body string `mapstructure:"body"`
	// Headers sets the http headers
	Headers map[string]string `mapstructure:"headers"`
}
