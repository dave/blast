package main

import (
	"context"
	"log"
	"os"

	"fmt"

	"os/signal"

	"github.com/dave/blast"
	"github.com/dave/blast/httpworker"
)

func main() {
	ctx := context.Background()

	// trap Ctrl+C and call cancel on the context
	ctx, cancel := context.WithCancel(ctx)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	defer func() {
		signal.Stop(c)
		cancel()
	}()
	go func() {
		select {
		case <-c:
			cancel()
		case <-ctx.Done():
		}
	}()

	b := blast.New(cancel)

	b.RegisterWorkerType("http", httpworker.New)

	if err := b.Start(ctx); err != nil {
		log.Fatal(fmt.Printf("%+v", err))
	}
}
