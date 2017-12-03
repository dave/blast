package blaster

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestDataLog(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	b := New(ctx, cancel)
	b.Rate = 0 // set rate to 0 so we can inject items synthetically
	b.itemFinishedChannel = make(chan struct{})

	b.SetWorker(func() Worker {
		return &ExampleWorker{
			SendFunc: func(ctx context.Context, in map[string]interface{}) (map[string]interface{}, error) {
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
