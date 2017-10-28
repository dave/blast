package blast

import (
	"bufio"
	"context"
	"fmt"
	"strconv"
	"strings"

	"io"

	"github.com/pkg/errors"
)

func (b *Blaster) startRateLoop(ctx context.Context) {

	b.mainWait.Add(1)

	readString := func() chan string {
		c := make(chan string)
		go func() {
			reader := bufio.NewReader(b.rateInputReader)
			text, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					return
				}
				b.error(errors.WithStack(err))
				return
			}
			c <- text
		}()
		return c
	}

	go func() {
		defer b.mainWait.Done()
		defer fmt.Fprintln(b.out, "Exiting rate loop")
		for {
			select {
			case <-ctx.Done():
				return
			case <-b.dataFinishedChannel:
				return
			case s := <-readString():
				s = strings.TrimSpace(s)
				if s == "" {
					b.printStatus(false)
					continue
				}
				f, err := strconv.ParseFloat(s, 64)
				if err != nil {
					b.error(errors.WithStack(err))
					return
				}
				b.changeRateChannel <- f
			}
		}
	}()
}
