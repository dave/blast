package blast

import (
	"context"
	"fmt"
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
		defer fmt.Fprintln(b.out, "Exiting status loop")
		defer b.mainWait.Done()
		for {
			select {
			// don't react to ctx.Done() here because we may need to wait until workers have finished
			case <-b.workersFinishedChannel:
				ticker.Stop()
				return
			case <-ticker.C:
				b.printStatus(false)
			}
		}
	}()
}

func (b *Blaster) printStatus(final bool) {
	success := atomic.LoadUint64(&b.stats.requestsSuccess)

	type def struct {
		status  int
		total   uint64
		instant uint64
	}
	statusQueue := b.stats.requestsStatusQueue.All()
	statusTotals := b.stats.requestsStatusTotal.All()
	summary := map[int]def{}
	for _, status := range statusQueue {
		if status == 0 {
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
		if status == 0 {
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
	w.Write([]byte(fmt.Sprintf("Finished:\t%d requests\n", atomic.LoadUint64(&b.stats.requestsFinished))))
	w.Write([]byte(fmt.Sprintf("Success:\t%d requests\n", atomic.LoadUint64(&b.stats.requestsSuccess))))
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
	} else {
		w.Write([]byte(fmt.Sprintf("Latency:\tn/a\n")))
	}

	if !final {
		w.Write([]byte(fmt.Sprintf("Concurrency:\t%d / %d workers in use\n", atomic.LoadInt64(&b.stats.workersBusy), b.config.Workers)))
	}
	skippedTicks := atomic.LoadUint64(&b.stats.ticksSkipped)
	if skippedTicks > 0 {
		w.Write([]byte(fmt.Sprintf("Skipped ticks:\t%d (when all workers are busy)\n", skippedTicks)))
	}
	w.Write([]byte("\t\n"))
	if len(ordered) > 0 {
		w.Write([]byte("Responses\t\n"))
		w.Write([]byte("=========\t\n"))
		for _, v := range ordered {
			w.Write([]byte(fmt.Sprintf("%d:\t%d requests (last %d: %d requests)\n", v.status, v.total, INSTANT_COUNT, v.instant)))
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

func NewThreadSaveMapIntInt() *ThreadSaveMapIntInt {
	return &ThreadSaveMapIntInt{
		data: map[int]uint64{},
	}
}

type ThreadSaveMapIntInt struct {
	data map[int]uint64
	m    sync.Mutex
}

func (t *ThreadSaveMapIntInt) Increment(key int) {
	t.m.Lock()
	defer t.m.Unlock()
	t.data[key] = t.data[key] + 1
}

func (t *ThreadSaveMapIntInt) All() map[int]uint64 {
	t.m.Lock()
	defer t.m.Unlock()
	out := map[int]uint64{}
	for k, v := range t.data {
		out[k] = v
	}
	return out
}

type FiloQueue struct {
	data   [INSTANT_COUNT]int
	m      sync.Mutex
	cursor int
}

func (f *FiloQueue) Add(v int) {
	f.m.Lock()
	defer f.m.Unlock()
	f.data[f.cursor] = v
	f.cursor++
	if f.cursor >= INSTANT_COUNT {
		f.cursor = 0
	}
}

func (f *FiloQueue) Sum() int {
	f.m.Lock()
	defer f.m.Unlock()
	var sum int
	for _, v := range f.data {
		sum += v
	}
	return sum
}

func (f *FiloQueue) All() [INSTANT_COUNT]int {
	f.m.Lock()
	defer f.m.Unlock()
	var out [INSTANT_COUNT]int
	for i, v := range f.data {
		out[i] = v
	}
	return out
}
