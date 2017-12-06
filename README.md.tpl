[![Build Status](https://travis-ci.org/dave/blast.svg?branch=master)](https://travis-ci.org/dave/blast) [![Go Report Card](https://goreportcard.com/badge/github.com/dave/blast)](https://goreportcard.com/report/github.com/dave/blast) [![codecov](https://codecov.io/gh/dave/blast/branch/master/graph/badge.svg)](https://codecov.io/gh/dave/blast)


{{ "doc_go[1:]" | doc }}


Workers
=======

{{ "Worker" | doc }}

{{ "Starter" | doc }} 

Examples
========

For load testing:

```yaml
rate: 20000
workers: 1000
worker-type: "dummy"
payload-template:
  method: "POST"
  path: "/foo/?id={{"{{"}} rand_int 1 10000000 {{"}}"}}"
worker-template:
  print: false
  base: "https://{{"{{"}} .region {{"}}"}}.my-api.com"
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
  path: "/{{"{{"}} .user_name {{"}}"}}/{{"{{"}} .action {{"}}"}}/{{"{{"}} .type {{"}}"}}/"
worker-template:  
  print: true
  base: "https://{{"{{"}} .region {{"}}"}}.my-api.com"
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
{{ "Config.Data" | doc }}

log
---
{{ "Config.Log" | doc }}

resume
------
{{ "Config.Resume" | doc }}

rate
----
{{ "Config.Rate" | doc }}

workers
-------
{{ "Config.Workers" | doc }}

worker-type
-----------
{{ "Config.WorkerType" | doc }}

payload-template
----------------
{{ "Config.PayloadTemplate" | doc }}


Advanced configuration options
==============================

timeout
-------
{{ "Config.Timeout" | doc }} 

headers
-------
{{ "Config.Headers" | doc }}

log-data
--------
{{ "Config.LogData" | doc }}

log-output
----------
{{ "Config.LogOutput" | doc }}

worker-template
---------------
{{ "Config.WorkerTemplate" | doc }}

worker-variants
---------------
{{ "Config.WorkerVariants" | doc }}

payload-variants
----------------
{{ "Config.PayloadVariants" | doc }}

Control by code
===============
The blaster package may be used to start blast from code without using the command. Here's a some 
examples of usage:

{{ "ExampleBlaster_Start_batchJob" | example }}

{{ "ExampleBlaster_Start_loadTest" | example }}
 
To do
=====  
- [ ] Adjust rate automatically in response to latency? PID controller?  
- [ ] Only use part of file: part i of j parts  
