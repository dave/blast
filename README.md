Blast
=====

Blast makes API requests at a fixed rate, based on input data from a CSV file. Upon restarting, 
successful items from previous runs are skipped. The number of concurrent workers is configurable. 
The rate may be changed during execution.

Config
======

Blast is configured with config.json:

data
----
The data file to load. If this starts with `gs://`, the data file is streamed directly from Google 
Cloud Services. The data file should be a CSV with a header row. The header columns may be used as 
variables in the `payload-template`. 

log
---
The filename of the log file to create or append to.

log-data
--------
Optional array of data items to add to the log file. Note adding lots of items to the log can 
affect performance.

resume
------
If resume is true, the previous log file is loaded, and items that were successful are skipped.

rate
----
The initial rate in items per second. Simply enter a new rate during execution to adjust this.

workers
-------
The number of concurrent workers to use.

worker-type
-----------
Worker types can easily be registered with `RegisterWorkerType`. Your worker should satisfy the 
`Worker` interface, and optionally `Starter`, `Stopper`. See [httpworker](https://github.com/dave/blast/blob/master/httpworker/httpworker.go)
for a simple example.

payload-template
----------------
This is the payload template that is rendered and passed to the worker `Send` function. Variables 
from the data file can be used in the form `{{name}}`.  

payload-variants
----------------
If multiple requests per item are needed, add an array of maps here.  

worker-template
---------------
If your worker satisfies the `Starter` or `Stopper` interfaces, `worker-template` will be rendered 
with one of the `worker-variants` data items, and this will be passed to the `Start` and `Stop` 
methods.

worker-variants
---------------
An array of maps containing data to be passed to the `Start` and `Stop` methods of workers. 


To do
=====

- [x] Read config  
- [x] Load CSV from GCS?  
- [x] Load CSV from file  
- [x] Parse CSV  
- [x] Testing framework  
- [x] Render template json into request  
- [x] Make request  
- [x] Save logs  
- [x] Parse previous log on load  
- [x] Skip successes from log  
- [x] Retry fails in log  
- [x] Adjust rate with interactive terminal input  
- [ ] Adjust rate automatically in response to latency? PID controller?  
- [ ] Only use part of file: part i of j parts  

Pipeline
========

Ticker
------
Tick -> is main loop waiting? If no, skip this tick. If yes, send to main loop.

Main loop (just one)
--------------------
Wait for timer tick -> Pull record from data -> Look up in skip and maybe skip -> Wait for available worker and send.

Worker loop (multiple)
----------------------
Wait for input -> Send -> Send log

Log loop (just one)
-------------------
Wait for log write -> write log -> send stats to rate loop 

Rate loop (just one)
--------------------
Wait for stats from log loop -> compute average stats -> PID calculation -> adjust rate?
Wait for rate change event -> adjust rate
