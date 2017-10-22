package blast

import (
	"context"
	"fmt"

	"strings"

	"sync/atomic"

	"time"

	"github.com/pkg/errors"
)

func (b *Blaster) startWorkers(ctx context.Context) {
	b.workerChannel = make(chan workDef)
	for i := 0; i < b.config.Workers; i++ {

		// assign rotated vars from config
		workerVariantData := map[string]string{}
		if b.config.WorkerVariants != nil {
			for k, v := range b.config.WorkerVariants[i%len(b.config.WorkerVariants)] {
				workerVariantData[k] = v
			}
		}

		workerFunc := b.workerTypes[b.config.WorkerType]
		w := workerFunc()

		if s, ok := w.(Starter); ok {
			workerSetup := replaceMap(b.config.WorkerTemplate, workerVariantData)
			if err := s.Start(ctx, workerSetup); err != nil {
				b.errorChannel <- errors.WithStack(err)
				return
			}
		}

		b.workerWait.Add(1)
		go func(index int) {
			defer fmt.Println("Exiting worker", index)
			defer func() {
				if s, ok := w.(Stopper); ok {
					workerSetup := replaceMap(b.config.WorkerTemplate, workerVariantData)
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
					atomic.AddInt64(&b.stats.workersBusy, 1)

					data := map[string]string{}
					for k, v := range workerVariantData {
						data[k] = v
					}
					for i, key := range b.dataHeaders {
						data[key] = work.Record[i]
					}

					atomic.AddUint64(&b.stats.itemsStarted, 1)

					success := true
					for _, variationData := range b.config.PayloadVariants {
						atomic.AddUint64(&b.stats.requestsStarted, 1)
						start := time.Now()
						if err := b.sendWithPayloadVariation(ctx, w, data, variationData); err != nil {
							success = false
						}
						elapsed := time.Since(start).Nanoseconds() / 1000000
						atomic.AddUint64(&b.stats.requestsFinished, 1)
						if success {
							// only log
							atomic.AddUint64(&b.stats.requestsSuccess, 1)
							atomic.AddUint64(&b.stats.requestsSuccessDuration, uint64(elapsed))
							b.stats.requestsDurationQueue.Add(uint64(elapsed))
						}
						if !success {
							break
						}
					}
					if success {
						atomic.AddUint64(&b.stats.itemsSuccess, 1)
					} else {
						atomic.AddUint64(&b.stats.itemsFailed, 1)
					}

					atomic.AddUint64(&b.stats.itemsFinished, 1)

					var extraFields []string
					for _, key := range b.config.LogData {
						extraFields = append(extraFields, data[key])
					}

					lr := logRecord{
						PayloadHash: work.Hash,
						Result:      success,
						ExtraFields: extraFields,
					}
					b.logChannel <- lr
					atomic.AddInt64(&b.stats.workersBusy, -1)
				}
			}
		}(i)
	}
}

func (b *Blaster) sendWithPayloadVariation(ctx context.Context, w Worker, payloadData map[string]string, variationData map[string]string) error {

	data := map[string]string{}

	for k, v := range payloadData {
		data[k] = v
	}

	if variationData != nil {
		for k, v := range variationData {
			data[k] = v
		}
	}

	renderedTemplate := replaceMap(b.config.PayloadTemplate, data)

	if err := w.Send(ctx, renderedTemplate); err != nil {
		return errors.WithStack(err)
	}

	return nil
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
