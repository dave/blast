package blast

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"time"
)

const DEBUG = false
const INSTANT_COUNT = 100

type Blaster struct {
	config         *configDef
	rate           float64
	skip           map[string]struct{}
	dataReadCloser io.ReadCloser
	dataReader     *csv.Reader
	dataHeaders    []string
	logFile        *os.File
	logWriter      *csv.Writer
	cancel         context.CancelFunc

	mainChannel            chan struct{}
	errorChannel           chan error
	workerChannel          chan workDef
	logChannel             chan logRecord
	dataFinishedChannel    chan struct{}
	workersFinishedChannel chan struct{}
	changeRateChannel      chan float64
	signalChannel          chan os.Signal

	mainWait   *sync.WaitGroup
	workerWait *sync.WaitGroup

	workerTypes map[string]func() Worker

	stats statsDef
}

type statsDef struct {
	itemsStarted  uint64
	itemsFinished uint64
	itemsSkipped  uint64
	itemsSuccess  uint64
	itemsFailed   uint64

	requestsStarted         uint64
	requestsFinished        uint64
	requestsSuccess         uint64
	requestsSuccessDuration uint64
	requestsDurationQueue   *FiloQueue

	workersBusy  int64
	ticksSkipped uint64
}

func New(ctx context.Context, cancel context.CancelFunc) *Blaster {

	b := &Blaster{
		cancel:                 cancel,
		mainWait:               new(sync.WaitGroup),
		workerWait:             new(sync.WaitGroup),
		workerTypes:            make(map[string]func() Worker),
		dataFinishedChannel:    make(chan struct{}),
		workersFinishedChannel: make(chan struct{}),
		changeRateChannel:      make(chan float64, 1),
		stats: statsDef{
			requestsDurationQueue: &FiloQueue{},
		},
	}

	// trap Ctrl+C and call cancel on the context
	b.signalChannel = make(chan os.Signal, 1)
	signal.Notify(b.signalChannel, os.Interrupt)
	go func() {
		select {
		case <-b.signalChannel:
			b.cancel()
		case <-ctx.Done():
		}
	}()

	return b
}

func (b *Blaster) Exit() {
	signal.Stop(b.signalChannel)
	b.cancel()
}

func (b *Blaster) Start(ctx context.Context) error {

	if err := b.loadConfig(); err != nil {
		return err
	}

	if err := b.openDataFile(ctx); err != nil {
		return err
	}
	defer b.closeDataFile()

	if err := b.openLogAndInit(); err != nil {
		return err
	}
	defer b.flushAndCloseLog()

	b.startTickerLoop(ctx)
	b.startMainLoop(ctx)
	b.startErrorLoop(ctx)
	b.startWorkers(ctx)
	b.startLogLoop(ctx)
	b.startStatusLoop(ctx)
	b.startRateLoop(ctx)

	b.printStatus()

	// wait for cancel or finished
	select {
	case <-ctx.Done():
	case <-b.dataFinishedChannel:
	}

	fmt.Println("Waiting for workers to finish...")
	b.workerWait.Wait()

	// signal to log and error loop that it's tine to exit
	close(b.workersFinishedChannel)

	fmt.Println("Waiting for processes to finish...")
	b.mainWait.Wait()

	b.printStatus()

	return nil
}

func (b *Blaster) RegisterWorkerType(key string, workerFunc func() Worker) {
	b.workerTypes[key] = workerFunc
}

type Worker interface {
	Send(ctx context.Context, payload map[string]interface{}) error
}

type Starter interface {
	Start(ctx context.Context, payload map[string]interface{}) error
}

type Stopper interface {
	Stop(ctx context.Context, payload map[string]interface{}) error
}

func init() {
	if DEBUG {
		go func() {
			// debug to see if goroutines aren't being closed...
			ticker := time.NewTicker(time.Millisecond * 200)
			for range ticker.C {
				fmt.Println("runtime.NumGoroutine(): ", runtime.NumGoroutine())
			}
		}()
	}
}
