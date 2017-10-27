package main

import (
	"context"
	"log"

	"fmt"

	"os"

	"github.com/dave/blast"
	"github.com/dave/blast/dummyworker"
	"github.com/dave/blast/httpworker"
)

const DEBUG = false

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	b := blast.New(ctx, cancel)
	defer b.Exit()

	b.RegisterWorkerType("dummy", dummyworker.New)
	b.RegisterWorkerType("http", httpworker.New)

	if err := b.Start(ctx); err != nil {
		if DEBUG {
			log.Fatal(fmt.Printf("%+v", err))
		} else {
			fmt.Println(err)
			os.Exit(1)
		}
	}
}
