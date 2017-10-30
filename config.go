package blast

import (
	"encoding/json"

	"strings"

	"fmt"

	"os"

	"time"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type configDef struct {
	Data            string                 `mapstructure:"data" json:"data"`
	Log             string                 `mapstructure:"log" json:"log"`
	LogData         []string               `mapstructure:"log-data" json:"log-data"`
	LogOutput       []string               `mapstructure:"log-output" json:"log-output"`
	Resume          bool                   `mapstructure:"resume" json:"resume"`
	Repeat          bool                   `mapstructure:"repeat" json:"repeat"`
	Rate            float64                `mapstructure:"rate" json:"rate"`
	Workers         int                    `mapstructure:"workers" json:"workers"`
	Timeout         int                    `mapstructure:"timeout" json:"timeout"`
	WorkerType      string                 `mapstructure:"worker-type" json:"worker-type"`
	WorkerTemplate  map[string]interface{} `mapstructure:"worker-template" json:"worker-template"`
	PayloadTemplate map[string]interface{} `mapstructure:"payload-template" json:"payload-template"`
	PayloadVariants []map[string]string    `mapstructure:"payload-variants" json:"payload-variants"`
	WorkerVariants  []map[string]string    `mapstructure:"worker-variants" json:"worker-variants"`
}

func (b *Blaster) loadConfigViper() error {

	dryRun := pflag.Bool("dry", false, "If true, just prints the current config and exits.")
	configFlag := pflag.String("config", "", "The config file to load. This may be set with the BLAST_CONFIG environment variable.")
	pflag.String("data", "", "The data file to load. Stream directly from a GCS bucket with 'gs://{bucket}/{filename}.csv'. Data should be in CSV format with a header row. This may be set with the BLAST_DATA environment variable or the data config option.")
	pflag.String("log", "", "The log file to create / append to. This may be set with the BLAST_LOG environment variable or the log config option.")
	pflag.Bool("resume", true, "If true, try to load the log file and skip previously successful items (failed items will be retried). This may be set with the BLAST_RESUME environment variable or the resume config option.")
	pflag.Bool("repeat", false, "When the end of the data file is found, repeats from the start. Useful for load testing. This may be set with the BLAST_REPEAT environment variable or the repeat config option.")
	pflag.Float64("rate", 1.0, "Initial rate in items per second. Simply enter a new rate during execution to adjust this. This may be set with the BLAST_RATE environment variable or the rate config option.")
	pflag.Int("workers", 5, "Number of workers. This may be set with the BLAST_WORKERS environment variable or the workers config option.")
	pflag.Int("timeout", 1000, "The context passed to the worker has this timeout (in ms). The default value is 1000ms. Workers must respect this the context cancellation. We exit with an error if any worker is processing for timeout + 500ms. This may be set with the BLAST_TIMEOUT environment variable or the timeout config option.")
	pflag.String("worker-type", "", "The selected worker type. Register new worker types with the `RegisterWorkerType` method. This may be set with the BLAST_WORKER_TYPE environment variable or the worker-type config option.")
	pflag.String("log-data", "", "Array of data fields to include in the output log. This may be set as a json encoded []string with the BLAST_LOG_DATA environment variable or the log-data config option.")
	pflag.String("log-output", "", "Array of worker response fields to include in the output log. This may be set as a json encoded []string with the BLAST_LOG_OUTPUT environment variable or the log-output config option.")
	pflag.String("payload-template", "", "This template is rendered and passed to the worker `Send` method. This may be set as a json encoded map[string]interface{} with the BLAST_PAYLOAD_TEMPLATE environment variable or the payload-template config option.")
	pflag.String("worker-template", "", "If the selected worker type satisfies the `Starter` or `Stopper` interfaces, the worker template will be rendered and passed to the `Start` or `Stop` methods to initialise each worker. Use with `worker-variants` to configure several workers differently to spread load. This may be set as a json encoded map[string]interface{} with the BLAST_WORKER_TEMPLATE environment variable or the worker-template config option.")
	pflag.String("payload-variants", "", "An array of maps that will cause each item to be repeated with the provided data. This may be set as a json encoded []map[string]string with the BLAST_PAYLOAD_VARIANTS environment variable or the payload-variants config option.")
	pflag.String("worker-variants", "", "An array of maps that will cause each worker to be initialised with different data. This may be set as a json encoded []map[string]string with the BLAST_WORKER_VARIANTS environment variable or the -worker-variants config option.")

	pflag.Parse()

	if configFlag != nil && *configFlag != "" {
		b.viper.SetConfigFile(*configFlag)
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
	b.viper.SetDefault("resume", true)
	b.viper.SetDefault("repeat", false)
	b.viper.SetDefault("rate", 1.0)
	b.viper.SetDefault("workers", 5)
	b.viper.SetDefault("timeout", 1000)
	b.viper.SetDefault("worker-type", "")
	b.viper.SetDefault("log-data", []string{})
	b.viper.SetDefault("log-output", []string{})
	b.viper.SetDefault("worker-template", map[string]interface{}{})
	b.viper.SetDefault("payload-template", map[string]interface{}{})
	b.viper.SetDefault("payload-variants", []map[string]string{{}})
	b.viper.SetDefault("worker-variants", []map[string]string{{}})

	b.viper.SetEnvPrefix("blast")
	b.viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	b.viper.AutomaticEnv()

	// Unmarshal the config to the configDef. Note: viper uses the mapstructure lib to unmarshal
	// data, so we need to use the "mapstructure" struct tag key instead of "json".
	b.config = &configDef{}

	// viper is unable to unmarshal complex data types, so we must do them manually:
	if err := b.viper.UnmarshalKey("data", &b.config.Data); err != nil {
		return errors.WithStack(err)
	}
	if err := b.viper.UnmarshalKey("log", &b.config.Log); err != nil {
		return errors.WithStack(err)
	}
	if err := b.viper.UnmarshalKey("resume", &b.config.Resume); err != nil {
		return errors.WithStack(err)
	}
	if err := b.viper.UnmarshalKey("repeat", &b.config.Repeat); err != nil {
		return errors.WithStack(err)
	}
	if err := b.viper.UnmarshalKey("rate", &b.config.Rate); err != nil {
		return errors.WithStack(err)
	}
	if err := b.viper.UnmarshalKey("workers", &b.config.Workers); err != nil {
		return errors.WithStack(err)
	}
	if err := b.viper.UnmarshalKey("timeout", &b.config.Timeout); err != nil {
		return errors.WithStack(err)
	}
	if err := b.viper.UnmarshalKey("worker-type", &b.config.WorkerType); err != nil {
		return errors.WithStack(err)
	}
	if err := b.viper.UnmarshalKey("log-data", &b.config.LogData); err != nil {
		if s := b.viper.GetString("log-data"); s != "" {
			if err := json.Unmarshal([]byte(s), &b.config.LogData); err != nil {
				return errors.WithStack(err)
			}
		}
	}
	if err := b.viper.UnmarshalKey("log-output", &b.config.LogOutput); err != nil {
		if s := b.viper.GetString("log-output"); s != "" {
			if err := json.Unmarshal([]byte(s), &b.config.LogOutput); err != nil {
				return errors.WithStack(err)
			}
		}
	}
	if err := b.viper.UnmarshalKey("worker-template", &b.config.WorkerTemplate); err != nil {
		if s := b.viper.GetString("worker-template"); s != "" {
			if err := json.Unmarshal([]byte(s), &b.config.WorkerTemplate); err != nil {
				return errors.WithStack(err)
			}
		}
	}
	if err := b.viper.UnmarshalKey("payload-template", &b.config.PayloadTemplate); err != nil {
		if s := b.viper.GetString("payload-template"); s != "" {
			if err := json.Unmarshal([]byte(s), &b.config.PayloadTemplate); err != nil {
				return errors.WithStack(err)
			}
		}
	}
	if err := b.viper.UnmarshalKey("worker-variants", &b.config.WorkerVariants); err != nil {
		if s := b.viper.GetString("worker-variants"); s != "" {
			if err := json.Unmarshal([]byte(s), &b.config.WorkerVariants); err != nil {
				return errors.WithStack(err)
			}
		}
	}
	if err := b.viper.UnmarshalKey("payload-variants", &b.config.PayloadVariants); err != nil {
		if s := b.viper.GetString("payload-variants"); s != "" {
			if err := json.Unmarshal([]byte(s), &b.config.PayloadVariants); err != nil {
				return errors.WithStack(err)
			}
		}
	}

	if *dryRun {
		by, _ := json.MarshalIndent(b.config, "", "\t")
		fmt.Println(string(by))
		os.Exit(0)
	}

	// Set the current rate to the config rate.
	b.rate = b.config.Rate

	b.softTimeout = time.Duration(b.config.Timeout) * time.Millisecond
	b.hardTimeout = time.Duration(b.config.Timeout+500) * time.Millisecond

	if b.config.Resume && b.config.Repeat {
		panic("Can't use repeat and resume at the same time!")
	}

	return nil
}
