package blaster

import (
	"context"
	"fmt"
	"io"

	"encoding/json"

	"github.com/leemcloughlin/gofarmhash"
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

					skipped := true

					for _, payloadVariantData := range b.config.PayloadVariants {

						// Build the full data map that will be passed to the worker
						data := map[string]string{}
						for i, k := range b.dataHeaders {
							// Add data from the CSV data source
							data[k] = record[i]
						}
						for k, v := range payloadVariantData {
							// Add data from the payload-variants config
							data[k] = v
						}

						// Calculate the hash of the incoming data
						j, err := json.Marshal(data)
						if err != nil {
							b.error(errors.WithStack(err))
							return
						}
						hash := farmhash.Hash128(j)

						// In resume mode, check to see if the hash occurred in a previous run
						// (skip only contains successful requests from previous runs).
						if b.config.Resume {
							if _, skip := b.skip[hash]; skip {
								b.metrics.logSkip()
								continue
							}
						}

						skipped = false

						b.workerChannel <- workDef{Data: data, Hash: hash}
					}
					if skipped {
						// if we've skipped all variants, continue with the next item immediately
						continue
					}
					break
				}
			}
		}
	}()
}

type workDef struct {
	Data map[string]string
	Hash farmhash.Uint128
}
