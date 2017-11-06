package blaster

import (
	"context"
	"io"

	"encoding/json"

	"github.com/leemcloughlin/gofarmhash"
	"github.com/pkg/errors"
)

func (b *Blaster) startMainLoop(ctx context.Context) {

	b.mainWait.Add(1)

	go func() {
		defer b.mainWait.Done()
		defer b.println("Exiting main loop")
		for {
			select {
			case <-ctx.Done():
				return
			case <-b.mainChannel:
				for {
					var record []string
					if b.dataReader != nil {
						var err error
						record, err = b.dataReader.Read()
						if err != nil {
							if err == io.EOF {
								b.println("Found end of data file")
								// finish gracefully
								close(b.dataFinishedChannel)
								return
							}
							b.error(errors.WithStack(err))
							return
						}
					}

					skipped := true

					for _, payloadVariantData := range b.PayloadVariants {

						// Build the full data map that will be passed to the worker
						data := map[string]string{}
						for i, k := range b.Headers {
							// Add data from the CSV data source
							data[k] = record[i]
						}
						for k, v := range payloadVariantData {
							// Add data from the payload-variants config
							data[k] = v
						}

						var hash farmhash.Uint128
						if b.logWriter != nil {
							// Calculate the hash of the incoming data
							j, err := json.Marshal(data)
							if err != nil {
								b.error(errors.WithStack(err))
								return
							}
							hash = farmhash.Hash128(j)

							// In resume mode, check to see if the hash occurred in a previous run
							// (skip only contains successful requests from previous runs).
							if b.Resume {
								if _, skip := b.skip[hash]; skip {
									b.metrics.logSkip()
									continue
								}
							}
						}

						skipped = false

						b.workerChannel <- workDef{data: data, hash: hash}
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
	data map[string]string
	hash farmhash.Uint128
}
