package blaster

var doc = map[string]string{
	"Blaster":                    "Blaster provides the back-end blast: a simple tool for API load testing and batch jobs. Use the New function to create a Blaster with default values.",
	"Blaster.ChangeRate":         "ChangeRate changes the sending rate during execution.",
	"Blaster.Command":            "Command processes command line flags, loads the config and starts the blast run.",
	"Blaster.Exit":               "Exit cancels any goroutines that are still processing, and closes all files.",
	"Blaster.Headers":            "Headers sets the data headers. See Config.Headers for more details.",
	"Blaster.Initialise":         "Initialise configures the Blaster with config options in a provided Config",
	"Blaster.LoadConfig":         "LoadConfig parses command line flags and loads a config file from disk. A Config is returned which may be used with the Initialise method to complete configuration.",
	"Blaster.LoadLogs":           "LoadLogs loads the logs from a previous run, and stores successfully completed items so they can be skipped in the current run.",
	"Blaster.LogData":            "LogData sets the data fields to be logged. See Config.LogData for more details.",
	"Blaster.LogOutput":          "LogOutput sets the output fields to be logged. See Config.LogOutput for more details.",
	"Blaster.PayloadVariants":    "PayloadVariants sets the payload variants. See Config.PayloadVariants for more details.",
	"Blaster.PrintStatus":        "PrintStatus prints the status message to the output writer",
	"Blaster.Quiet":              "Quiet disables the status output.",
	"Blaster.Rate":               "Rate sets the initial sending rate. Do not change this during a run - use the ChangeRate method instead. See Config.Resume for more details.",
	"Blaster.ReadHeaders":        "ReadHeaders reads one row from the data source and stores that in Headers",
	"Blaster.RegisterWorkerType": "RegisterWorkerType registers a new worker function that can be referenced in config file by the worker-type string field.",
	"Blaster.Resume":             "Resume sets the resume option. See Config.Resume for more details.",
	"Blaster.SetData":            "SetData sets the CSV data source. If the provided io.Reader also satisfies io.Closer it will be\nclosed on exit.",
	"Blaster.SetInput":           "SetInput sets the rate adjustment reader, and allows testing rate adjustments. The Command method sets this to os.Stdin for interactive command line usage.",
	"Blaster.SetLog":             "SetLog sets the log output. If the provided writer also satisfies io.Closer, it will be closed on exit.",
	"Blaster.SetOutput":          "SetOutput sets the summary output writer, and allows the output to be redirected. The Command method sets this to os.Stdout for command line usage.",
	"Blaster.SetPayloadTemplate": "SetPayloadTemplate sets the payload template. See Config.PayloadTemplate for more details.",
	"Blaster.SetTimeout":         "SetTimeout sets the timeout. See Config.Timeout for more details.",
	"Blaster.SetWorker":          "SetWorker sets the worker creation function. See httpworker for a simple example.",
	"Blaster.SetWorkerTemplate":  "SetWorkerTemplate sets the worker template. See Config.WorkerTemplate for more details.",
	"Blaster.Start":              "Start starts the blast run without processing any config.",
	"Blaster.WorkerVariants":     "WorkerVariants sets the worker variants. See Config.WorkerVariants for more details.",
	"Blaster.Workers":            "Workers sets the number of workers. See Config.Workers for more details.",
	"Blaster.WriteLogHeaders":    "WriteLogHeaders writes the log headers to the log writer.",
	"Config":                     "Config provides all the standard config options. Use the Initialise method to configure with a provided Config.",
	"Config.Data":                "Data sets the the data file to load. If none is specified, the worker will be called repeatedly until interrupted (useful for load testing). Load a local file or stream directly from a GCS bucket with `gs://{bucket}/{filename}.csv`. Data should be in csv format, and if `headers` is not specified the first record will be used as the headers. If a newline character is found, this string is read as the data.",
	"Config.Headers":             "Headers sets the data file headers. If omitted, the first record of the csv data source is used. When setting this by command line flag or environment variable, use a json encoded string.",
	"Config.Log":                 "Log sets the filename of the log file to create / append to.",
	"Config.LogData":             "LogData sets an array of data fields to include in the output log. When setting this by command line flag or environment variable, use a json encoded string.",
	"Config.LogOutput":           "LogOutput sets an array of worker response fields to include in the output log. When setting this by command line flag or environment variable, use a json encoded string.",
	"Config.PayloadTemplate":     "PayloadTemplate sets the template that is rendered and passed to the worker `Send` method. When setting this by command line flag or environment variable, use a json encoded string.",
	"Config.PayloadVariants":     "PayloadVariants sets an array of maps that will cause each data item to be repeated with the provided data. When setting this by command line flag or environment variable, use a json encoded string.",
	"Config.Quiet":               "Quiet instructs the tool to prevent interactive features. No summary is printed during operation and the rate cannot be changed interactively.",
	"Config.Rate":                "Rate sets the initial rate in requests per second. Simply enter a new rate during execution to adjust this. (Default: 10 requests / second).",
	"Config.Resume":              "Resume instructs the tool to load the log file and skip previously successful items. Failed items will be retried.",
	"Config.Timeout":             "Timeout sets the deadline in the context passed to the worker. Workers must respect this the context cancellation. We exit with an error if any worker is processing for timeout + 1 second. (Default: 1 second).",
	"Config.WorkerTemplate":      "WorkerTemplate sets a template to render and pass to the worker `Start` or `Stop` methods if the worker satisfies the `Starter` or `Stopper` interfaces. Use with `worker-variants` to configure several workers differently to spread load. When setting this by command line flag or environment variable, use a json encoded string.",
	"Config.WorkerType":          "WorkerType sets the selected worker type. Register new worker types with the `RegisterWorkerType` method.",
	"Config.WorkerVariants":      "WorkerVariants sets an array of maps that will cause each worker to be initialised with different data. When setting this by command line flag or environment variable, use a json encoded string.",
	"Config.Workers":             "Workers sets the number of concurrent workers. (Default: 10 workers).",
	"DummyCloser":                "",
	"ExampleWorker":              "ExampleWorker facilitates code examples by satisfying the Worker, Starter and Stopper interfaces with provided functions.",
	"ExampleWorker.Send":         "Send satisfies the Worker interface.",
	"ExampleWorker.Start":        "Start satisfies the Starter interface.",
	"ExampleWorker.Stop":         "Stop satisfies the Stopper interface.",
	"LoggingWorker":              "",
	"LoggingWorkerLog":           "",
	"LoggingWriter":              "",
	"New":                        "New creates a new Blaster with defaults.",
	"Starter":                    "Starter and Stopper are interfaces a worker can optionally satisfy to provide initialization or finalization logic. See `httpworker` and `dummyworker` for simple examples.",
	"Stopper":                    "Stopper is an interface a worker can optionally satisfy to provide finalization logic.",
	"Summary":                    "Summary provides a simple summary of the completed run, and is returned by the Start method.",
	"ThreadSafeBuffer":           "",
	"Worker":                     "Worker is an interface that allows blast to easily be extended to support any protocol. See `main.go` for an example of how to build a command with your custom worker type.",
	"csvReader":                  "",
	"csvWriteFlusher":            "",
	"debug":                      "Set debug to true to print the number of active goroutines with every status.",
	"doc_go":                     "Package blaster provides the back-end for blast - a tool for load testing and sending api requests in bulk.\n\n Blast\n =====\n\n * Blast makes API requests at a fixed rate.\n * The number of concurrent workers is configurable.\n * The rate may be changed interactively during execution.\n * Blast is protocol agnostic, and adding a new worker type is trivial.\n * For load testing: random data can be added to API requests.\n * For batch jobs: CSV data can be loaded from local file or GCS bucket, and successful items from previous runs are skipped.\n\n Installation\n ============\n ## Mac\n ```\n brew tap dave/blast\n brew install blast\n ```\n\n ## Linux\n See the [releases page](https://github.com/dave/blast/releases)\n\n ## From source\n ```\n go get -u github.com/dave/blast\n ```\n\n Examples\n ========\n Using the dummy worker to send at 20,000 requests per second (the dummy worker returns after a random wait, and occasionally returns errors):\n ```\n blast --rate=20000 --workers=1000 --worker-type=\"dummy\" --worker-template='{\"min\":25,\"max\":50}'\n ```\n\n Using the http worker to request Google's homepage at one request per second (warning: this is making real http requests - don't turn the rate up!):\n ```\n blast --rate=1 --worker-type=\"http\" --payload-template='{\"method\":\"GET\",\"url\":\"http://www.google.com/\"}'\n ```\n\n Status\n ======\n\n Blast prints a summary every ten seconds. While blast is running, you can hit enter for an updated\n summary, or enter a number to change the sending rate. Each time you change the rate a new column\n of metrics is created. If the worker returns a field named `status` in it's response, the values\n are summarised as rows.\n\n Here's an example of the output:\n\n ```\n Metrics\n =======\n Concurrency:      1999 / 2000 workers in use\n\n Desired rate:     (all)        10000        1000         100\n Actual rate:      2112         5354         989          100\n Avg concurrency:  1733         1976         367          37\n Duration:         00:40        00:12        00:14        00:12\n\n Total\n -----\n Started:          84525        69004        14249        1272\n Finished:         82525        67004        14249        1272\n Mean:             376.0 ms     374.8 ms     379.3 ms     377.9 ms\n 95th:             491.1 ms     488.1 ms     488.2 ms     489.6 ms\n\n 200\n ---\n Count:            79208 (96%)  64320 (96%)  13663 (96%)  1225 (96%)\n Mean:             376.2 ms     381.9 ms     374.7 ms     378.1 ms\n 95th:             487.6 ms     489.0 ms     487.2 ms     490.5 ms\n\n 404\n ---\n Count:            2467 (3%)    2002 (3%)    430 (3%)     35 (3%)\n Mean:             371.4 ms     371.0 ms     377.2 ms     358.9 ms\n 95th:             487.1 ms     487.1 ms     486.0 ms     480.4 ms\n\n 500\n ---\n Count:            853 (1%)     685 (1%)     156 (1%)     12 (1%)\n Mean:             371.2 ms     370.4 ms     374.5 ms     374.3 ms\n 95th:             487.6 ms     487.1 ms     488.2 ms     466.3 ms\n\n Current rate is 10000 requests / second. Enter a new rate or press enter to view status.\n\n Rate?\n ```\n\n Config\n ======\n Blast is configured by config file, command line flags or environment variables. The `--config` flag specifies the config file to load, and can be `json`, `yaml`, `toml` or anything else that [viper](https://github.com/spf13/viper) can read. If the config flag is omitted, blast searches for `blast-config.xxx` in the current directory, `$HOME/.config/blast/` and `/etc/blast/`.\n\n Environment variables and command line flags override config file options. Environment variables are upper case and prefixed with \"BLAST\" e.g. `BLAST_PAYLOAD_TEMPLATE`.\n\n Templates\n =========\n The `payload-template` and `worker-template` options accept values that are rendered using the Go text/template system. Variables of the form `{{ .name }}` or `{{ \"name\" }}` are replaced with data.\n\n Additionally, several simple functions are available to inject random data which is useful in load testing scenarios:\n\n * `{{ rand_int -5 5 }}` - a random integer between -5 and 5.\n * `{{ rand_float -5 5 }}` - a random float between -5 and 5.\n * `{{ rand_string 10 }}` - a random string, length 10.",
	"logRecord":                  "",
	"mapR":                       "",
	"metricsDef":                 "",
	"metricsItem":                "",
	"metricsSegment":             "",
	"native":                     "",
	"nativeR":                    "",
	"renderer":                   "",
	"sliceR":                     "",
	"templateR":                  "",
	"threadSafeWriter":           "",
	"threadSafeWriter.Write":     "Write writes to the underlying writer in a thread safe manner.",
	"workDef":                    "",
}
