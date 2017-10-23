package blast

import (
	"fmt"
	"sync"
	"sync/atomic"
)

/*
func (b *Blaster) startStatusLoop(ctx context.Context) {

	b.mainWait.Add(1)
	ticker := time.NewTicker(time.Second * 5)

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
				b.printStatus()
			}
		}
	}()
}
*/

func (b *Blaster) printStatus() {
	var durationTotal, durationInstant uint64
	success := atomic.LoadUint64(&b.stats.requestsSuccess)
	if success > 0 {
		durationTotal = atomic.LoadUint64(&b.stats.requestsSuccessDuration) / success
	}
	if success > INSTANT_COUNT {
		durationInstant = b.stats.requestsDurationQueue.Sum() / INSTANT_COUNT
	}
	fmt.Fprintf(b.out, `
Status
======
Rate:          %.0f items / second
Started:       %d items (%d requests)
Finished:      %d items (%d requests)
Success:       %d
Failed:        %d
Skipped:       %d (from previous run)
Latency:       %v ms (last %d requests: %v ms)
Concurrency:   %d / %d workers in use
Skipped ticks: %d (when all workers are busy)

`,
		b.rate,
		atomic.LoadUint64(&b.stats.itemsStarted),
		atomic.LoadUint64(&b.stats.requestsStarted),
		atomic.LoadUint64(&b.stats.itemsFinished),
		atomic.LoadUint64(&b.stats.requestsFinished),
		atomic.LoadUint64(&b.stats.itemsSuccess),
		atomic.LoadUint64(&b.stats.itemsFailed),
		atomic.LoadUint64(&b.stats.itemsSkipped),
		durationTotal,
		INSTANT_COUNT,
		durationInstant,
		atomic.LoadInt64(&b.stats.workersBusy),
		b.config.Workers,
		atomic.LoadUint64(&b.stats.ticksSkipped),
	)
}

func (b *Blaster) printRatePrompt() {
	fmt.Fprintf(b.out, `
Current rate is %.0f items / second. Enter a new rate or press enter to view status.

Rate?
`,
		b.rate,
	)
}

type FiloQueue struct {
	data   [INSTANT_COUNT]uint64
	m      sync.Mutex
	cursor int
}

func (f *FiloQueue) Add(v uint64) {
	f.m.Lock()
	defer f.m.Unlock()
	f.data[f.cursor] = v
	f.cursor++
	if f.cursor >= INSTANT_COUNT {
		f.cursor = 0
	}
}

func (f *FiloQueue) Sum() uint64 {
	f.m.Lock()
	defer f.m.Unlock()
	var sum uint64
	for _, v := range f.data {
		sum += v
	}
	return sum
}
