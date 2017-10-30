package blast

import (
	"context"
	"fmt"
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
	w := tabwriter.NewWriter(b.out, 0, 0, 2, ' ', 0)
	b.metrics.summary(w)
	w.Flush()

	if !final {
		b.printRatePrompt()
	}
}

func (b *Blaster) printRatePrompt() {
	fmt.Fprintf(b.out, `
Current rate is %.0f requests / second. Enter a new rate or press enter to view status.

Rate?
`,
		b.rate,
	)
}
