package blast

import (
	"context"
	"fmt"
	"io"

	"encoding/json"

	"sync/atomic"

	"github.com/leemcloughlin/gofarmhash"
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
					j, err := json.Marshal(record)
					if err != nil {
						b.errorChannel <- errors.WithStack(err)
						return
					}
					hash := farmhash.Hash128(j)
					//hash := binary.BigEndian.Uint64(cityhash.New64().Sum(j))
					if b.skip != nil {
						if _, skip := b.skip[hash]; skip {
							atomic.AddUint64(&b.stats.itemsSkipped, 1)
							continue
						}
					}
					b.workerChannel <- workDef{Record: record, Hash: hash}
					break
				}
			}
		}
	}()
}

type workDef struct {
	Record []string
	Hash   farmhash.Uint128
}
