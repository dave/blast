package blast

import (
	"encoding/csv"
	"os"

	"context"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/pkg/errors"
)

func (b *Blaster) openDataFile(ctx context.Context) error {
	var err error
	if strings.HasPrefix(b.config.Data, "gs://") {
		name := strings.TrimPrefix(b.config.Data, "gs://")
		bucket := name[:strings.Index(name, "/")]
		handle := name[strings.Index(name, "/")+1:]
		client, err := storage.NewClient(ctx)
		if err != nil {
			return errors.WithStack(err)
		}
		b.dataReadCloser, err = client.Bucket(bucket).Object(handle).NewReader(ctx)
		if err != nil {
			return errors.WithStack(err)
		}
	} else {
		b.dataReadCloser, err = os.Open(b.config.Data)
		if err != nil {
			return errors.WithStack(err)
		}
	}
	b.dataReader = csv.NewReader(b.dataReadCloser)
	b.dataHeaders, err = b.dataReader.Read()
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (b *Blaster) closeDataFile() {
	_ = b.dataReadCloser.Close() // ignore error
}
