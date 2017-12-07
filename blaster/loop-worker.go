package blaster

import (
	"context"
	"fmt"

	"time"

	"encoding/json"

	"github.com/pkg/errors"
)

func (b *Blaster) startWorkers(ctx context.Context) {
	for i := 0; i < b.Workers; i++ {

		// assign rotated vars from config
		workerVariantData := map[string]string{}
		if b.WorkerVariants != nil {
			for k, v := range b.WorkerVariants[i%len(b.WorkerVariants)] {
				workerVariantData[k] = v
			}
		}

		w := b.workerFunc()

		if s, ok := w.(Starter); ok {
			workerSetup, err := renderMap(b.workerRenderer, workerVariantData)
			if err != nil {
				// notest
				b.error(err)
				return
			}
			if err := s.Start(ctx, workerSetup); err != nil {
				// notest
				b.error(errors.WithStack(err))
				return
			}
		}

		b.workerWait.Add(1)
		go func(index int) {
			defer b.workerWait.Done()
			defer func() {
				if s, ok := w.(Stopper); ok {
					workerSetup, err := renderMap(b.workerRenderer, workerVariantData)
					if err != nil {
						// notest
						b.error(err)
						return
					}
					if err := s.Stop(ctx, workerSetup); err != nil {
						// notest
						b.error(errors.WithStack(err))
						return
					}
				}
			}()

			for {
				select {
				case <-ctx.Done():
					return
				case <-b.dataFinishedChannel:
					// exit gracefully
					return
				case work := <-b.workerChannel:
					if err := b.send(ctx, w, work); err != nil {
						// notest
						b.error(err)
						return
					}
					if b.itemFinishedChannel != nil {
						// only used in tests
						b.itemFinishedChannel <- struct{}{}
					}
				}
			}
		}(i)
	}
}

func (b *Blaster) send(ctx context.Context, w Worker, work workDef) error {

	b.metrics.logStart(work.segment)
	b.metrics.logBusy(work.segment)
	b.metrics.busy.Inc(1)
	defer b.metrics.busy.Dec(1)

	// Record the start time
	start := time.Now()

	// Render the payload template with the data generated above
	renderedTemplate, err := renderMap(b.payloadRenderer, work.data)
	if err != nil {
		return err
	}

	// Create a child context with the selected timeout
	child, cancel := context.WithTimeout(ctx, b.softTimeout)
	defer cancel()

	finished := make(chan struct{})

	success := true
	var out map[string]interface{}
	go func() {
		out, err = w.Send(child, renderedTemplate)
		if err != nil {
			success = false
		}
		close(finished)
	}()

	var hardTimeoutExceeded bool
	select {
	case <-finished:
		// When Send finishes successfully, cancel the child context.
		cancel()
	case <-ctx.Done():
		// In the event of the main context being cancelled, cancel the child context and wait for
		// the sending goroutine to exit.
		cancel()
		select {
		case <-finished: // notest
			// Only continue when finished channel is closed - e.g. sending goroutine has exited.
		case <-time.After(b.hardTimeout):
			hardTimeoutExceeded = true
		}
	case <-time.After(b.hardTimeout):
		hardTimeoutExceeded = true
	}

	if hardTimeoutExceeded {
		// If we get here then the worker is not respecting the context cancellation deadline, and
		// we should exit with an error. We don't simply log this as an unsuccessful request
		// because the sending goroutine is still running and would crete a memory leak.
		b.error(errors.New("a worker was still sending after timeout + 1 second. This indicates a bug in the worker code. Workers should immediately exit on receiving a signal from ctx.Done()"))
		return nil
	}

	var val string
	if out != nil {
		if status, ok := out["status"]; ok {
			val = stringify(status)
		}
	}
	if val == "" {
		val = "(none)"
	}
	b.metrics.logFinish(work.segment, val, time.Since(start), success)

	if b.logWriter != nil {
		var fields []string
		for _, key := range b.LogData {
			var val string
			if v, ok := work.data[key]; ok {
				val = v
			}
			fields = append(fields, val)
		}
		for _, key := range b.LogOutput {
			var val string
			if out != nil {
				if v, ok := out[key]; ok {
					val = stringify(v)
				}
			}
			fields = append(fields, val)
		}

		lr := logRecord{
			hash:   work.hash,
			result: success,
			fields: fields,
		}
		b.logChannel <- lr
	}
	return nil
}

func renderMap(r renderer, data map[string]string) (map[string]interface{}, error) {
	if r == nil {
		return map[string]interface{}{}, nil
	}
	rendered, err := r.render(data)
	if err != nil {
		return nil, err
	}
	renderedMap, ok := rendered.(map[string]interface{})
	if !ok {
		return nil, errors.New("rendered template not a map")
	}
	return renderedMap, nil
}

func stringify(v interface{}) string {
	switch v := v.(type) {
	case string:
		return v
	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, uintptr, float32, float64, complex64, complex128:
		return fmt.Sprint(v)
	default:
		j, _ := json.Marshal(v)
		return string(j)
	}
}

// ExampleWorker facilitates code examples by satisfying the Worker, Starter and Stopper interfaces with provided functions.
type ExampleWorker struct {
	SendFunc  func(ctx context.Context, self *ExampleWorker, in map[string]interface{}) (map[string]interface{}, error)
	StartFunc func(ctx context.Context, self *ExampleWorker, payload map[string]interface{}) error
	StopFunc  func(ctx context.Context, self *ExampleWorker, payload map[string]interface{}) error
	Local     map[string]interface{}
}

// Send satisfies the Worker interface.
func (e *ExampleWorker) Send(ctx context.Context, in map[string]interface{}) (map[string]interface{}, error) {
	// notest
	if e.SendFunc != nil {
		return e.SendFunc(ctx, e, in)
	}
	return nil, nil
}

// Start satisfies the Starter interface.
func (e *ExampleWorker) Start(ctx context.Context, payload map[string]interface{}) error {
	// notest
	if e.StartFunc != nil {
		return e.StartFunc(ctx, e, payload)
	}
	return nil
}

// Stop satisfies the Stopper interface.
func (e *ExampleWorker) Stop(ctx context.Context, payload map[string]interface{}) error {
	// notest
	if e.StopFunc != nil {
		return e.StopFunc(ctx, e, payload)
	}
	return nil
}
