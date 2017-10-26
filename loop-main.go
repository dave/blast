package blast

import (
	"context"
	"fmt"
	"io"

	"github.com/pkg/errors"
)

func (b *Blaster) startMainLoop(ctx context.Context) {

	b.mainWait.Add(1)
	b.mainChannel = make(chan struct{})

	go func() {
		defer fmt.Fprintln(b.out, "Exiting main loop")
		defer b.mainWait.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case <-b.mainChannel:
				for {
					record, err := b.dataReader.Read()
					if err != nil {
						if err == io.EOF {
							fmt.Fprintln(b.out, "Found end of data file")
							// finish gracefully
							close(b.dataFinishedChannel)
							return
						}
						b.errorChannel <- errors.WithStack(err)
						return
					}
					b.workerChannel <- workDef{Record: record}
					break
				}
			}
		}
	}()
}

type workDef struct {
	Record []string
}
