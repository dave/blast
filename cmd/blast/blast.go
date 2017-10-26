package main

import (
	"context"
	"log"

	"fmt"

	"github.com/dave/blast"
	"github.com/dave/blast/dummyworker"
	"github.com/dave/blast/httpworker"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	b := blast.New(ctx, cancel)
	defer b.Exit()

	b.RegisterWorkerType("dummy", dummyworker.New)
	b.RegisterWorkerType("http", httpworker.New)

	if err := b.Start(ctx); err != nil {
		log.Fatal(fmt.Printf("%+v", err))
	}
}
