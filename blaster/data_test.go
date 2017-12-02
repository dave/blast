package blaster

import (
	"context"
	"io"
	"testing"
)

func TestOpenGcs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	b := New(ctx, cancel)

	opener := &loggingOpener{}
	b.gcsOpener = opener

	must(t, b.openData(ctx, "gs://a/b", false))

	if opener.bucket != "a" || opener.handle != "b" {
		t.Fatalf("Got bucket=%s, handle=%s", opener.bucket, opener.handle)
	}
}

type loggingOpener struct {
	bucket string
	handle string
}

func (l *loggingOpener) open(ctx context.Context, bucket, handle string) (io.Reader, error) {
	l.bucket = bucket
	l.handle = handle
	return nil, nil
}
