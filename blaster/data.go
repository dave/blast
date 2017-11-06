package blaster

import (
	"os"

	"context"
	"strings"

	"io"

	"encoding/csv"

	"cloud.google.com/go/storage"
	"github.com/pkg/errors"
)

func (b *Blaster) ReadHeaders() error {
	h, err := b.dataReader.Read()
	if err != nil {
		return errors.WithStack(err)
	}
	b.Headers = h
	return nil
}

func (b *Blaster) SetData(r io.Reader) {
	if r == nil {
		b.dataReader = nil
		b.dataCloser = nil
		return
	}
	b.dataReader = csv.NewReader(r)
	if c, ok := r.(io.Closer); ok {
		b.dataCloser = c
	} else {
		b.dataCloser = nil
	}
}

func (b *Blaster) openData(ctx context.Context, value string, headers bool) error {
	if value == "" {
		return nil
	}
	var r io.Reader
	if strings.Contains(value, "\n") {
		r = strings.NewReader(value)
	} else if strings.HasPrefix(value, "gs://") {
		name := strings.TrimPrefix(value, "gs://")
		bucket := name[:strings.Index(name, "/")]
		handle := name[strings.Index(name, "/")+1:]
		client, err := storage.NewClient(ctx)
		if err != nil {
			return errors.WithStack(err)
		}
		gr, err := client.Bucket(bucket).Object(handle).NewReader(ctx)
		if err != nil {
			return errors.WithStack(err)
		}
		r = gr
	} else {
		fr, err := os.Open(value)
		if err != nil {
			return errors.WithStack(err)
		}
		r = fr
	}

	b.SetData(r)

	if headers {
		if err := b.ReadHeaders(); err != nil {
			return err
		}
	}
	return nil

}
