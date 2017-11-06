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

const DEBUG = false

type Blaster struct {
	Quiet           bool
	Resume          bool
	Rate            float64
	Workers         int
	LogData         []string
	LogOutput       []string
	Headers         []string
	PayloadVariants []map[string]string
	WorkerVariants  []map[string]string

	workerFunc func() Worker

	viper *viper.Viper

	softTimeout time.Duration
	hardTimeout time.Duration
	skip        map[farmhash.Uint128]struct{}

	logWriter  CsvWriteFlusher
	logCloser  io.Closer
	outWriter  io.Writer
	outCloser  io.Closer
	dataReader CsvReader
	dataCloser io.Closer

	inputReader io.Reader

	cancel context.CancelFunc

	payloadRenderer renderer
	workerRenderer  renderer

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

	errorsIgnored uint64
	metrics       *metricsDef
	err           error
}

func (b *Blaster) SetTimeout(timeout time.Duration) {
	b.softTimeout = timeout
	b.hardTimeout = timeout + time.Second
}

func (b *Blaster) SetWorker(wf func() Worker) {
	b.workerFunc = wf
}

func (b *Blaster) SetPayloadTemplate(t map[string]interface{}) error {
	var err error
	if b.payloadRenderer, err = parseRenderer(t); err != nil {
		return err
	}
	return nil
}

func (b *Blaster) SetWorkerTemplate(t map[string]interface{}) error {
	var err error
	if b.workerRenderer, err = parseRenderer(t); err != nil {
		return err
	}
	return nil
}

func (b *Blaster) SetInput(r io.Reader) {
	b.inputReader = r
}

// SetOutput sets the output, and allows the summary output to be redirected. The Command method sets this to os.Stdout.
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
		mainChannel:            make(chan struct{}),
		workerChannel:          make(chan workDef),
		Rate:                   10,
		Workers:                10,
		softTimeout:            time.Second,
		hardTimeout:            time.Second * 2,
		WorkerVariants:         []map[string]string{{}},
		PayloadVariants:        []map[string]string{{}},
	}
	b.metrics = newMetricsDef(b)

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

type Summary struct {
	Success int64
	Fail    int64
}

// Command process command line flags, loads the config and starts the blast tool
func (b *Blaster) Command(ctx context.Context) error {

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

// Command starts the blast tool
func (b *Blaster) Start(ctx context.Context) (Summary, error) {

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

	if b.Rate <= 0 {
		panic("Must specify rate!")
	}

	err := b.start(ctx)

	summary := Summary{
		Success: b.metrics.all.total.success.Count(),
		Fail:    b.metrics.all.total.fail.Count(),
	}

	return summary, err
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

	b.printStatus(true)
	return nil
}

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

func (t *threadSafeWriter) Write(p []byte) (n int, err error) {
	t.m.Lock()
	defer t.m.Unlock()
	return t.w.Write(p)
}

type CsvReader interface {
	Read() (record []string, err error)
}

type CsvWriteFlusher interface {
	Write(record []string) error
	Flush()
}
