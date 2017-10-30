Blast
=====

* Blast makes API requests at a fixed rate, based on input data from a CSV file.      
* The number of concurrent workers is configurable.  
* The rate may be changed interactively during execution.  
* Blast is protocol agnostic, and adding a new worker type is trivial.  
* With the `resume` option, successful items from previous runs are skipped.  

Installation
============
```
go get -u github.com/dave/blast/cmd/blast
```

Usage
=====
```
blast [options]
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

Blast is configured by config file, command line flags or environment variables. The `--config` 
flag specifies the config file to load, and can be `json`, `yaml`, `toml` or anything else that 
[viper](https://github.com/spf13/viper) can read. If the config flag is omitted, blast searches for 
`blast-config.json|yaml|toml|etc` in the current directory, `~/.config/blast/` and `/etc/blast/`. 
Environment variables and command line flags override config file options.

See [blast-config.yaml](https://github.com/dave/blast/blob/master/blast-config.yaml) for a simple 
annotated example. See [test-config-load-test.yaml](https://github.com/dave/blast/blob/master/test-config-load-test.yaml)
for a load-testing specific example.

Templates
=========

The `payload-template` and `worker-template` options accept values that are rendered using a simple
template syntax. Variables of the form `{{name}}` are replaced with data. Note: No whitespace is 
allowed surrounding the variable name. 

Required configuration options
==============================

data
----
The data file to load. Stream directly from a GCS bucket with `gs://{bucket}/{filename}.csv`. 
Data should be in CSV format with a header row. This may be set with the `BLAST_DATA` environment 
variable or the `--data` flag.

log
---
The log file to create / append to. This may be set with the `BLAST_LOG` environment variable or 
the `--log` flag.

resume
------
If `true`, try to load the log file and skip previously successful items (failed items will be 
retried). This may be set with the `BLAST_RESUME` environment variable or the `--resume` flag.

rate
----
Initial rate in items per second. Simply enter a new rate during execution to adjust this. This may 
be set with the `BLAST_RATE` environment variable or the `--rate` flag.

workers
-------
Number of workers. This may be set with the `BLAST_WORKERS` environment variable or the `--workers` 
flag.

worker-type
-----------
The selected worker type. Register new worker types with the `RegisterWorkerType` method. This may 
be set with the `BLAST_WORKER_TYPE` environment variable or the `--worker-type` flag.

Your worker should satisfy the `Worker` interface, and optionally `Starter`, `Stopper`. See 
[httpworker](https://github.com/dave/blast/blob/master/httpworker/httpworker.go) and 
[dummyworker](https://github.com/dave/blast/blob/master/dummyworker/dummyworker.go) for simple 
examples. See the [blast command](https://github.com/dave/blast/blob/master/cmd/blast/blast.go) for 
an example of how to build a command with your custom worker type.

payload-template
----------------
This template is rendered and passed to the worker `Send` method. This may be set as a json encoded 
`map[string]interface{}` with the `BLAST_PAYLOAD_TEMPLATE` environment variable or the 
`--payload-template` flag.

Optional configuration options
==============================

repeat
------
When the end of the data file is found, repeats from the start. Useful for load testing. This may 
be set with the `BLAST_REPEAT` environment variable or the `--repeat` flag.

timeout
-------
The context passed to the worker has this timeout (in ms). The default value is 1000ms. Workers 
must respect this the context cancellation. We exit with an error if any worker is processing for 
timeout + 500ms. This may be set with the `BLAST_TIMEOUT` environment variable or the `--timeout` 
flag. 

log-data
--------
Array of data fields to include in the output log. This may be set as a json encoded `[]string` 
with the `BLAST_LOG_DATA` environment variable or the `--log-data` flag.

log-output
----------
Array of worker response fields to include in the output log. This may be set as a json encoded 
`[]string` with the `BLAST_LOG_OUTPUT` environment variable or the `--log-output` flag.

payload-variants
----------------
An array of maps that will cause each item to be repeated with the provided data. This may be set 
as a json encoded `[]map[string]string` with the `BLAST_PAYLOAD_VARIANTS` environment variable or 
the `--payload-variants` flag.  

worker-template
---------------
If the selected worker type satisfies the `Starter` or `Stopper` interfaces, the worker template 
will be rendered and passed to the `Start` or `Stop` methods to initialise each worker. Use with 
`worker-variants` to configure several workers differently to spread load. This may be set as a 
json encoded `map[string]interface{}` with the `BLAST_WORKER_TEMPLATE` environment variable or the 
`--worker-template` flag.

worker-variants
---------------
An array of maps that will cause each worker to be initialised with different data. This may be set 
as a json encoded `[]map[string]string` with the `BLAST_WORKER_VARIANTS` environment variable or 
the `--worker-variants` flag. 

To do
=====  
- [ ] Improve templating syntax and add functions for random data
- [ ] GCS worker with automatic authentication
- [ ] Adjust rate automatically in response to latency? PID controller?  
- [ ] Only use part of file: part i of j parts  
