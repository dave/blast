package blast

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

func (b *Blaster) startTickerLoop(ctx context.Context) {

	b.mainWait.Add(1)
	ticker := time.NewTicker(time.Second / time.Duration(b.rate))

	go func() {
		defer fmt.Fprintln(b.out, "Exiting ticker loop")
		defer b.mainWait.Done()
		for {
			<-ticker.C
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-b.dataFinishedChannel:
				ticker.Stop()
				return
			case rate := <-b.changeRateChannel:
				b.rate = rate
				ticker.Stop()
				ticker = time.NewTicker(time.Second / time.Duration(b.rate))
				b.printStatus()
			case b.mainChannel <- struct{}{}:
				// if main loop is waiting, send it a message
			default:
				// if main loop is busy, skip this tick
				atomic.AddUint64(&b.stats.ticksSkipped, 1)
			}
		}
	}()
}
