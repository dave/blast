package blaster

import (
	"context"
	"sync/atomic"
)

func (b *Blaster) startErrorLoop(ctx context.Context) {

	b.mainWait.Add(1)

	go func() {
		defer b.mainWait.Done()
		defer b.println("Exiting error loop")
		for {
			select {
			// don't react to ctx.Done() here because we may need to wait until workers have finished
			case <-b.workersFinishedChannel:
				// exit gracefully
				return
			case err := <-b.errorChannel:
				b.println("Exiting with fatal error...")
				b.err = err
				b.cancel()
				return
			}
		}
	}()
}

func (b *Blaster) error(err error) {
	select {
	case b.errorChannel <- err:
	default:
		// don't send to error channel if errorChannel isn't listening
		atomic.AddUint64(&b.errorsIgnored, 1)
	}
}
