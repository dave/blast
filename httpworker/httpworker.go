package httpworker

import (
	"context"

	"net/http"

	"bytes"

	"github.com/dave/blast"
	"github.com/pkg/errors"
)

func New() blast.Worker {
	return &Worker{}
}

type Worker struct{}

func (w *Worker) Send(ctx context.Context, payloadRaw map[string]interface{}) error {
	payload := parsePayload(payloadRaw)
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

func parsePayload(in map[string]interface{}) payloadDef {
	p := payloadDef{}
	if v, ok := in["method"]; ok {
		p.Method = v.(string)
	}
	if v, ok := in["url"]; ok {
		p.Url = v.(string)
	}
	if v, ok := in["body"]; ok {
		p.Body = v.(string)
	}
	return p
}

type payloadDef struct {
	Method string
	Url    string
	Body   string
}
