package blaster

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestWorkerVariants(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	b := New(ctx, cancel)
	b.Rate = 0 // set rate to 0 so we can inject items synthetically
	b.itemFinishedChannel = make(chan struct{})
	b.Workers = 9

	b.WorkerVariants = []map[string]string{
		{"index": "1"},
		{"index": "2"},
		{"index": "3"},
	}
	b.SetWorkerTemplate(map[string]interface{}{
		"index": "{{ .index }}",
	})

	b.SetWorker(func() Worker {
		return &ExampleWorker{
			StartFunc: func(ctx context.Context, self *ExampleWorker, payload map[string]interface{}) error {
				if self.Local == nil {
					self.Local = map[string]interface{}{}
				}
				self.Local["index"] = payload["index"]
				return nil
			},
			SendFunc: func(ctx context.Context, self *ExampleWorker, in map[string]interface{}) (map[string]interface{}, error) {
				return map[string]interface{}{"index": self.Local["index"]}, nil
			},
		}
	})

	log := &LoggingWriter{buf: new(bytes.Buffer)}
	b.SetLog(log)

	b.LogOutput = []string{"index"}

	finished := make(chan error, 1)
	go func() {
		finished <- b.start(ctx)
	}()

	// synthetically call the main channel, which is what the ticker would do
	for i := 0; i < 1000; i++ {
		b.mainChannel <- 0
		<-b.itemFinishedChannel
	}

	// another tick and the data will reach EOF, and gracefully exit
	close(b.dataFinishedChannel)

	// wait for the start method to finish
	must(t, <-finished)

	b.Exit()

	all := map[interface{}]int{}
	for _, value := range log.All() {
		all[value[2]]++
	}
	if len(all) != 3 {
		t.Fatal("Unexpected worker variants summary counts:", all)
	}
	if all["1"] > 500 || all["1"] < 200 ||
		all["2"] > 500 || all["2"] < 200 ||
		all["3"] > 500 || all["3"] < 200 {
		t.Fatal("Unexpected worker variants summary counts:", all)
	}

}

func TestDataLog(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	b := New(ctx, cancel)
	b.Rate = 0 // set rate to 0 so we can inject items synthetically
	b.itemFinishedChannel = make(chan struct{})

	b.SetWorker(func() Worker {
		return &ExampleWorker{
			SendFunc: func(ctx context.Context, self *ExampleWorker, in map[string]interface{}) (map[string]interface{}, error) {
				return map[string]interface{}{
					"d": fmt.Sprintf("d%s", in["index"]),
					"e": fmt.Sprintf("e%s", in["index"]),
					"f": fmt.Sprintf("f%s", in["index"]),
				}, nil
			},
		}
	})

	log := &LoggingWriter{buf: new(bytes.Buffer)}
	b.SetLog(log)

	out := new(ThreadSafeBuffer)
	b.SetOutput(out)

	b.Headers = []string{"i", "a", "b", "c"}
	b.SetData(strings.NewReader("1,a1,b1,c1\n2,a2,b2,c2"))
	b.SetPayloadTemplate(map[string]interface{}{
		"index": "{{ .i }}",
	})

	b.LogData = []string{"a", "c"}
	b.LogOutput = []string{"d", "f"}

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

	log.mustLen(t, 2)
	log.must(t, 0, []string{"9014d7c39df9fe2f|bc161f64e03e832e", "true", "a1", "c1", "d1", "f1"})
	log.must(t, 1, []string{"e049f1ccadc8a57d|c017a12dd7fee7fb", "true", "a2", "c2", "d2", "f2"})

}

func TestStringify(t *testing.T) {
	s := stringify([]string{"a", "b"})
	if s != `["a","b"]` {
		t.Fatal("Unexpected:", s)
	}
}
