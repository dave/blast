package blaster

import (
	"bytes"
	"context"
	"encoding/csv"
	"regexp"
	"sync"
	"testing"

	"fmt"

	"time"

	"io"

	"strings"

	"github.com/pkg/errors"
)

func TestSuccess(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	b := New(ctx, cancel)
	b.Rate = 0 // set rate to 0 so we can inject items synthetically
	b.itemFinishedChannel = make(chan struct{})

	workerLog := new(LoggingWorker)
	b.SetWorker(workerLog.NewSuccess)

	finished := make(chan struct{})
	go func() {
		must(t, b.start(ctx))
		close(finished)
	}()

	// synthetically call the main channel, which is what the ticker would do
	b.mainChannel <- 0
	<-b.itemFinishedChannel

	b.mainChannel <- 0
	<-b.itemFinishedChannel

	// start graceful exit process
	close(b.dataFinishedChannel)

	// wait for the start method to finish
	<-finished

	b.Exit()

	workerLog.mustLen(t, 2)
	workerLog.must(t, 0, map[string]string{"_success": "true"})
	workerLog.must(t, 1, map[string]string{"_success": "true"})

}

func TestFail(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	b := New(ctx, cancel)
	b.Rate = 0 // set rate to 0 so we can inject items synthetically
	b.itemFinishedChannel = make(chan struct{})

	worker := new(LoggingWorker)
	b.SetWorker(worker.NewFail)

	finished := make(chan struct{})
	go func() {
		must(t, b.start(ctx))
		close(finished)
	}()

	// synthetically call the main channel, which is what the ticker would do
	b.mainChannel <- 0
	<-b.itemFinishedChannel

	b.mainChannel <- 0
	<-b.itemFinishedChannel

	// start graceful exit process
	close(b.dataFinishedChannel)

	// wait for the start method to finish
	<-finished

	b.Exit()

	worker.mustLen(t, 2)
	worker.must(t, 0, map[string]string{"_success": "false"})
	worker.must(t, 1, map[string]string{"_success": "false"})

}

func TestHung(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	b := New(ctx, cancel)
	b.Rate = 0 // set rate to 0 so we can inject items synthetically
	b.SetTimeout(200 * time.Millisecond)
	b.itemFinishedChannel = make(chan struct{})

	worker := new(LoggingWorker)
	b.SetWorker(worker.NewHang(100))

	finished := make(chan struct{})
	go func() {
		must(t, b.start(ctx))
		close(finished)
	}()

	// synthetically call the main channel, which is what the ticker would do
	b.mainChannel <- 0
	<-b.itemFinishedChannel

	// start graceful exit process
	close(b.dataFinishedChannel)

	// wait for the start method to finish
	<-finished

	b.Exit()

	worker.mustLen(t, 1)
	worker.must(t, 0, map[string]string{"_hung": "true"})

}

func TestTimeout(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	b := New(ctx, cancel)
	b.Rate = 0 // set rate to 0 so we can inject items synthetically
	b.SetTimeout(5 * time.Millisecond)
	b.itemFinishedChannel = make(chan struct{})

	worker := new(LoggingWorker)
	b.SetWorker(worker.NewHang(500))

	finished := make(chan struct{})
	go func() {
		must(t, b.start(ctx))
		close(finished)
	}()

	// synthetically call the main channel, which is what the ticker would do
	b.mainChannel <- 0
	<-b.itemFinishedChannel

	// start graceful exit process
	close(b.dataFinishedChannel)

	// wait for the start method to finish
	<-finished

	b.Exit()

	worker.mustLen(t, 1)
	worker.must(t, 0, map[string]string{"_cancelled": "true"})

}

func TestCancel(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	b := New(ctx, cancel)
	b.Rate = 0 // set rate to 0 so we can inject items synthetically
	b.SetTimeout(500 * time.Millisecond)

	worker := new(LoggingWorker)
	b.SetWorker(worker.NewHang(400))

	finished := make(chan struct{})
	go func() {
		must(t, b.start(ctx))
		close(finished)
	}()

	// synthetically call the main channel, which is what the ticker would do
	b.mainChannel <- 0

	b.Exit()

	// wait for the start method to finish
	<-finished

	worker.mustLen(t, 1)
	worker.must(t, 0, map[string]string{"_cancelled": "true"})

}

func TestLog(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	b := New(ctx, cancel)
	b.Rate = 0 // set rate to 0 so we can inject items synthetically
	b.itemFinishedChannel = make(chan struct{})

	worker := new(LoggingWorker)
	b.SetWorker(worker.NewSuccess)

	log := &LoggingWriter{buf: new(bytes.Buffer)}
	b.SetLog(log)

	out := new(ThreadSafeBuffer)
	b.SetOutput(out)

	b.Headers = []string{"head"}
	b.SetData(strings.NewReader("a\nb"))

	finished := make(chan struct{})
	go func() {
		must(t, b.start(ctx))
		close(finished)
	}()

	// synthetically call the main channel, which is what the ticker would do
	b.mainChannel <- 0
	<-b.itemFinishedChannel

	b.mainChannel <- 0
	<-b.itemFinishedChannel

	// another tick and the data will reach EOF, and gracefully exit
	b.mainChannel <- 0

	// wait for the start method to finish
	<-finished

	b.Exit()

	worker.mustLen(t, 2)
	worker.must(t, 0, map[string]string{"_success": "true"})
	worker.must(t, 1, map[string]string{"_success": "true"})

	log.mustLen(t, 2)
	log.must(t, 0, []string{"45583464115695f2|e60a15c85c691ab8", "true"})
	log.must(t, 1, []string{"6258a554f446f0a7|4111d6d36a631a68", "true"})

	mustMatch(t, out, 1, `\n\[success\]\s*\n---------\s*\nCount\:\s+2\s`)

}

func TestResume(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	b := New(ctx, cancel)
	b.Resume = true
	b.Rate = 0 // set rate to 0 so we can inject items synthetically
	b.itemFinishedChannel = make(chan struct{})

	worker := new(LoggingWorker)
	b.SetWorker(worker.NewSuccess)

	log := &LoggingWriter{buf: new(bytes.Buffer)}
	b.SetLog(log)

	out := new(ThreadSafeBuffer)
	b.SetOutput(out)

	b.Headers = []string{"head"}
	b.SetData(strings.NewReader("a\nb\nc"))

	// In this log fragment, second item failed on first run so will retry:
	must(t, b.LoadLogs(bytes.NewBufferString("hash,result\n45583464115695f2|e60a15c85c691ab8,true\n6258a554f446f0a7|4111d6d36a631a68,false")))

	finished := make(chan struct{})
	go func() {
		must(t, b.start(ctx))
		close(finished)
	}()

	// this will skip the first item and complete the second item
	b.mainChannel <- 0
	<-b.itemFinishedChannel

	// this complete the third item
	b.mainChannel <- 0
	<-b.itemFinishedChannel

	// another tick and the data will reach EOF, and gracefully exit
	b.mainChannel <- 0

	// wait for the start method to finish
	<-finished

	b.Exit()

	mustMatch(t, out, 1, `\n\[success\]\s*\n---------\s*\nCount\:\s+2\s`)
	mustMatch(t, out, 1, `\nSkipped\:\s+1 from previous runs`)

	log.mustLen(t, 2)
	log.must(t, 0, []string{"6258a554f446f0a7|4111d6d36a631a68", "true"})
	log.must(t, 1, []string{"d0e4144aef1f25ee|f44a70605aeac064", "true"})

}

func TestPayloadVariants(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	b := New(ctx, cancel)
	b.Rate = 0 // set rate to 0 so we can inject items synthetically
	b.itemFinishedChannel = make(chan struct{})

	must(t, b.SetPayloadTemplate(map[string]interface{}{
		"v1": "{{.head}}-{{.p1}}",
		"v2": "{{.p2}}",
	}))

	b.PayloadVariants = []map[string]string{
		{"p1": "p1v1", "p2": "p2v1"},
		{"p1": "p1v2", "p2": "p2v2"},
	}

	b.Headers = []string{"head"}
	b.SetData(strings.NewReader("a\nb"))

	worker := new(LoggingWorker)
	b.SetWorker(worker.NewSuccess)

	finished := make(chan struct{})
	go func() {
		must(t, b.start(ctx))
		close(finished)
	}()

	// each signal on the main channel will complete all the payload variants of an item, but
	// itemFinishedChannel needs to be read once for each variant
	b.mainChannel <- 0
	<-b.itemFinishedChannel
	<-b.itemFinishedChannel

	b.mainChannel <- 0
	<-b.itemFinishedChannel
	<-b.itemFinishedChannel

	// another tick and the data will reach EOF, and gracefully exit
	b.mainChannel <- 0

	// wait for the start method to finish
	<-finished

	b.Exit()

	worker.mustLen(t, 4)
	worker.must(t, -1, map[string]string{"_success": "true", "v1": "a-p1v1", "v2": "p2v1"})
	worker.must(t, -1, map[string]string{"_success": "true", "v1": "a-p1v2", "v2": "p2v2"})
	worker.must(t, -1, map[string]string{"_success": "true", "v1": "b-p1v1", "v2": "p2v1"})
	worker.must(t, -1, map[string]string{"_success": "true", "v1": "b-p1v2", "v2": "p2v2"})

}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func mustMatch(t *testing.T, buf *ThreadSafeBuffer, num int, pattern string) {
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
	buf *bytes.Buffer
}

func (l *LoggingWriter) Write(p []byte) (n int, err error) {
	return l.buf.Write(p)
}

func (l *LoggingWriter) Debug() {
	reader := csv.NewReader(bytes.NewBuffer(l.buf.Bytes()))
	for {
		r, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				fmt.Println(err.Error())
			}
		}
		fmt.Println(r)
	}
}

func (l *LoggingWriter) mustLen(t *testing.T, expected int) {
	t.Helper()
	var log [][]string
	reader := csv.NewReader(bytes.NewBuffer(l.buf.Bytes()))
	for {
		r, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				fmt.Println(err.Error())
			}
		}
		log = append(log, r)
	}
	if expected != len(log) {
		t.Fatalf("Log is not length %d:\n%v", expected, log)
	}
}

func (l *LoggingWriter) must(t *testing.T, index int, expected []string) {
	t.Helper()
	var log [][]string
	reader := csv.NewReader(bytes.NewBuffer(l.buf.Bytes()))
	for {
		r, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				fmt.Println(err.Error())
			}
		}
		log = append(log, r)
	}

	record := log[index]
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
	t.Fatalf("Record %s not found at index %d in output log %s", expected, index, log)
}

type LoggingWorker struct {
	Log []map[string]string
	m   sync.Mutex
}

func (l *LoggingWorker) Debug() {
	for _, r := range l.Log {
		fmt.Println(r)
	}
}

func (l *LoggingWorker) mustLen(t *testing.T, length int) {
	if len(l.Log) != length {
		t.Fatalf("Worker log is not length %d:\n%v", length, l.Log)
	}
}

func (l *LoggingWorker) must(t *testing.T, index int, expected map[string]string) {
	t.Helper()
	compare := func(record map[string]string) bool {
		if len(record) == len(expected) {
			found := true
			for k, value := range record {
				if value != expected[k] {
					found = false
					break
				}
			}
			if found {
				return true
			}
		}
		return false
	}
	if index > -1 {
		if compare(l.Log[index]) {
			return
		}
		t.Fatalf("Record %s not found at index %d in worker log %s", expected, index, l.Log)
	} else {
		for _, record := range l.Log {
			if compare(record) {
				return
			}
		}
		t.Fatalf("Record %s not found in worker log %s", expected, l.Log)
	}
}

func (l *LoggingWorker) Append(message map[string]string) {
	l.m.Lock()
	defer l.m.Unlock()
	l.Log = append(l.Log, message)
}

type loggingWorker struct {
	Result bool
	Hang   int
	Log    *LoggingWorker
}

func (l *LoggingWorker) NewSuccess() Worker {
	return &loggingWorker{Log: l, Result: true}
}

func (l *LoggingWorker) NewFail() Worker {
	return &loggingWorker{Log: l, Result: false}
}

func (l *LoggingWorker) NewHang(duration int) func() Worker {
	return func() Worker { return &loggingWorker{Log: l, Result: true, Hang: duration} }
}

func (l *loggingWorker) Send(ctx context.Context, in map[string]interface{}) (map[string]interface{}, error) {
	log := map[string]string{}
	if l.Hang > 0 {
		select {
		case <-time.After(time.Duration(l.Hang) * time.Millisecond):
			log["_hung"] = "true"
		case <-ctx.Done():
			log["_cancelled"] = "true"
		}
	} else if l.Result {
		log["_success"] = "true"
	} else {
		log["_success"] = "false"
	}
	for k, v := range in {
		log[k] = fmt.Sprint(v)
	}
	l.Log.Append(log)
	if l.Result {
		return map[string]interface{}{"status": "[success]"}, nil
	}
	return map[string]interface{}{"status": "[fail]"}, errors.New("fail")
}

type DummyCloser struct{}

func (DummyCloser) Close() error { return nil }

type ThreadSafeBuffer struct {
	b bytes.Buffer
	m sync.Mutex
}

func (b *ThreadSafeBuffer) Read(p []byte) (n int, err error) {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.Read(p)
}
func (b *ThreadSafeBuffer) Write(p []byte) (n int, err error) {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.Write(p)
}
func (b *ThreadSafeBuffer) String() string {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.String()
}
