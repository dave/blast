package main

import (
	"context"
	"log"

	"fmt"

	"github.com/dave/blast"
	"github.com/dave/blast/httpworker"
)

func main() {
	ctx := context.Background()

	b := blast.New(ctx)

	b.RegisterWorkerType("http", httpworker.New)

	if err := b.Start(ctx); err != nil {
		log.Fatal(fmt.Printf("%+v", err))
	}
}
