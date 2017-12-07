package main

// This generate command uses github.com/dave/rebecca to build the documentation from the
// README.md.tpl template, and create a dump of doc comments in doc-gen.go.
//go:generate becca -package=github.com/dave/blast/blaster -literals=blaster/doc-gen.go

import (
	"context"
	"log"

	"fmt"

	"os"

	"github.com/dave/blast/blaster"
	"github.com/dave/blast/dummyworker"
	"github.com/dave/blast/gcsworker"
	"github.com/dave/blast/httpworker"
)

// Set debug to true to dump full stack info on every error.
const debug = false

func main() {

	// notest

	ctx, cancel := context.WithCancel(context.Background())

	b := blaster.New(ctx, cancel)
	defer b.Exit()

	b.RegisterWorkerType("dummy", dummyworker.New)
	b.RegisterWorkerType("http", httpworker.New)
	b.RegisterWorkerType("gcs", gcsworker.New)

	if err := b.Command(ctx); err != nil {
		if debug {
			log.Fatal(fmt.Printf("%+v", err))
		} else {
			fmt.Println(err)
			os.Exit(1)
		}
	}
}
