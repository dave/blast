package blast

import (
	"context"
	"fmt"
	"io"

	"github.com/pkg/errors"
)

func (b *Blaster) startMainLoop(ctx context.Context) {

	b.mainWait.Add(1)

	go func() {
		defer b.mainWait.Done()
		defer fmt.Fprintln(b.out, "Exiting main loop")
		for {
			select {
			case <-ctx.Done():
				return
			case <-b.mainChannel:
				for {
					record, err := b.dataReader.Read()
					if err != nil {
						if err == io.EOF {
							if b.config.Repeat {
								b.closeDataFile()
								if _, err := b.openDataFile(ctx); err != nil {
									b.error(errors.WithStack(err))
									return
								}
								continue
							} else {
								fmt.Fprintln(b.out, "Found end of data file")
								// finish gracefully
								close(b.dataFinishedChannel)
								return
							}
						}
						b.error(errors.WithStack(err))
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
