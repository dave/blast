package blaster

import (
	"bufio"
	"context"
	"strconv"
	"strings"

	"io"

	"github.com/pkg/errors"
)

func (b *Blaster) startRateLoop(ctx context.Context) {

	if b.inputReader == nil {
		return
	}

	b.mainWait.Add(1)

	readString := func() chan string {
		c := make(chan string)
		go func() {
			reader := bufio.NewReader(b.inputReader)
			text, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					return
				}
				// notest
				b.error(errors.WithStack(err))
				return
			}
			c <- text
		}()
		return c
	}

	go func() {
		defer b.mainWait.Done()
		defer b.println("Exiting rate loop")
		for {
			select {
			case <-ctx.Done():
				return
			case <-b.dataFinishedChannel:
				return
			case s := <-readString():
				s = strings.TrimSpace(s)
				if s == "" {
					// notest
					b.printStatus(false)
					continue
				}
				f, err := strconv.ParseFloat(s, 64)
				if err != nil {
					// notest
					b.error(errors.WithStack(err))
					return
				}
				b.changeRateChannel <- f
			}
		}
	}()
}
