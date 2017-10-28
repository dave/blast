package blast

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"text/tabwriter"
	"time"
)

func (b *Blaster) startStatusLoop(ctx context.Context) {

	b.mainWait.Add(1)
	ticker := time.NewTicker(time.Second * 10)

	go func() {
		defer b.mainWait.Done()
		defer fmt.Fprintln(b.out, "Exiting status loop")
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-b.dataFinishedChannel:
				ticker.Stop()
				return
			case <-ticker.C:
				b.printStatus(false)
			}
		}
	}()
}

func (b *Blaster) printStatus(final bool) {
	type def struct {
		status  string
		total   uint64
		instant uint64
	}
	statusQueue := b.stats.requestsStatusQueue.All()
	statusTotals := b.stats.requestsStatusTotal.All()
	summary := map[string]def{}
	for _, status := range statusQueue {
		if status == "" {
			continue
		}
		if _, found := summary[status]; !found {
			summary[status] = def{
				status:  status,
				total:   0,
				instant: 0,
			}
		}
		current := summary[status]
		summary[status] = def{
			status:  current.status,
			total:   current.total,
			instant: current.instant + 1,
		}
	}
	for status, total := range statusTotals {
		if status == "" {
			continue
		}
		if _, found := summary[status]; !found {
			summary[status] = def{
				status:  status,
				total:   0,
				instant: 0,
			}
		}
		current := summary[status]
		summary[status] = def{
			status:  current.status,
			total:   total,
			instant: current.instant,
		}
	}
	var ordered []def
	for _, v := range summary {
		ordered = append(ordered, v)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].status < ordered[j].status })

	w := tabwriter.NewWriter(b.out, 0, 0, 1, ' ', 0)

	w.Write([]byte("\n"))
	w.Write([]byte("Summary\n"))
	w.Write([]byte("=======\n"))
	if !final {
		if len(b.config.PayloadVariants) > 1 {
			requestsPerSec := b.rate * float64(len(b.config.PayloadVariants))
			w.Write([]byte(fmt.Sprintf("Rate:\t%.0f items/sec (%.0f requests/sec)\n", b.rate, requestsPerSec)))
		} else {
			w.Write([]byte(fmt.Sprintf("Rate:\t%.0f items/sec \n", b.rate)))
		}
	}
	w.Write([]byte(fmt.Sprintf("Started:\t%d requests\n", atomic.LoadUint64(&b.stats.requestsStarted))))

	finished := atomic.LoadUint64(&b.stats.requestsFinished)
	w.Write([]byte(fmt.Sprintf("Finished:\t%d requests\n", finished)))

	success := atomic.LoadUint64(&b.stats.requestsSuccess)
	w.Write([]byte(fmt.Sprintf("Success:\t%d requests\n", success)))
	w.Write([]byte(fmt.Sprintf("Failed:\t%d requests\n", atomic.LoadUint64(&b.stats.requestsFailed))))

	requestsSkipped := atomic.LoadUint64(&b.stats.requestsSkipped)
	if requestsSkipped > 0 {
		w.Write([]byte(fmt.Sprintf("Skipped:\t%d requests (from previous run)\n", requestsSkipped)))
	}

	if success > 0 {
		durationTotal := atomic.LoadUint64(&b.stats.requestsSuccessDuration) / success
		if success > INSTANT_COUNT {
			durationInstant := b.stats.requestsDurationQueue.Sum() / INSTANT_COUNT
			w.Write([]byte(fmt.Sprintf("Latency:\t%v ms per request (last %d: %v ms per request)\n", durationTotal, INSTANT_COUNT, durationInstant)))
		} else {
			w.Write([]byte(fmt.Sprintf("Latency:\t%v ms per request\n", durationTotal)))
		}
	}

	if !final {
		w.Write([]byte(fmt.Sprintf("Concurrency:\t%d / %d workers in use\n", atomic.LoadInt64(&b.stats.workersBusy), b.config.Workers)))
	}
	skippedTicks := atomic.LoadUint64(&b.stats.ticksSkipped)
	if skippedTicks > 0 {
		w.Write([]byte(fmt.Sprintf("Skipped ticks:\t%d (when all workers are busy)\n", skippedTicks)))
	}

	if DEBUG {
		w.Write([]byte(fmt.Sprintf("Goroutines:\t%d\n", runtime.NumGoroutine())))
	}

	w.Write([]byte("\t\n"))
	if len(ordered) > 0 {
		w.Write([]byte("Responses\t\n"))
		w.Write([]byte("=========\t\n"))
		for _, v := range ordered {
			if finished > INSTANT_COUNT {
				w.Write([]byte(fmt.Sprintf("%s:\t%d requests (last %d: %d requests)\n", v.status, v.total, INSTANT_COUNT, v.instant)))
			} else {
				w.Write([]byte(fmt.Sprintf("%s:\t%d requests\n", v.status, v.total)))
			}
		}
		w.Write([]byte("\n"))
	}
	w.Flush()

	if !final {
		b.printRatePrompt()
	}
}

func (b *Blaster) printRatePrompt() {
	fmt.Fprintf(b.out, `
Current rate is %.0f items / second. Enter a new rate or press enter to view status.

Rate?
`,
		b.rate,
	)
}

func NewThreadSaveMapStringInt() *ThreadSaveMapStringInt {
	return &ThreadSaveMapStringInt{
		data: map[string]uint64{},
	}
}

type ThreadSaveMapStringInt struct {
	data map[string]uint64
	m    sync.Mutex
}

func (t *ThreadSaveMapStringInt) Increment(key string) {
	t.m.Lock()
	defer t.m.Unlock()
	t.data[key] = t.data[key] + 1
}

func (t *ThreadSaveMapStringInt) All() map[string]uint64 {
	t.m.Lock()
	defer t.m.Unlock()
	out := map[string]uint64{}
	for k, v := range t.data {
		out[k] = v
	}
	return out
}

type FiloQueueInt struct {
	data   [INSTANT_COUNT]int
	m      sync.Mutex
	cursor int
}

func (f *FiloQueueInt) Add(v int) {
	f.m.Lock()
	defer f.m.Unlock()
	f.data[f.cursor] = v
	f.cursor++
	if f.cursor >= INSTANT_COUNT {
		f.cursor = 0
	}
}

func (f *FiloQueueInt) Sum() int {
	f.m.Lock()
	defer f.m.Unlock()
	var sum int
	for _, v := range f.data {
		sum += v
	}
	return sum
}

type FiloQueueString struct {
	data   [INSTANT_COUNT]string
	m      sync.Mutex
	cursor int
}

func (f *FiloQueueString) Add(v string) {
	f.m.Lock()
	defer f.m.Unlock()
	f.data[f.cursor] = v
	f.cursor++
	if f.cursor >= INSTANT_COUNT {
		f.cursor = 0
	}
}

func (f *FiloQueueString) All() [INSTANT_COUNT]string {
	f.m.Lock()
	defer f.m.Unlock()
	var out [INSTANT_COUNT]string
	for i, v := range f.data {
		out[i] = v
	}
	return out
}
