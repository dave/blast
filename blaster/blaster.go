package blaster

import (
	"context"
	"io"
	"os"
	"os/signal"
	"sync"

	"time"

	"sync/atomic"

	"github.com/leemcloughlin/gofarmhash"
	"github.com/spf13/viper"
)

// Set debug to true to print the number of active goroutines with every status.
const debug = false

// Blaster provides the back-end blast: a simple tool for API load testing and batch jobs. Use the New function to create a Blaster with default values.
type Blaster struct {
	// Quiet disables the status output.
	Quiet bool

	// Resume sets the resume option. See Config.Resume for more details.
	Resume bool

	// Rate sets the initial sending rate. Do not change this during a run - use the ChangeRate method instead. See Config.Resume for more details.
	Rate float64

	// Workers sets the number of workers. See Config.Workers for more details.
	Workers int

	// LogData sets the data fields to be logged. See Config.LogData for more details.
	LogData []string

	// LogOutput sets the output fields to be logged. See Config.LogOutput for more details.
	LogOutput []string

	// Headers sets the data headers. See Config.Headers for more details.
	Headers []string

	// PayloadVariants sets the payload variants. See Config.PayloadVariants for more details.
	PayloadVariants []map[string]string

	// WorkerVariants sets the worker variants. See Config.WorkerVariants for more details.
	WorkerVariants []map[string]string

	workerFunc func() Worker

	viper *viper.Viper

	softTimeout time.Duration
	hardTimeout time.Duration
	skip        map[farmhash.Uint128]struct{}

	logWriter  csvWriteFlusher
	logCloser  io.Closer
	outWriter  io.Writer
	outCloser  io.Closer
	dataReader csvReader
	dataCloser io.Closer

	inputReader io.Reader

	cancel context.CancelFunc

	payloadRenderer renderer
	workerRenderer  renderer

	mainChannel            chan int
	errorChannel           chan error
	workerChannel          chan workDef
	logChannel             chan logRecord
	dataFinishedChannel    chan struct{}
	workersFinishedChannel chan struct{}
	itemFinishedChannel    chan struct{}
	changeRateChannel      chan float64
	signalChannel          chan os.Signal

	mainWait   *sync.WaitGroup
	workerWait *sync.WaitGroup

	workerTypes map[string]func() Worker

	errorsIgnored uint64
	metrics       *metricsDef
	err           error
	gcs           opener
}

// SetTimeout sets the timeout. See Config.Timeout for more details.
func (b *Blaster) SetTimeout(timeout time.Duration) {
	b.softTimeout = timeout
	b.hardTimeout = timeout + time.Second
}

// SetWorker sets the worker creation function. See httpworker for a simple example.
func (b *Blaster) SetWorker(wf func() Worker) {
	b.workerFunc = wf
}

// SetPayloadTemplate sets the payload template. See Config.PayloadTemplate for more details.
func (b *Blaster) SetPayloadTemplate(t map[string]interface{}) error {
	var err error
	if b.payloadRenderer, err = parseRenderer(t); err != nil {
		return err
	}
	return nil
}

// SetWorkerTemplate sets the worker template. See Config.WorkerTemplate for more details.
func (b *Blaster) SetWorkerTemplate(t map[string]interface{}) error {
	var err error
	if b.workerRenderer, err = parseRenderer(t); err != nil {
		return err
	}
	return nil
}

// SetInput sets the rate adjustment reader, and allows testing rate adjustments. The Command method sets this to os.Stdin for interactive command line usage.
func (b *Blaster) SetInput(r io.Reader) {
	b.inputReader = r
}

// SetOutput sets the summary output writer, and allows the output to be redirected. The Command method sets this to os.Stdout for command line usage.
func (b *Blaster) SetOutput(w io.Writer) {
	if w == nil {
		b.outWriter = nil
		b.outCloser = nil
		return
	}
	b.outWriter = newThreadSafeWriter(w)
	if c, ok := w.(io.Closer); ok {
		b.outCloser = c
	} else {
		b.outCloser = nil
	}
}

// ChangeRate changes the sending rate during execution.
func (b *Blaster) ChangeRate(rate float64) {
	b.changeRateChannel <- rate
}

// New creates a new Blaster with defaults.
func New(ctx context.Context, cancel context.CancelFunc) *Blaster {

	b := &Blaster{
		viper:                  viper.New(),
		cancel:                 cancel,
		mainWait:               new(sync.WaitGroup),
		workerWait:             new(sync.WaitGroup),
		workerTypes:            make(map[string]func() Worker),
		skip:                   make(map[farmhash.Uint128]struct{}),
		dataFinishedChannel:    make(chan struct{}),
		workersFinishedChannel: make(chan struct{}),
		changeRateChannel:      make(chan float64, 1),
		errorChannel:           make(chan error),
		logChannel:             make(chan logRecord),
		mainChannel:            make(chan int),
		workerChannel:          make(chan workDef),
		Rate:                   10,
		Workers:                10,
		softTimeout:            time.Second,
		hardTimeout:            time.Second * 2,
		WorkerVariants:         []map[string]string{{}},
		PayloadVariants:        []map[string]string{{}},
		gcs:                    googleCloudOpener{},
	}
	b.metrics = newMetricsDef(b)

	// trap Ctrl+C and call cancel on the context
	b.signalChannel = make(chan os.Signal, 1)
	signal.Notify(b.signalChannel, os.Interrupt)
	go func() {
		select {
		case <-b.signalChannel:
			// notest
			b.cancel()
		case <-ctx.Done():
		}
	}()

	return b
}

// Exit cancels any goroutines that are still processing, and closes all files.
func (b *Blaster) Exit() {
	if b.logWriter != nil {
		b.logWriter.Flush()
	}
	if b.logCloser != nil {
		_ = b.logCloser.Close() // ignore error
	}
	if b.outCloser != nil {
		_ = b.outCloser.Close() // ignore error
	}
	if b.dataCloser != nil {
		_ = b.dataCloser.Close() // ignore error
	}
	signal.Stop(b.signalChannel)
	b.cancel()
}

// Command processes command line flags, loads the config and starts the blast run.
func (b *Blaster) Command(ctx context.Context) error {

	// notest

	c, err := b.LoadConfig()
	if err != nil {
		return err
	}

	if err := b.Initialise(ctx, c); err != nil {
		return err
	}

	b.SetOutput(os.Stdout)

	if !b.Quiet {
		b.SetInput(os.Stdin)
	}

	_, err = b.Start(ctx)

	return err
}

// Start starts the blast run without processing any config.
func (b *Blaster) Start(ctx context.Context) (Stats, error) {

	if b.dataReader == nil && b.Resume {
		panic("In resume mode, data must be specified!")
	}

	if b.logWriter == nil && b.Resume {
		panic("In resume mode, log must be specified!")
	}

	if b.logWriter == nil && (len(b.LogOutput) > 0 || len(b.LogData) > 0) {
		panic("If log-output or log-data is specified, log file must be specified!")
	}

	if b.workerFunc == nil {
		panic("Must specify worker-type!")
	}

	if b.Workers < 1 {
		panic("Must specify workers!")
	}

	if b.Rate < 0 {
		panic("Rate must not be negative!")
	}

	err := b.start(ctx)

	return b.Stats(), err
}

// Stats returns a snapshot of the metrics (as is printed during interactive execution).
func (b *Blaster) Stats() Stats {
	return b.metrics.stats()
}

func (b *Blaster) start(ctx context.Context) error {

	b.metrics.addSegment(b.Rate)

	b.startTickerLoop(ctx)
	b.startMainLoop(ctx)
	b.startErrorLoop(ctx)
	b.startWorkers(ctx)

	b.startLogLoop(ctx)
	b.startStatusLoop(ctx)
	b.startRateLoop(ctx)
	b.printRatePrompt()

	// wait for cancel or finished
	select {
	case <-ctx.Done():
	case <-b.dataFinishedChannel:
	}

	b.println("Waiting for workers to finish...")
	b.workerWait.Wait()
	b.println("All workers finished.")

	// signal to log and error loop that it's tine to exit
	close(b.workersFinishedChannel)

	b.println("Waiting for processes to finish...")
	b.mainWait.Wait()
	b.println("All processes finished.")

	if b.err != nil {
		b.println("")
		errorsIgnored := atomic.LoadUint64(&b.errorsIgnored)
		if errorsIgnored > 0 {
			b.printf("%d errors were ignored because we were already exiting with an error.\n", errorsIgnored)
		}
		b.printf("Fatal error: %v\n", b.err)
		return b.err
	}
	b.println("")
	b.printStatus(true)
	return nil
}

// RegisterWorkerType registers a new worker function that can be referenced in config file by the worker-type string field.
func (b *Blaster) RegisterWorkerType(key string, workerFunc func() Worker) {
	b.workerTypes[key] = workerFunc
}

// Worker is an interface that allows blast to easily be extended to support any protocol. See `main.go` for an example of how to build a command with your custom worker type.
type Worker interface {
	Send(ctx context.Context, payload map[string]interface{}) (response map[string]interface{}, err error)
}

// Starter and Stopper are interfaces a worker can optionally satisfy to provide initialization or finalization logic. See `httpworker` and `dummyworker` for simple examples.
type Starter interface {
	Start(ctx context.Context, payload map[string]interface{}) error
}

// Stopper is an interface a worker can optionally satisfy to provide finalization logic.
type Stopper interface {
	Stop(ctx context.Context, payload map[string]interface{}) error
}

func newThreadSafeWriter(w io.Writer) *threadSafeWriter {
	return &threadSafeWriter{
		w: w,
	}
}

type threadSafeWriter struct {
	w io.Writer
	m sync.Mutex
}

// Write writes to the underlying writer in a thread safe manner.
func (t *threadSafeWriter) Write(p []byte) (n int, err error) {
	t.m.Lock()
	defer t.m.Unlock()
	return t.w.Write(p)
}

type csvReader interface {
	Read() (record []string, err error)
}

type csvWriteFlusher interface {
	Write(record []string) error
	Flush()
}
