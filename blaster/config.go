package blaster

import (
	"encoding/json"

	"strings"

	"fmt"

	"os"

	"time"

	"context"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Note: viper uses the mapstructure lib to unmarshal data, so we need to use the "mapstructure"
// struct tag key in addition to "json".

// Config provides all the standard config options. Use the Initialise method to configure with a provided Config.
type Config struct {
	// Data sets the the data file to load. If none is specified, the worker will be called repeatedly until interrupted (useful for load testing). Load a local file or stream directly from a GCS bucket with `gs://{bucket}/{filename}.csv`. Data should be in csv format, and if `headers` is not specified the first record will be used as the headers. If a newline character is found, this string is read as the data.
	Data string `mapstructure:"data" json:"data"`

	// Log sets the filename of the log file to create / append to.
	Log string `mapstructure:"log" json:"log"`

	// Resume instructs the tool to load the log file and skip previously successful items. Failed items will be retried.
	Resume bool `mapstructure:"resume" json:"resume"`

	// Rate sets the initial rate in requests per second. Simply enter a new rate during execution to adjust this. (Default: 10 requests / second).
	Rate float64 `mapstructure:"rate" json:"rate"`

	// Workers sets the number of concurrent workers. (Default: 10 workers).
	Workers int `mapstructure:"workers" json:"workers"`

	// WorkerType sets the selected worker type. Register new worker types with the `RegisterWorkerType` method.
	WorkerType string `mapstructure:"worker-type" json:"worker-type"`

	// PayloadTemplate sets the template that is rendered and passed to the worker `Send` method. When setting this by command line flag or environment variable, use a json encoded string.
	PayloadTemplate map[string]interface{} `mapstructure:"payload-template" json:"payload-template"`

	// Timeout sets the deadline in the context passed to the worker. Workers must respect this the context cancellation. We exit with an error if any worker is processing for timeout + 1 second. (Default: 1 second).
	Timeout int `mapstructure:"timeout" json:"timeout"`

	// LogData sets an array of data fields to include in the output log. When setting this by command line flag or environment variable, use a json encoded string.
	LogData []string `mapstructure:"log-data" json:"log-data"`

	// LogOutput sets an array of worker response fields to include in the output log. When setting this by command line flag or environment variable, use a json encoded string.
	LogOutput []string `mapstructure:"log-output" json:"log-output"`

	// PayloadVariants sets an array of maps that will cause each data item to be repeated with the provided data. When setting this by command line flag or environment variable, use a json encoded string.
	PayloadVariants []map[string]string `mapstructure:"payload-variants" json:"payload-variants"`

	// WorkerVariants sets an array of maps that will cause each worker to be initialised with different data. When setting this by command line flag or environment variable, use a json encoded string.
	WorkerVariants []map[string]string `mapstructure:"worker-variants" json:"worker-variants"`

	// WorkerTemplate sets a template to render and pass to the worker `Start` or `Stop` methods if the worker satisfies the `Starter` or `Stopper` interfaces. Use with `worker-variants` to configure several workers differently to spread load. When setting this by command line flag or environment variable, use a json encoded string.
	WorkerTemplate map[string]interface{} `mapstructure:"worker-template" json:"worker-template"`

	// Headers sets the data file headers. If omitted, the first record of the csv data source is used. When setting this by command line flag or environment variable, use a json encoded string.
	Headers []string `mapstructure:"headers" json:"headers"`

	// Quiet instructs the tool to prevent interactive features. No summary is printed during operation and the rate cannot be changed interactively.
	Quiet bool `mapstructure:"quiet" json:"quiet"`
}

// LoadConfig parses command line flags and loads a config file from disk. A Config is returned which may be used with the Initialise method to complete configuration.
func (b *Blaster) LoadConfig() (Config, error) {

	c := Config{}

	dryRunFlag, configFlag := b.setupFlags()

	if err := b.setupViper(configFlag); err != nil {
		return Config{}, err
	}

	if err := b.unmarshalConfig(&c); err != nil {
		return Config{}, err
	}

	if dryRunFlag {
		by, _ := json.MarshalIndent(c, "", "\t")
		fmt.Println(string(by))
		os.Exit(0)
	}

	return c, nil
}

func (b *Blaster) setupFlags() (dryRunFlag bool, configFlag string) {
	dryRunFlagRaw := pflag.Bool("dry", false, "`` If true, just prints the current config and exits.")
	configFlagRaw := pflag.String("config", "", "`` The config file to load.")

	pflag.String("data", "", "`` "+doc["Config.Data"])
	pflag.String("log", "", "`` "+doc["Config.Log"])
	pflag.Bool("resume", false, "`` "+doc["Config.Resume"])
	pflag.String("headers", "", "`` "+doc["Config.Headers"])
	pflag.Float64("rate", 10.0, "`` "+doc["Config.Rate"])
	pflag.Int("workers", 10, "`` "+doc["Config.Workers"])
	pflag.Int("timeout", 1000, "`` "+doc["Config.Timeout"])
	pflag.String("worker-type", "", "`` "+doc["Config.WorkerType"])
	pflag.String("log-data", "", "`` "+doc["Config.LogData"])
	pflag.String("log-output", "", "`` "+doc["Config.LogOutput"])
	pflag.String("payload-template", "", "`` "+doc["Config.PayloadTemplate"])
	pflag.String("worker-template", "", "`` "+doc["Config.WorkerTemplate"])
	pflag.String("payload-variants", "", "`` "+doc["Config.PayloadVariants"])
	pflag.String("worker-variants", "", "`` "+doc["Config.WorkerVariants"])
	pflag.Bool("quiet", false, "`` "+doc["Config.Quiet"])

	pflag.Parse()

	if dryRunFlagRaw != nil {
		dryRunFlag = *dryRunFlagRaw
	}
	if configFlagRaw != nil {
		configFlag = *configFlagRaw
	}
	return dryRunFlag, configFlag
}

func (b *Blaster) setupViper(configFlag string) error {

	if configFlag != "" {
		b.viper.SetConfigFile(configFlag)
	} else {
		b.viper.SetConfigName("blast-config") // name of config file (without extension)
		b.viper.AddConfigPath("/etc/blast/")
		b.viper.AddConfigPath("$HOME/.config/blast/")
		b.viper.AddConfigPath(".")
	}
	if err := b.viper.ReadInConfig(); err != nil {
		if _, isNotFound := err.(viper.ConfigFileNotFoundError); !isNotFound {
			return errors.WithStack(err)
		}
	}

	b.viper.BindPFlags(pflag.CommandLine)

	b.viper.SetTypeByDefaultValue(true)
	b.viper.SetDefault("data", "")
	b.viper.SetDefault("config", "")
	b.viper.SetDefault("log", "")
	b.viper.SetDefault("resume", false)
	b.viper.SetDefault("rate", 10.0)
	b.viper.SetDefault("workers", 10)
	b.viper.SetDefault("timeout", 1000)
	b.viper.SetDefault("worker-type", "")
	b.viper.SetDefault("log-data", []string{})
	b.viper.SetDefault("log-output", []string{})
	b.viper.SetDefault("headers", []string{})
	b.viper.SetDefault("worker-template", map[string]interface{}{})
	b.viper.SetDefault("payload-template", map[string]interface{}{})
	b.viper.SetDefault("payload-variants", []map[string]string{{}})
	b.viper.SetDefault("worker-variants", []map[string]string{{}})
	b.viper.SetDefault("quiet", false)

	b.viper.SetEnvPrefix("blast")
	b.viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	b.viper.AutomaticEnv()
	return nil
}

func (b *Blaster) unmarshalConfig(c *Config) error {
	// viper is unable to unmarshal complex data types, so we must do them manually:
	if err := b.viper.UnmarshalKey("data", &c.Data); err != nil {
		return errors.WithStack(err)
	}
	if err := b.viper.UnmarshalKey("log", &c.Log); err != nil {
		return errors.WithStack(err)
	}
	if err := b.viper.UnmarshalKey("resume", &c.Resume); err != nil {
		return errors.WithStack(err)
	}
	if s := b.viper.GetString("headers"); s != "" {
		// if array type data is actually a string, unmarshal it from json
		if err := json.Unmarshal([]byte(s), &c.Headers); err != nil {
			return errors.WithStack(err)
		}
	} else {
		if err := b.viper.UnmarshalKey("headers", &c.Headers); err != nil {
			return errors.WithStack(err)
		}
	}
	if err := b.viper.UnmarshalKey("rate", &c.Rate); err != nil {
		return errors.WithStack(err)
	}
	if err := b.viper.UnmarshalKey("workers", &c.Workers); err != nil {
		return errors.WithStack(err)
	}
	if err := b.viper.UnmarshalKey("timeout", &c.Timeout); err != nil {
		return errors.WithStack(err)
	}
	if err := b.viper.UnmarshalKey("worker-type", &c.WorkerType); err != nil {
		return errors.WithStack(err)
	}
	if s := b.viper.GetString("log-data"); s != "" {
		// if array type data is actually a string, unmarshal it from json
		if err := json.Unmarshal([]byte(s), &c.LogData); err != nil {
			return errors.WithStack(err)
		}
	} else {
		if err := b.viper.UnmarshalKey("log-data", &c.LogData); err != nil {
			return errors.WithStack(err)
		}
	}
	if s := b.viper.GetString("log-output"); s != "" {
		// if array type data is actually a string, unmarshal it from json
		if err := json.Unmarshal([]byte(s), &c.LogOutput); err != nil {
			return errors.WithStack(err)
		}
	} else {
		if err := b.viper.UnmarshalKey("log-output", &c.LogOutput); err != nil {
			return errors.WithStack(err)
		}
	}
	if err := b.viper.UnmarshalKey("worker-template", &c.WorkerTemplate); err != nil {
		if s := b.viper.GetString("worker-template"); s != "" {
			if err := json.Unmarshal([]byte(s), &c.WorkerTemplate); err != nil {
				return errors.WithStack(err)
			}
		}
	}
	if err := b.viper.UnmarshalKey("payload-template", &c.PayloadTemplate); err != nil {
		if s := b.viper.GetString("payload-template"); s != "" {
			if err := json.Unmarshal([]byte(s), &c.PayloadTemplate); err != nil {
				return errors.WithStack(err)
			}
		}
	}
	if s := b.viper.GetString("worker-variants"); s != "" {
		// if array type data is actually a string, unmarshal it from json
		if err := json.Unmarshal([]byte(s), &c.WorkerVariants); err != nil {
			return errors.WithStack(err)
		}
	} else {
		if err := b.viper.UnmarshalKey("worker-variants", &c.WorkerVariants); err != nil {
			return errors.WithStack(err)
		}
	}
	if s := b.viper.GetString("payload-variants"); s != "" {
		if err := json.Unmarshal([]byte(s), &c.PayloadVariants); err != nil {
			return errors.WithStack(err)
		}
	} else {
		if err := b.viper.UnmarshalKey("payload-variants", &c.PayloadVariants); err != nil {
			return errors.WithStack(err)
		}
	}
	if err := b.viper.UnmarshalKey("quiet", &c.Quiet); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// Initialise configures the Blaster with config options in a provided Config
func (b *Blaster) Initialise(ctx context.Context, c Config) error {

	b.Rate = c.Rate
	if c.Workers != 0 {
		b.Workers = c.Workers
	}
	b.Quiet = c.Quiet
	b.Resume = c.Resume

	if len(c.LogData) > 0 {
		b.LogData = c.LogData
	}
	if len(c.LogOutput) > 0 {
		b.LogOutput = c.LogOutput
	}

	if len(c.WorkerVariants) > 0 {
		b.WorkerVariants = c.WorkerVariants
	}
	if len(c.PayloadVariants) > 0 {
		b.PayloadVariants = c.PayloadVariants
	}

	if len(c.Headers) > 0 {
		b.Headers = c.Headers
	}

	if c.Timeout > 0 {
		b.SetTimeout(time.Duration(c.Timeout) * time.Millisecond)
	}

	if c.WorkerType != "" {
		wf, ok := b.workerTypes[c.WorkerType]
		if !ok {
			panic(fmt.Sprintf("Worker type %s not found", c.WorkerType))
		}
		b.SetWorker(wf)
	}

	if err := b.SetPayloadTemplate(c.PayloadTemplate); err != nil {
		return err
	}

	if err := b.SetWorkerTemplate(c.WorkerTemplate); err != nil {
		return err
	}

	if c.Data != "" {
		if err := b.openData(ctx, c.Data, len(c.Headers) == 0); err != nil {
			return err
		}
	}

	if c.Log != "" {
		if err := b.initialiseLog(c.Log); err != nil {
			return err
		}
	}

	return nil
}
