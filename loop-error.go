package blast

import (
	"context"
	"fmt"
)

func (b *Blaster) startErrorLoop(ctx context.Context) {

	b.mainWait.Add(1)
	b.errorChannel = make(chan error)

	go func() {
		defer fmt.Fprintln(b.out, "Exiting error loop")
		defer b.mainWait.Done()
		for {
			select {
			// don't react to ctx.Done() here because we may need to wait until workers have finished
			case <-b.workersFinishedChannel:
				// exit gracefully
				return
			case err := <-b.errorChannel:
				fmt.Fprintf(b.out, "%+v\n", err)
				b.cancel()
				return
			}
		}
	}()
}
