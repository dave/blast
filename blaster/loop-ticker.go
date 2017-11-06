package blaster

import (
	"context"
	"time"
)

func (b *Blaster) startTickerLoop(ctx context.Context) {

	b.mainWait.Add(1)
	ticker := time.NewTicker(time.Second / time.Duration(b.Rate/float64(len(b.PayloadVariants))))

	go func() {
		defer b.mainWait.Done()
		defer b.println("Exiting ticker loop")
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
				b.Rate = rate
				ticker.Stop()
				ticker = time.NewTicker(time.Second / time.Duration(b.Rate/float64(len(b.PayloadVariants))))
				b.metrics.addSegment(b.Rate)
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
