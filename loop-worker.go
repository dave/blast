package blast

import (
	"context"
	"fmt"

	"strings"

	"sync/atomic"

	"time"

	"encoding/json"

	"github.com/leemcloughlin/gofarmhash"
	"github.com/pkg/errors"
)

func (b *Blaster) startWorkers(ctx context.Context) {
	b.workerChannel = make(chan workDef)
	for i := 0; i < b.config.Workers; i++ {

		// assign rotated vars from config
		workerVariationData := map[string]string{}
		if b.config.WorkerVariants != nil {
			for k, v := range b.config.WorkerVariants[i%len(b.config.WorkerVariants)] {
				workerVariationData[k] = v
			}
		}

		workerFunc := b.workerTypes[b.config.WorkerType]
		w := workerFunc()

		if s, ok := w.(Starter); ok {
			workerSetup := replaceMap(b.config.WorkerTemplate, workerVariationData)
			if err := s.Start(ctx, workerSetup); err != nil {
				b.errorChannel <- errors.WithStack(err)
				return
			}
		}

		b.workerWait.Add(1)
		go func(index int) {
			defer fmt.Fprintln(b.out, "Exiting worker", index)
			defer func() {
				if s, ok := w.(Stopper); ok {
					workerSetup := replaceMap(b.config.WorkerTemplate, workerVariationData)
					if err := s.Stop(ctx, workerSetup); err != nil {
						b.errorChannel <- errors.WithStack(err)
						return
					}
				}
			}()
			defer b.workerWait.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case <-b.dataFinishedChannel:
					// exit gracefully
					return
				case work := <-b.workerChannel:
					for _, payloadVariationData := range b.config.PayloadVariants {
						atomic.AddInt64(&b.stats.workersBusy, 1)
						b.send(ctx, w, workerVariationData, work, payloadVariationData)
						atomic.AddInt64(&b.stats.workersBusy, -1)
					}

				}
			}
		}(i)
	}
}

func (b *Blaster) send(ctx context.Context, w Worker, workerVariantData map[string]string, work workDef, variationData map[string]string) {

	data := map[string]string{}
	for k, v := range workerVariantData {
		data[k] = v
	}
	for i, k := range b.dataHeaders {
		data[k] = work.Record[i]
	}
	for k, v := range variationData {
		data[k] = v
	}

	j, err := json.Marshal(data)
	if err != nil {
		b.errorChannel <- errors.WithStack(err)
		return
	}
	hash := farmhash.Hash128(j)
	if b.skip != nil {
		if _, skip := b.skip[hash]; skip {
			atomic.AddUint64(&b.stats.requestsSkipped, 1)
			return
		}
	}

	atomic.AddUint64(&b.stats.requestsStarted, 1)
	start := time.Now()
	renderedTemplate := replaceMap(b.config.PayloadTemplate, data)

	success := true
	out, err := w.Send(ctx, renderedTemplate)
	if err != nil {
		success = false
	}
	if out == nil {
		out = map[string]interface{}{}
	}
	elapsed := time.Since(start).Nanoseconds() / 1000000
	if success {
		atomic.AddUint64(&b.stats.requestsSuccess, 1)
		atomic.AddUint64(&b.stats.requestsSuccessDuration, uint64(elapsed))
		b.stats.requestsDurationQueue.Add(uint64(elapsed))
	} else {
		atomic.AddUint64(&b.stats.requestsFailed, 1)
	}
	atomic.AddUint64(&b.stats.requestsFinished, 1)

	var extraFields []string
	for _, key := range b.config.LogData {
		extraFields = append(extraFields, data[key])
	}
	for _, key := range b.config.LogOutput {
		var val string
		switch v := out[key].(type) {
		case string:
			val = v
		case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, uintptr, float32, float64, complex64, complex128:
			val = fmt.Sprint(v)
		default:
			j, _ := json.Marshal(v)
			val = string(j)
		}
		extraFields = append(extraFields, val)
	}

	lr := logRecord{
		PayloadHash: hash,
		Result:      success,
		ExtraFields: extraFields,
	}
	b.logChannel <- lr
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
