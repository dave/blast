[![Build Status](https://travis-ci.org/dave/blast.svg?branch=master)](https://travis-ci.org/dave/blast) [![Go Report Card](https://goreportcard.com/badge/github.com/dave/blast)](https://goreportcard.com/report/github.com/dave/blast) [![codecov](https://codecov.io/gh/dave/blast/branch/master/graph/badge.svg)](https://codecov.io/gh/dave/blast)




 Blast
 =====

 * Blast makes API requests at a fixed rate.
 * The number of concurrent workers is configurable.
 * The rate may be changed interactively during execution.
 * Blast is protocol agnostic, and adding a new worker type is trivial.
 * For load testing: random data can be added to API requests.
 * For batch jobs: CSV data can be loaded from local file or GCS bucket, and successful items from previous runs are skipped.

 Installation
 ============
 ## Mac
 ```
 brew tap dave/blast
 brew install blast
 ```

 ## Linux
 See the [releases page](https://github.com/dave/blast/releases)

 ## From source
 ```
 go get -u github.com/dave/blast
 ```

 Examples
 ========
 Using the dummy worker to send at 20,000 requests per second (the dummy worker returns after a random wait, and occasionally returns errors):
 ```
 blast --rate=20000 --workers=1000 --worker-type="dummy" --worker-template='{"min":25,"max":50}'
 ```

 Using the http worker to request Google's homepage at one request per second (warning: this is making real http requests - don't turn the rate up!):
 ```
 blast --rate=1 --worker-type="http" --payload-template='{"method":"GET","url":"http://www.google.com/"}'
 ```

 Status
 ======

 Blast prints a summary every ten seconds. While blast is running, you can hit enter for an updated
 summary, or enter a number to change the sending rate. Each time you change the rate a new column
 of metrics is created. If the worker returns a field named `status` in it's response, the values
 are summarised as rows.

 Here's an example of the output:

 ```
 Metrics
 =======
 Concurrency:      1999 / 2000 workers in use

 Desired rate:     (all)        10000        1000         100
 Actual rate:      2112         5354         989          100
 Avg concurrency:  1733         1976         367          37
 Duration:         00:40        00:12        00:14        00:12

 Total
 -----
 Started:          84525        69004        14249        1272
 Finished:         82525        67004        14249        1272
 Mean:             376.0 ms     374.8 ms     379.3 ms     377.9 ms
 95th:             491.1 ms     488.1 ms     488.2 ms     489.6 ms

 200
 ---
 Count:            79208 (96%)  64320 (96%)  13663 (96%)  1225 (96%)
 Mean:             376.2 ms     381.9 ms     374.7 ms     378.1 ms
 95th:             487.6 ms     489.0 ms     487.2 ms     490.5 ms

 404
 ---
 Count:            2467 (3%)    2002 (3%)    430 (3%)     35 (3%)
 Mean:             371.4 ms     371.0 ms     377.2 ms     358.9 ms
 95th:             487.1 ms     487.1 ms     486.0 ms     480.4 ms

 500
 ---
 Count:            853 (1%)     685 (1%)     156 (1%)     12 (1%)
 Mean:             371.2 ms     370.4 ms     374.5 ms     374.3 ms
 95th:             487.6 ms     487.1 ms     488.2 ms     466.3 ms

 Current rate is 10000 requests / second. Enter a new rate or press enter to view status.

 Rate?
 ```

 Config
 ======
 Blast is configured by config file, command line flags or environment variables. The `--config` flag specifies the config file to load, and can be `json`, `yaml`, `toml` or anything else that [viper](https://github.com/spf13/viper) can read. If the config flag is omitted, blast searches for `blast-config.xxx` in the current directory, `$HOME/.config/blast/` and `/etc/blast/`.

 Environment variables and command line flags override config file options. Environment variables are upper case and prefixed with "BLAST" e.g. `BLAST_PAYLOAD_TEMPLATE`.

 Templates
 =========
 The `payload-template` and `worker-template` options accept values that are rendered using the Go text/template system. Variables of the form `{{ .name }}` or `{{ "name" }}` are replaced with data.

 Additionally, several simple functions are available to inject random data which is useful in load testing scenarios:

 * `{{ rand_int -5 5 }}` - a random integer between -5 and 5.
 * `{{ rand_float -5 5 }}` - a random float between -5 and 5.
 * `{{ rand_string 10 }}` - a random string, length 10.


Workers
=======

Worker is an interface that allows blast to easily be extended to support any protocol. See `main.go` for an example of how to build a command with your custom worker type.

Starter and Stopper are interfaces a worker can optionally satisfy to provide initialization or finalization logic. See `httpworker` and `dummyworker` for simple examples. 

Examples
========

For load testing:

```yaml
rate: 20000
workers: 1000
worker-type: "dummy"
payload-template:
  method: "POST"
  path: "/foo/?id={{ rand_int 1 10000000 }}"
worker-template:
  print: false
  base: "https://{{ .region }}.my-api.com"
  min: 10
  max: 20
worker-variants:
  - region: "europe-west1"
  - region: "us-east1"
```

For bulk API tasks:

```yaml
# data would usually be the filename of a local CSV file, or an object in a GCS bucket. However, 
# for the purposes of this example, a CSV fragment is also accepted.
data: |
  user_name,action
  dave,subscribe
  john,subscribe
  pete,unsubscribe
  jimmy,unsubscribe
resume: true
log: "out.log"
rate: 100
workers: 20
worker-type: "dummy"
payload-template: 
  method: "POST"
  path: "/{{ .user_name }}/{{ .action }}/{{ .type }}/"
worker-template:  
  print: true
  base: "https://{{ .region }}.my-api.com"
  min: 250
  max: 500
payload-variants: 
  - type: "email"
  - type: "phone"
worker-variants: 
  - region: "europe-west1"
  - region: "us-east1"
log-data:
  - "user_name"
  - "action"
log-output: 
  - "status"
```

Configuration options
=====================

data
----
Data sets the the data file to load. If none is specified, the worker will be called repeatedly until interrupted (useful for load testing). Load a local file or stream directly from a GCS bucket with `gs://{bucket}/{filename}.csv`. Data should be in csv format, and if `headers` is not specified the first record will be used as the headers. If a newline character is found, this string is read as the data.

log
---
Log sets the filename of the log file to create / append to.

resume
------
Resume instructs the tool to load the log file and skip previously successful items. Failed items will be retried.

rate
----
Rate sets the initial rate in requests per second. Simply enter a new rate during execution to adjust this. (Default: 10 requests / second).

workers
-------
Workers sets the number of concurrent workers. (Default: 10 workers).

worker-type
-----------
WorkerType sets the selected worker type. Register new worker types with the `RegisterWorkerType` method.

payload-template
----------------
PayloadTemplate sets the template that is rendered and passed to the worker `Send` method. When setting this by command line flag or environment variable, use a json encoded string.


Advanced configuration options
==============================

timeout
-------
Timeout sets the deadline in the context passed to the worker. Workers must respect this the context cancellation. We exit with an error if any worker is processing for timeout + 1 second. (Default: 1 second). 

headers
-------
Headers sets the data file headers. If omitted, the first record of the csv data source is used. When setting this by command line flag or environment variable, use a json encoded string.

log-data
--------
LogData sets an array of data fields to include in the output log. When setting this by command line flag or environment variable, use a json encoded string.

log-output
----------
LogOutput sets an array of worker response fields to include in the output log. When setting this by command line flag or environment variable, use a json encoded string.

worker-template
---------------
WorkerTemplate sets a template to render and pass to the worker `Start` or `Stop` methods if the worker satisfies the `Starter` or `Stopper` interfaces. Use with `worker-variants` to configure several workers differently to spread load. When setting this by command line flag or environment variable, use a json encoded string.

worker-variants
---------------
WorkerVariants sets an array of maps that will cause each worker to be initialised with different data. When setting this by command line flag or environment variable, use a json encoded string.

payload-variants
----------------
PayloadVariants sets an array of maps that will cause each data item to be repeated with the provided data. When setting this by command line flag or environment variable, use a json encoded string.

Control by code
===============
The blaster package may be used to start blast from code without using the command. Here's a some 
examples of usage:

```go
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
```

```go
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
```
 
To do
=====  
- [ ] Adjust rate automatically in response to latency? PID controller?  
- [ ] Only use part of file: part i of j parts  
