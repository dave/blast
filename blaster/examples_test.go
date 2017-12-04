package blaster_test

import (
	"context"

	"fmt"

	"strings"

	"time"

	"sync"

	"github.com/dave/blast/blaster"
)

func ExampleBlaster_Start_batchJob() {
	ctx, cancel := context.WithCancel(context.Background())
	b := blaster.New(ctx, cancel)
	defer b.Exit()
	b.SetWorker(func() blaster.Worker {
		return &blaster.ExampleWorker{
			SendFunc: func(ctx context.Context, self *blaster.ExampleWorker, in map[string]interface{}) (map[string]interface{}, error) {
				return map[string]interface{}{"status": 200}, nil
			},
		}
	})
	b.Headers = []string{"header"}
	b.SetData(strings.NewReader("foo\nbar"))
	stats, err := b.Start(ctx)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Printf("Success == 2: %v\n", stats.All.Summary.Success == 2)
	fmt.Printf("Fail == 0: %v", stats.All.Summary.Fail == 0)
	// Output:
	// Success == 2: true
	// Fail == 0: true
}

func ExampleBlaster_Start_loadTest() {
	ctx, cancel := context.WithCancel(context.Background())
	b := blaster.New(ctx, cancel)
	defer b.Exit()
	b.SetWorker(func() blaster.Worker {
		return &blaster.ExampleWorker{
			SendFunc: func(ctx context.Context, self *blaster.ExampleWorker, in map[string]interface{}) (map[string]interface{}, error) {
				return map[string]interface{}{"status": 200}, nil
			},
		}
	})
	b.Rate = 1000
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		stats, err := b.Start(ctx)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		fmt.Printf("Success > 10: %v\n", stats.All.Summary.Success > 10)
		fmt.Printf("Fail == 0: %v", stats.All.Summary.Fail == 0)
		wg.Done()
	}()
	<-time.After(time.Millisecond * 100)
	b.Exit()
	wg.Wait()
	// Output:
	// Success > 10: true
	// Fail == 0: true
}
