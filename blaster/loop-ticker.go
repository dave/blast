package blaster

import (
	"context"
	"time"
)

func (b *Blaster) startTickerLoop(ctx context.Context) {

	b.mainWait.Add(1)

	var ticker *time.Ticker

	updateTicker := func() {
		if b.Rate == 0 {
			ticker = &time.Ticker{} // empty *time.Ticker will have nil C, so block forever.
			return
		}
		ticker = time.NewTicker(time.Second / time.Duration(b.Rate/float64(len(b.PayloadVariants))))
	}

	changeRate := func(rate float64) {
		b.Rate = rate
		if ticker != nil {
			ticker.Stop()
		}
		b.metrics.addSegment(b.Rate)
		updateTicker()
		b.printStatus(false)
	}

	updateTicker()

	go func() {
		defer b.mainWait.Done()
		defer b.println("Exiting ticker loop")
		defer func() {
			if ticker != nil {
				ticker.Stop()
			}
		}()
		for {

			// First wait for a tick... but we should also wait for an exit signal, data finished
			// signal or rate change command (we could be waiting forever on rate = 0).
			select {
			case <-ticker.C:
				// continue
			case <-ctx.Done():
				return
			case <-b.dataFinishedChannel:
				return
			case rate := <-b.changeRateChannel:
				// Restart the for loop after a rate change. If rate == 0, we may not want to send
				// any more.
				changeRate(rate)
				continue
			}

			segment := b.metrics.currentSegment()

			// Next send on the main channel. The channel won't have a listener if there is no idle
			// worker. In this case we should continue and log a miss.
			select {
			case b.mainChannel <- segment:
				// if main loop is waiting, send it a message
			case <-ctx.Done():
				return
			case <-b.dataFinishedChannel:
				return
			default:
				// if main loop is busy, skip this tick
				b.metrics.logMiss(segment)
			}
		}
	}()
}
