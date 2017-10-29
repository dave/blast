package blast

import (
	"context"
	"fmt"

	"strings"

	"sync/atomic"

	"time"

	"encoding/json"

	"github.com/pkg/errors"
)

func (b *Blaster) startWorkers(ctx context.Context) {
	for i := 0; i < b.config.Workers; i++ {

		// assign rotated vars from config
		workerVariantData := map[string]string{}
		if b.config.WorkerVariants != nil {
			for k, v := range b.config.WorkerVariants[i%len(b.config.WorkerVariants)] {
				workerVariantData[k] = v
			}
		}

		workerFunc, ok := b.workerTypes[b.config.WorkerType]
		if !ok {
			b.error(errors.Errorf("Worker type %s not found", b.config.WorkerType))
			return
		}
		w := workerFunc()

		if s, ok := w.(Starter); ok {
			workerSetup := replaceMap(b.config.WorkerTemplate, workerVariantData)
			if err := s.Start(ctx, workerSetup); err != nil {
				b.error(errors.WithStack(err))
				return
			}
		}

		b.workerWait.Add(1)
		go func(index int) {
			defer b.workerWait.Done()
			defer func() {
				if s, ok := w.(Stopper); ok {
					workerSetup := replaceMap(b.config.WorkerTemplate, workerVariantData)
					if err := s.Stop(ctx, workerSetup); err != nil {
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
					b.send(ctx, w, work)
				}
			}
		}(i)
	}
}

func (b *Blaster) send(ctx context.Context, w Worker, work workDef) {

	atomic.AddInt64(&b.stats.workersBusy, 1)
	defer atomic.AddInt64(&b.stats.workersBusy, -1)

	// Count the started request
	atomic.AddUint64(&b.stats.requestsStarted, 1)

	// Record the start time
	start := time.Now()

	// Render the payload template with the data generated above
	renderedTemplate := replaceMap(b.config.PayloadTemplate, work.Data)

	// Create a child context with the selected timeout
	child, cancel := context.WithTimeout(ctx, b.softTimeout)
	defer cancel()

	finished := make(chan struct{})

	success := true
	var out map[string]interface{}
	var err error
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
		case <-finished:
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
		b.error(errors.New("A worker was still sending after timeout + 500ms. This indicates a bug in the worker code. Workers should immediately exit on receiving a signal from ctx.Done()."))
		return
	}

	elapsed := time.Since(start).Nanoseconds() / 1000000
	if success {
		atomic.AddUint64(&b.stats.requestsSuccess, 1)
		atomic.AddUint64(&b.stats.requestsSuccessDuration, uint64(elapsed))
		b.stats.requestsDurationQueue.Add(int(elapsed))
	} else {
		atomic.AddUint64(&b.stats.requestsFailed, 1)
	}
	atomic.AddUint64(&b.stats.requestsFinished, 1)

	var val string
	if out != nil {
		if status, ok := out["status"]; ok {
			val = stringify(status)
		}
	}
	if val == "" {
		val = "(none)"
	}
	b.stats.requestsStatusTotal.Increment(val)
	b.stats.requestsStatusQueue.Add(val)

	var fields []string
	for _, key := range b.config.LogData {
		var val string
		if v, ok := work.Data[key]; ok {
			val = v
		}
		fields = append(fields, val)
	}
	for _, key := range b.config.LogOutput {
		var val string
		if out != nil {
			if v, ok := out[key]; ok {
				val = stringify(v)
			}
		}
		fields = append(fields, val)
	}

	lr := logRecord{
		Hash:   work.Hash,
		Result: success,
		Fields: fields,
	}
	b.logChannel <- lr
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

func replace(template interface{}, substitutions map[string]string) interface{} {
	switch template := template.(type) {
	case string:
		return replaceString(template, substitutions)
	case map[string]interface{}:
		return replaceMap(template, substitutions)
	case []interface{}:
		return replaceSlice(template, substitutions)
	}
	return template
}

func replaceMap(template map[string]interface{}, substitutions map[string]string) map[string]interface{} {
	out := map[string]interface{}{}
	for k, v := range template {
		out[k] = replace(v, substitutions)
	}
	return out
}

func replaceSlice(template []interface{}, substitutions map[string]string) []interface{} {
	out := []interface{}{}
	for _, v := range template {
		out = append(out, replace(v, substitutions))
	}
	return out
}

func replaceString(template string, substitutions map[string]string) string {
	out := template
	for key, sub := range substitutions {
		out = strings.Replace(out, fmt.Sprint("{{", key, "}}"), sub, -1)
	}
	return out
}
