Blast
=====

* Blast makes API requests at a fixed rate, based on input data from a CSV file.   
* Upon restarting, successful items from previous runs are skipped.   
* The number of concurrent workers is configurable.  
* The rate may be changed interactively during execution.  
* Blast is protocol agnostic, and adding a new worker type is trivial.  

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
Concurrency:      1837 / 2000 workers in use
                                                                                             
Desired rate:     (all)        5000         1000         400         200         100                       
Actual rate:      1122         4839         986          399         200         100                       
Avg concurrency:  1439         1790         365          149         74          36                        
Duration:         01:20        00:13        00:12        00:23       00:12       00:18                     
                                                                                             
Total                                                                                        
-----                                                                                        
Started:          89758        63592        12568        9234        2532        1832        
Finished:         87921        61755        12568        9234        2532        1832        
Mean:             374.6 ms     371.7 ms     377.6 ms     374.2 ms    376.3 ms    376.4 ms                  
95th:             486.2 ms     487.1 ms     486.2 ms     489.6 ms    486.4 ms    488.2 ms                  
                                                                                             
200                                                                                          
---                                                                                          
Count:            84404 (96%)  59321 (96%)  12025 (96%)  8854 (96%)  2441 (96%)  1763 (96%)  
Mean:             372.9 ms     371.6 ms     373.9 ms     373.7 ms    374.1 ms    375.9 ms                  
95th:             487.1 ms     485.1 ms     490.2 ms     489.4 ms    485.9 ms    487.8 ms                  
                                                                                             
404                                                                                          
---                                                                                          
Count:            2633 (3%)    1815 (3%)    397 (3%)     292 (3%)    70 (3%)     59 (3%)     
Mean:             373.0 ms     372.9 ms     371.5 ms     367.0 ms    365.3 ms    377.4 ms                  
95th:             487.1 ms     488.1 ms     481.7 ms     483.9 ms    474.2 ms    481.7 ms                  
                                                                                             
500                                                                                          
---                                                                                          
Count:            887 (1%)     622 (1%)     146 (1%)     88 (1%)     21 (1%)     10 (1%)     
Mean:             375.4 ms     374.9 ms     380.1 ms     363.8 ms    400.9 ms    386.3 ms                  
95th:             487.1 ms     487.1 ms     483.4 ms     489.3 ms    497.2 ms    483.9 ms                  

Current rate is 5000 requests / second. Enter a new rate or press enter to view status.

Rate?
```

Config
======

Blast is configured by config file, command line flags or environment variables.

The `--config` flag specifies the config file to load, and can be `json`, `yaml`, `toml` or 
anything else that [viper](https://github.com/spf13/viper) can read. If the config flag is omitted, 
blast searches for `blast-config.json|yaml|toml` in current directory, `$HOME/.config/blast/` and 
`/etc/blast/`. Environment variables and command line flags override config file options.

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
- [ ] GCS worker with automatic authentication
- [ ] Adjust rate automatically in response to latency? PID controller?  
- [ ] Only use part of file: part i of j parts  
