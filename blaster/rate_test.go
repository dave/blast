package blaster

import (
	"bytes"
	"context"
	"testing"
)

func TestChangeRate(t *testing.T) {

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

	if b.Rate != 0 {
		t.Fail()
	}

	b.ChangeRate(10.0)

	<-b.itemFinishedChannel // item will only finish once rate is changed

	if b.Rate != 10 {
		t.Fail()
	}

	b.Exit()

}

func TestInput(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	b := New(ctx, cancel)
	b.Rate = 0 // set rate to 0 so we can inject items synthetically
	b.itemFinishedChannel = make(chan struct{})

	input := &bytes.Buffer{}
	b.SetInput(input)

	workerLog := new(LoggingWorker)
	b.SetWorker(workerLog.NewSuccess)

	finished := make(chan struct{})
	go func() {
		must(t, b.start(ctx))
		close(finished)
	}()

	if b.Rate != 0 {
		t.Fail()
	}

	input.WriteString("10\n")

	<-b.itemFinishedChannel // item will only finish once rate is changed

	if b.Rate != 10 {
		t.Fail()
	}

	b.Exit()

}
