package blast

import (
	"bytes"
	"context"
	"encoding/csv"
	"regexp"
	"strings"
	"sync"
	"testing"

	"fmt"

	"time"

	"github.com/pkg/errors"
)

func defaultOptions(
	ctx context.Context,
	cancel context.CancelFunc,
	in string,
	workerType string,
	logWriter LogWriteFlusher,
	workerLog *LoggingWorkerLog,
) (*Blaster, *bytes.Buffer) {
	b := New(ctx, cancel)
	b.RegisterWorkerType("success", workerLog.NewSuccess)
	b.RegisterWorkerType("fail", workerLog.NewFail)
	b.RegisterWorkerType("hang", workerLog.NewHang)
	b.config = &configDef{}
	b.config.Workers = 1
	b.config.WorkerType = workerType
	b.config.PayloadVariants = []map[string]string{{}}
	b.rate = 100
	b.dataHeaders = []string{"head"}
	b.dataReader = csv.NewReader(strings.NewReader(in))
	b.logWriter = logWriter
	outbuf := new(bytes.Buffer)
	b.out = outbuf
	b.rateInputReader = strings.NewReader("")
	return b, outbuf
}

func TestNew(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	workerLog := new(LoggingWorkerLog)
	outLog := new(LoggingWriter)
	outLog.Write([]string{"hash", "result"})

	b, outbuf := defaultOptions(
		ctx,
		cancel,
		"a\nb\nc",
		"success",
		outLog,
		workerLog,
	)

	must(t, b.start(ctx))

	mustMatch(t, outbuf, 1, `\nSuccess\:\s+0\n`)
	mustMatch(t, outbuf, 1, `\nSuccess\:\s+3\n`)

	outLog.must(t, 1, []string{"3763b9c0e1b2307c|c1377b027e806557", "true"})
	outLog.must(t, 2, []string{"db7a669e37739bf|b4a36ba02942a475", "true"})
	outLog.must(t, 3, []string{"deb69562b047222|3cec67420f8a6588", "true"})

	b1, outbuf1 := defaultOptions(
		ctx,
		cancel,
		"a\nb\nc\nd",
		"success",
		outLog,
		workerLog,
	)

	must(t, b1.loadPreviousLogsFromReader(outLog.reader()))
	must(t, b1.start(ctx))

	mustMatch(t, outbuf1, 1, `\nSuccess\:\s+0\n`)
	mustMatch(t, outbuf1, 1, `\nSuccess\:\s+1\n`)
	mustMatch(t, outbuf1, 1, `\nSkipped\:\s+3 \(from previous run\)\n`)
	outLog.must(t, 4, []string{"73d81ec7b7251e65|fab9096e8c84809f", "true"})

	b2, outbuf2 := defaultOptions(
		ctx,
		cancel,
		"e",
		"fail",
		outLog,
		workerLog,
	)

	must(t, b2.start(ctx))
	mustMatch(t, outbuf2, 1, `\nFailed\:\s+1\n`)
	outLog.must(t, 5, []string{"21e8bfb8d271d28d|c546009ae399ca09", "false"})

	//for i, v := range outLog.Log {
	//	fmt.Println(i, v)
	//}

}

func TestPayloadVariants(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	workerLog := new(LoggingWorkerLog)
	outLog := new(LoggingWriter)
	outLog.Write([]string{"hash", "result"})

	b, _ := defaultOptions(
		ctx,
		cancel,
		"a\nb",
		"success",
		outLog,
		workerLog,
	)
	b.config.PayloadTemplate = map[string]interface{}{
		"v1": "{{head}}-{{p1}}",
		"v2": "{{p2}}",
	}
	b.config.PayloadVariants = []map[string]string{
		{"p1": "p1v1", "p2": "p2v1"},
		{"p1": "p1v2", "p2": "p2v2"},
	}
	must(t, b.start(ctx))

	workerLog.must(t, 0, map[string]string{"v1": "a-p1v1", "v2": "p2v1"})
	workerLog.must(t, 1, map[string]string{"v1": "a-p1v2", "v2": "p2v2"})
	workerLog.must(t, 2, map[string]string{"v1": "b-p1v1", "v2": "p2v1"})
	workerLog.must(t, 3, map[string]string{"v1": "b-p1v2", "v2": "p2v2"})

}

func TestCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	workerLog := new(LoggingWorkerLog)
	outLog := new(LoggingWriter)
	outLog.Write([]string{"hash", "result"})

	b, _ := defaultOptions(
		ctx,
		cancel,
		"a\nb\nc",
		"hang",
		outLog,
		workerLog,
	)
	b.rate = 20
	finished := make(chan struct{})
	go func() {
		must(t, b.start(ctx))
		close(finished)
	}()
	<-time.After(time.Millisecond * 70) // rate is 20/sec, so first will fire at 50ms
	b.cancel()
	select {
	case <-finished:
	case <-time.After(time.Millisecond * 200):
		t.Fatal("timeout")
	}

	workerLog.mustLen(t, 1)
	workerLog.must(t, 0, map[string]string{"_cancelled": "true"})

	//for i, v := range workerLog.Log {
	//	fmt.Println(i, v)
	//}

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

type LoggingWriter struct {
	Log [][]string
}

func (l *LoggingWriter) Write(record []string) error {
	l.Log = append(l.Log, record)
	return nil
}
func (l *LoggingWriter) Flush() {}

func (l *LoggingWriter) reader() *bytes.Buffer {
	buf := &bytes.Buffer{}
	w := csv.NewWriter(buf)
	for _, v := range l.Log {
		w.Write(v)
	}
	w.Flush()
	return buf
}

func (l *LoggingWriter) must(t *testing.T, index int, expected []string) {
	t.Helper()
	record := l.Log[index]
	if len(record) == len(expected) {
		found := true
		for i, value := range record {
			if value != expected[i] {
				found = false
				break
			}
		}
		if found {
			return
		}
	}
	t.Fatalf("Record %s not found at index %d in output log %s", expected, index, l.Log)
}

type LoggingWorkerLog struct {
	Log []map[string]string
	m   sync.Mutex
}

func (l *LoggingWorkerLog) mustLen(t *testing.T, length int) {
	if len(l.Log) != length {
		t.Fatalf("Worker log is not length %d:\n%v", length, l.Log)
	}
}

func (l *LoggingWorkerLog) must(t *testing.T, index int, expected map[string]string) {
	t.Helper()
	record := l.Log[index]
	if len(record) == len(expected) {
		found := true
		for k, value := range record {
			if value != expected[k] {
				found = false
				break
			}
		}
		if found {
			return
		}
	}
	t.Fatalf("Record %s not found at index %d in worker log %s", expected, index, l.Log)
}

func (l *LoggingWorkerLog) Append(message map[string]string) {
	l.m.Lock()
	defer l.m.Unlock()
	l.Log = append(l.Log, message)
}

type LoggingWorker struct {
	Result bool
	Hang   bool
	Log    *LoggingWorkerLog
}

func (l *LoggingWorkerLog) NewSuccess() Worker {
	return &LoggingWorker{Log: l, Result: true}
}

func (l *LoggingWorkerLog) NewFail() Worker {
	return &LoggingWorker{Log: l, Result: false}
}

func (l *LoggingWorkerLog) NewHang() Worker {
	return &LoggingWorker{Log: l, Result: true, Hang: true}
}

func (l *LoggingWorker) Send(ctx context.Context, in map[string]interface{}) error {
	log := map[string]string{}
	if l.Hang {
		select {
		case <-time.After(time.Second):
			log["_hung"] = "true"
		case <-ctx.Done():
			log["_cancelled"] = "true"
		}
	}
	for k, v := range in {
		log[k] = fmt.Sprint(v)
	}
	l.Log.Append(log)
	if l.Result {
		return nil
	}
	return errors.New("fail")
}

type DummyCloser struct{}

func (DummyCloser) Close() error { return nil }
