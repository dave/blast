package blast

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

func (b *Blaster) startRateLoop(ctx context.Context) {

	b.mainWait.Add(1)

	readString := func() chan string {
		c := make(chan string)
		go func() {
			fmt.Println("New rate?")
			reader := bufio.NewReader(os.Stdin)
			text, _ := reader.ReadString('\n')
			c <- text
		}()
		return c
	}

	go func() {
		defer fmt.Println("Exiting rate loop")
		defer b.mainWait.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case <-b.dataFinishedChannel:
				return
			case s := <-readString():
				f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
				if err != nil {
					b.errorChannel <- errors.WithStack(err)
					return
				}
				b.changeRateChannel <- f
			}
		}
	}()

}

/*
func (b *Blaster) startRateLoop(ctx context.Context) {

	b.mainWait.Add(1)
	ticker := time.NewTicker(time.Second)
	pid := pidctrl.NewPIDController(0.5, 0.5, 0.5).
		SetOutputLimits(-1, 1).
		Set(b.config.MaxLatency / 100.0)

	go func() {
		defer fmt.Println("Exiting rate loop")
		defer b.mainWait.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case <-b.dataFinishedChannel:
				return
			case <-ticker.C:
				// recalculate rate
				success := atomic.LoadUint64(&b.stats.requestsSuccess)
				if success > INSTANT_COUNT {
					latency := float64(b.stats.requestsDurationQueue.Sum()/INSTANT_COUNT) / 100.0
					delta := pid.Update(latency)
					if b.rate+delta > b.config.MaxRate {
						b.changeRateChannel <- b.config.MaxRate
					} else if b.rate+delta < b.config.MinRate {
						b.changeRateChannel <- b.config.MinRate
					} else {
						b.changeRateChannel <- b.rate + delta
					}
					//fmt.Println("Rate changed to", b.rate+delta)
				}
			}
		}
	}()
}
*/
