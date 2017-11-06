package blaster

import (
	"context"
)

func (b *Blaster) startLogLoop(ctx context.Context) {

	if b.logWriter == nil {
		return
	}

	b.mainWait.Add(1)

	go func() {
		defer b.mainWait.Done()
		defer b.println("Exiting log loop")
		var count uint64
		for {
			count++
			select {
			// don't react to ctx.Done() here because we may need to wait until workers have finished
			case <-b.workersFinishedChannel:
				// exit gracefully
				return
			case lr := <-b.logChannel:
				b.logWriter.Write(lr.toCsv())
				if count%1000 == 0 {
					b.logWriter.Flush()
				}
			}
		}
	}()
}
