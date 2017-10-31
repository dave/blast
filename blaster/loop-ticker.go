package blaster

import (
	"context"
	"fmt"
	"time"
)

func (b *Blaster) startTickerLoop(ctx context.Context) {

	b.mainWait.Add(1)
	ticker := time.NewTicker(time.Second / time.Duration(b.rate/float64(len(b.config.PayloadVariants))))

	go func() {
		defer b.mainWait.Done()
		defer fmt.Fprintln(b.out, "Exiting ticker loop")
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
				ticker = time.NewTicker(time.Second / time.Duration(b.rate/float64(len(b.config.PayloadVariants))))
				b.metrics.addSegment(b.rate)
				b.printStatus(false)
			case b.mainChannel <- struct{}{}:
				// if main loop is waiting, send it a message
			default:
				// if main loop is busy, skip this tick
				b.metrics.logMiss()
			}
		}
	}()
}
