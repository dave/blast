package blast

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"regexp"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	newBlast := func(in string, workerType string, logbuf *bytes.Buffer) (*Blaster, *bytes.Buffer) {
		b := New(ctx, cancel)
		b.RegisterWorkerType("success", NewImmediateSuccessWorker)
		b.RegisterWorkerType("fail", NewImmediateFailWorker)
		b.config = &configDef{}
		b.config.Workers = 1
		b.config.WorkerType = workerType
		b.config.PayloadVariants = []map[string]string{{}}
		b.rate = 100
		b.dataHeaders = []string{"head"}
		b.dataReader = csv.NewReader(strings.NewReader(in))
		b.logWriter = csv.NewWriter(logbuf)
		outbuf := new(bytes.Buffer)
		b.out = outbuf
		return b, outbuf
	}

	logbuf := new(bytes.Buffer)
	logbuf.WriteString("hash,result\n")

	b, outbuf := newBlast("a\nb\nc", "success", logbuf)
	must(t, b.start(ctx))
	b.logWriter.Flush()

	mustMatch(t, outbuf, 1, `\nSuccess\:\s+0\n`)
	mustMatch(t, outbuf, 1, `\nSuccess\:\s+3\n`)
	mustMatch(t, logbuf, 1, `3763b9c0e1b2307c\|c1377b027e806557\,true\n`)
	mustMatch(t, logbuf, 1, `db7a669e37739bf\|b4a36ba02942a475\,true\n`)
	mustMatch(t, logbuf, 1, `deb69562b047222\|3cec67420f8a6588\,true\n`)

	b1, outbuf1 := newBlast("a\nb\nc\nd", "success", logbuf)

	must(t, b1.loadPreviousLogsFromReader(bytes.NewBuffer(logbuf.Bytes())))
	must(t, b1.start(ctx))
	b1.logWriter.Flush()

	mustMatch(t, outbuf1, 1, `\nSuccess\:\s+0\n`)
	mustMatch(t, outbuf1, 1, `\nSuccess\:\s+1\n`)
	mustMatch(t, outbuf1, 1, `\nSkipped\:\s+3 \(from previous run\)\n`)
	mustMatch(t, logbuf, 1, `73d81ec7b7251e65\|fab9096e8c84809f\,true\n`)

	b2, outbuf2 := newBlast("e", "fail", logbuf)
	must(t, b2.start(ctx))
	b2.logWriter.Flush()
	mustMatch(t, outbuf2, 1, `\nFailed\:\s+1\n`)

}

func must(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

func mustMatch(t *testing.T, buf *bytes.Buffer, num int, pattern string) {
	t.Helper()
	matches := regexp.MustCompile(pattern).FindAllString(buf.String(), -1)
	if len(matches) != num {
		t.Fatalf("Matches in output (%d) not expected (%d) for pattern %s:\n%s",
			len(matches),
			num,
			pattern,
			buf.String(),
		)
	}
}

type DummyCloser struct{}

func (DummyCloser) Close() error { return nil }

func NewImmediateSuccessWorker() Worker {
	return &ImmediateSuccessWorker{}
}

type ImmediateSuccessWorker struct{}

func (*ImmediateSuccessWorker) Send(context.Context, map[string]interface{}) error {
	return nil
}

func NewImmediateFailWorker() Worker {
	return &ImmediateFailWorker{}
}

type ImmediateFailWorker struct{}

func (*ImmediateFailWorker) Send(context.Context, map[string]interface{}) error {
	return errors.New("fail")
}
