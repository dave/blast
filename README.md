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

Config
======

Blast is configured by config file, command line flags or environment variables.

The config file should be called `blast-config.xxx`, and can be `json`, `yaml`, `toml` or anything 
else that [viper](https://github.com/spf13/viper) can read. Blast searches in `/etc/blast/`, 
`$HOME/.config/blast/` and the current directory for the config file. Only one config file may be 
used, but environment variables and command line flags override config options.

See [blast-config.yaml](https://github.com/dave/blast/blob/master/blast-config.yaml) for an 
annotated example.

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
- [ ] Adjust rate automatically in response to latency? PID controller?  
- [ ] Only use part of file: part i of j parts  
