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

	"reflect"

	"io/ioutil"

	"github.com/leemcloughlin/gofarmhash"
	"github.com/pkg/errors"
)

func TestPrintStatus(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	b := New(ctx, cancel)
	b.Rate = 0 // set rate to 0 so we can inject items synthetically
	b.itemFinishedChannel = make(chan struct{})

	worker := new(LoggingWorker)
	b.SetWorker(worker.NewSuccess)

	input := &bytes.Buffer{}
	b.SetInput(input)

	out := new(ThreadSafeBuffer)
	b.SetOutput(out)

	b.Headers = []string{"head"}
	b.SetData(strings.NewReader("a"))

	finished := make(chan error, 1)
	go func() {
		finished <- b.start(ctx)
	}()

	// synthetically call the main channel, which is what the ticker would do
	b.mainChannel <- 0
	<-b.itemFinishedChannel

	b.printStatus(false)

	// another tick and the data will reach EOF, and gracefully exit
	b.mainChannel <- 0

	// wait for the start method to finish
	must(t, <-finished)

	b.Exit()

	// metrics and rate prompt will print twice
	mustMatch(t, out, 2, `Metrics\n=======\n`)
	mustMatch(t, out, 2, `Current rate is 0 requests / second. Enter a new rate or press enter to view status.\n\nRate?`)

}

func TestOpenLogNotExist(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	b := New(ctx, cancel)

	// file doesn't exist
	must(t, b.openAndLoadLogs("./cvxyoicvyuohwerlmbxviuhsdiouh"))

	// exists but zero length
	f, _ := ioutil.TempFile("", "")
	must(t, b.openAndLoadLogs(f.Name()))
}

func TestError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	b := New(ctx, cancel)
	b.Rate = 0 // set rate to 0 so we can inject items synthetically
	b.itemFinishedChannel = make(chan struct{})

	out := new(ThreadSafeBuffer)
	b.SetOutput(out)

	worker := new(LoggingWorker)
	b.SetWorker(worker.NewSuccess)

	finished := make(chan error, 1)
	go func() {
		finished <- b.start(ctx)
	}()

	// synthetically call the main channel, which is what the ticker would do
	b.mainChannel <- 0
	<-b.itemFinishedChannel

	b.error(errors.New("a"))
	b.error(errors.New("b"))

	// wait for the start method to finish
	err := <-finished
	if err == nil || err.Error() != "a" {
		t.Fatalf("Unexpected error: %s", err)
	}

	mustMatch(t, out, 1, "Fatal error: a")
	mustMatch(t, out, 1, "1 errors were ignored because we were already exiting with an error")

}

func TestInitialiseLog(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	b := New(ctx, cancel)
	must(t, b.initialiseLog(""))

	content := "hash,result,a,b\n1|2,false,3,4\n5|6,true,7,8"

	f, _ := ioutil.TempFile("", "")
	f.WriteString(content)
	f.Close()

	b = New(ctx, cancel)
	b.Resume = false
	must(t, b.initialiseLog(f.Name()))
	if len(b.skip) != 0 {
		t.Fatal("Should be zero skips with resume = false")
	}

	b.Exit()

	// log file should now be empty
	after, _ := ioutil.ReadFile(f.Name())
	if string(after) != "hash,result\n" {
		t.Fatal("Not expected, got:", string(after))
	}

	f, _ = ioutil.TempFile("", "")
	f.WriteString(content)
	f.Close()

	b = New(ctx, cancel)
	b.Resume = true
	must(t, b.initialiseLog(f.Name()))
	if !reflect.DeepEqual(b.skip, map[farmhash.Uint128]struct{}{farmhash.Uint128{5, 6}: {}}) {
		t.Fatal("Enexpected contents in skip:", b.skip)
	}

	// log file should now be appended with a \n
	after, _ = ioutil.ReadFile(f.Name())
	if string(after) != content+"\n" {
		t.Fatal("Not expected, got:", string(after))
	}

}

func TestLoadEmptyLogs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	b := New(ctx, cancel)
	must(t, b.LoadLogs(&bytes.Buffer{}))
}

func TestWriteHeaders(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	b := New(ctx, cancel)
	log := NewLoggingReadWriteCloser("")
	b.SetLog(log)
	b.LogData = []string{"a", "b"}
	b.LogOutput = []string{"c", "d"}
	must(t, b.WriteLogHeaders())
	b.Exit()
	log.mustWrite(t)
	log.mustClose(t)
	if log.Buf.String() != "hash,result,a,b,c,d\n" {
		t.Fatalf("Log headers not correct. Got: %s", log.Buf.String())
	}
}

func TestOpenData(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	b := New(ctx, cancel)

	must(t, b.openData(ctx, "", false))

	must(t, b.openData(ctx, "a,b\n1,2", false))
	if len(b.Headers) != 0 {
		t.Fatal("Incorrect headers, got:", b.Headers)
	}

	b = New(ctx, cancel)
	must(t, b.openData(ctx, "a,b\n1,2", true))
	if !reflect.DeepEqual(b.Headers, []string{"a", "b"}) {
		t.Fatal("Incorrect headers, got:", b.Headers)
	}
	r, err := b.dataReader.Read()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r, []string{"1", "2"}) {
		t.Fatal("Incorrect data, got:", r)
	}

	f, _ := ioutil.TempFile("", "")
	f.WriteString("a,b,c\n1,2,3")
	f.Close()

	b = New(ctx, cancel)
	must(t, b.openData(ctx, f.Name(), true))
	if !reflect.DeepEqual(b.Headers, []string{"a", "b", "c"}) {
		t.Fatal("Incorrect headers, got:", b.Headers)
	}
	r, err = b.dataReader.Read()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r, []string{"1", "2", "3"}) {
		t.Fatal("Incorrect data, got:", r)
	}
}

func TestReadHeaders(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	b := New(ctx, cancel)
	data := NewLoggingReadWriteCloser("a,b,c\n1,2,3\n4,5,6")
	b.SetData(data)
	must(t, b.ReadHeaders())
	if !reflect.DeepEqual(b.Headers, []string{"a", "b", "c"}) {
		t.Fatal("Incorrect headers, got:", b.Headers)
	}
}

func TestExit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	b := New(ctx, cancel)
	b.Rate = 0 // set rate to 0 so we can inject items synthetically
	b.itemFinishedChannel = make(chan struct{})

	worker := new(LoggingWorker)
	b.SetWorker(worker.NewSuccess)

	data := NewLoggingReadWriteCloser("a")
	log := NewLoggingReadWriteCloser("")
	output := NewLoggingReadWriteCloser("")

	b.SetData(data)
	b.SetLog(log)
	b.SetOutput(output)

	finished := make(chan error, 1)
	go func() {
		finished <- b.start(ctx)
	}()

	// synthetically call the main channel, which is what the ticker would do
	b.mainChannel <- 0
	<-b.itemFinishedChannel

	// start graceful exit process
	close(b.dataFinishedChannel)

	// wait for the start method to finish
	must(t, <-finished)

	b.Exit()

	data.mustRead(t)
	log.mustWrite(t)
	output.mustWrite(t)

	data.mustClose(t)
	log.mustClose(t)
	output.mustClose(t)

}

func TestSetNil(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	b := New(ctx, cancel)
	b.Rate = 0 // set rate to 0 so we can inject items synthetically
	b.itemFinishedChannel = make(chan struct{})

	worker := new(LoggingWorker)
	b.SetWorker(worker.NewSuccess)

	data := NewLoggingReadWriteCloser("a")
	log := NewLoggingReadWriteCloser("")
	output := NewLoggingReadWriteCloser("")

	b.SetData(data)
	b.SetLog(log)
	b.SetOutput(output)

	b.SetData(nil)
	b.SetLog(nil)
	b.SetOutput(nil)

	finished := make(chan error, 1)
	go func() {
		finished <- b.start(ctx)
	}()

	// synthetically call the main channel, which is what the ticker would do
	b.mainChannel <- 0
	<-b.itemFinishedChannel

	// start graceful exit process
	close(b.dataFinishedChannel)

	// wait for the start method to finish
	must(t, <-finished)

	b.Exit()

	data.mustNotRead(t)
	log.mustNotWrite(t)
	output.mustNotWrite(t)

	data.mustNotClose(t)
	log.mustNotClose(t)
	output.mustNotClose(t)

}

func TestSuccess(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	b := New(ctx, cancel)
	b.Rate = 0 // set rate to 0 so we can inject items synthetically
	b.itemFinishedChannel = make(chan struct{})

	worker := new(LoggingWorker)
	b.SetWorker(worker.NewSuccess)

	finished := make(chan error, 1)
	go func() {
		finished <- b.start(ctx)
	}()

	// synthetically call the main channel, which is what the ticker would do
	b.mainChannel <- 0
	<-b.itemFinishedChannel

	b.mainChannel <- 0
	<-b.itemFinishedChannel

	// start graceful exit process
	close(b.dataFinishedChannel)

	// wait for the start method to finish
	must(t, <-finished)

	b.Exit()

	worker.mustLen(t, 2)
	worker.must(t, 0, map[string]string{"_success": "true"})
	worker.must(t, 1, map[string]string{"_success": "true"})

}

func TestFail(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	b := New(ctx, cancel)
	b.Rate = 0 // set rate to 0 so we can inject items synthetically
	b.itemFinishedChannel = make(chan struct{})

	worker := new(LoggingWorker)
	b.SetWorker(worker.NewFail)

	finished := make(chan error, 1)
	go func() {
		finished <- b.start(ctx)
	}()

	// synthetically call the main channel, which is what the ticker would do
	b.mainChannel <- 0
	<-b.itemFinishedChannel

	b.mainChannel <- 0
	<-b.itemFinishedChannel

	// start graceful exit process
	close(b.dataFinishedChannel)

	// wait for the start method to finish
	must(t, <-finished)

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

	finished := make(chan error, 1)
	go func() {
		finished <- b.start(ctx)
	}()

	// synthetically call the main channel, which is what the ticker would do
	b.mainChannel <- 0
	<-b.itemFinishedChannel

	// start graceful exit process
	close(b.dataFinishedChannel)

	// wait for the start method to finish
	must(t, <-finished)

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

	finished := make(chan error, 1)
	go func() {
		finished <- b.start(ctx)
	}()

	// synthetically call the main channel, which is what the ticker would do
	b.mainChannel <- 0
	<-b.itemFinishedChannel

	// start graceful exit process
	close(b.dataFinishedChannel)

	// wait for the start method to finish
	must(t, <-finished)

	b.Exit()

	worker.mustLen(t, 1)
	worker.must(t, 0, map[string]string{"_cancelled": "true"})

}

func TestHardTimeout(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	b := New(ctx, cancel)
	b.Rate = 0 // set rate to 0 so we can inject items synthetically
	b.softTimeout = 5 * time.Millisecond
	b.hardTimeout = 10 * time.Millisecond
	b.itemFinishedChannel = make(chan struct{})

	worker := new(LoggingWorker)
	b.SetWorker(worker.NewHangForever)

	finished := make(chan error, 1)
	go func() {
		finished <- b.start(ctx)
	}()

	// synthetically call the main channel, which is what the ticker would do
	b.mainChannel <- 0
	<-b.itemFinishedChannel

	// start graceful exit process
	close(b.dataFinishedChannel)

	// wait for the start method to finish
	err := <-finished
	if err.Error() != "a worker was still sending after timeout + 1 second. This indicates a bug in the worker code. Workers should immediately exit on receiving a signal from ctx.Done()" {
		t.Fatal("Unexpected error:", err)
	}

	b.Exit()

}

func TestHardTimeoutCancel(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	b := New(ctx, cancel)
	b.Rate = 0 // set rate to 0 so we can inject items synthetically
	b.softTimeout = 10 * time.Millisecond
	b.hardTimeout = 20 * time.Millisecond

	worker := new(LoggingWorker)
	b.SetWorker(worker.NewHangForever)

	finished := make(chan error, 1)
	go func() {
		finished <- b.start(ctx)
	}()

	// synthetically call the main channel, which is what the ticker would do
	b.mainChannel <- 0

	b.Exit()

	// wait for the start method to finish
	err := <-finished
	if err.Error() != "a worker was still sending after timeout + 1 second. This indicates a bug in the worker code. Workers should immediately exit on receiving a signal from ctx.Done()" {
		t.Fatal("Unexpected error:", err)
	}

}

func TestCancel(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	b := New(ctx, cancel)
	b.Rate = 0 // set rate to 0 so we can inject items synthetically
	b.SetTimeout(500 * time.Millisecond)

	worker := new(LoggingWorker)
	b.SetWorker(worker.NewHang(400))

	finished := make(chan error, 1)
	go func() {
		finished <- b.start(ctx)
	}()

	// synthetically call the main channel, which is what the ticker would do
	b.mainChannel <- 0

	b.Exit()

	// wait for the start method to finish
	must(t, <-finished)

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

	input := &bytes.Buffer{}
	b.SetInput(input)

	log := &LoggingWriter{buf: new(bytes.Buffer)}
	b.SetLog(log)

	out := new(ThreadSafeBuffer)
	b.SetOutput(out)

	b.Headers = []string{"head"}
	b.SetData(strings.NewReader("a\nb"))

	finished := make(chan error, 1)
	go func() {
		finished <- b.start(ctx)
	}()

	// synthetically call the main channel, which is what the ticker would do
	b.mainChannel <- 0
	<-b.itemFinishedChannel

	b.mainChannel <- 0
	<-b.itemFinishedChannel

	// another tick and the data will reach EOF, and gracefully exit
	b.mainChannel <- 0

	// wait for the start method to finish
	must(t, <-finished)

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

	finished := make(chan error, 1)
	go func() {
		finished <- b.start(ctx)
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
	must(t, <-finished)

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

	finished := make(chan error, 1)
	go func() {
		finished <- b.start(ctx)
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
	must(t, <-finished)

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
	Result      bool
	Hang        int
	Log         *LoggingWorker
	HangForever bool
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

func (l *LoggingWorker) NewHangForever() Worker {
	return &loggingWorker{Log: l, HangForever: true}
}

func (l *loggingWorker) Send(ctx context.Context, in map[string]interface{}) (map[string]interface{}, error) {
	log := map[string]string{}
	if l.HangForever {
		<-time.After(100 * time.Second)
	} else if l.Hang > 0 {
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

func NewLoggingReadWriteCloser(data string) *LoggingReadWriteCloser {
	return &LoggingReadWriteCloser{
		Buf: bytes.NewBufferString(data),
	}
}

type LoggingReadWriteCloser struct {
	Buf      *bytes.Buffer
	DidRead  bool
	DidWrite bool
	DidClose bool
}

func (l *LoggingReadWriteCloser) Read(p []byte) (n int, err error) {
	l.DidRead = true
	return l.Buf.Read(p)
}

func (l *LoggingReadWriteCloser) Write(p []byte) (n int, err error) {
	l.DidWrite = true
	return l.Buf.Write(p)
}

func (l *LoggingReadWriteCloser) Close() error {
	l.DidClose = true
	return nil
}

func (l *LoggingReadWriteCloser) mustClose(t *testing.T) {
	t.Helper()
	if !l.DidClose {
		t.Fatal("Did not close")
	}
}
func (l *LoggingReadWriteCloser) mustRead(t *testing.T) {
	t.Helper()
	if !l.DidRead {
		t.Fatal("Did not read")
	}
}
func (l *LoggingReadWriteCloser) mustWrite(t *testing.T) {
	t.Helper()
	if !l.DidWrite {
		t.Fatal("Did not write")
	}
}
func (l *LoggingReadWriteCloser) mustNotClose(t *testing.T) {
	t.Helper()
	if l.DidClose {
		t.Fatal("Did close")
	}
}
func (l *LoggingReadWriteCloser) mustNotRead(t *testing.T) {
	t.Helper()
	if l.DidRead {
		t.Fatal("Did read")
	}
}
func (l *LoggingReadWriteCloser) mustNotWrite(t *testing.T) {
	t.Helper()
	if l.DidWrite {
		t.Fatal("Did write")
	}
}
