/*
Package blaster provides the back-end for blast - a tool for load testing and sending api requests in bulk.

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
 go get -u github.com/dave/blast
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
 Blast is configured by config file, command line flags or environment variables. The `--config` flag specifies the config file to load, and can be `json`, `yaml`, `toml` or anything else that [viper](https://github.com/spf13/viper) can read. If the config flag is omitted, blast searches for `blast-config.xxx` in the current directory, `$HOME/.config/blast/` and `/etc/blast/`.

 Environment variables and command line flags override config file options. Environment variables are upper case and prefixed with "BLAST" e.g. `BLAST_PAYLOAD_TEMPLATE`.

 Templates
 =========
 The `payload-template` and `worker-template` options accept values that are rendered using the Go text/template system. Variables of the form `{{ .name }}` or `{{ "name" }}` are replaced with data.

 Additionally, several simple functions are available to inject random data which is useful in load testing scenarios:

 * `{{ rand_int -5 5 }}` - a random integer between -5 and 5.
 * `{{ rand_float -5 5 }}` - a random float between -5 and 5.
 * `{{ rand_string 10 }}` - a random string, length 10.

*/
package blaster
