package blaster

import (
	"context"
	"testing"
	"time"
)

func TestBlaster_unmarshalConfig(t *testing.T) {
	type spec struct {
		key      string
		value    interface{}
		expected func(b Config) (bool, error)
	}
	run := func(name string, test spec) {
		ctx, cancel := context.WithCancel(context.Background())
		b := New(ctx, cancel)
		b.viper.Set(test.key, test.value)
		c := &Config{}
		if err := b.unmarshalConfig(c); err != nil {
			t.Fatal(err.Error())
		}
		success, err := test.expected(*c)
		if err != nil {
			t.Fatal(err.Error())
		}
		if !success {
			t.Fatalf("Fail in %s", name)
		}
	}
	tests := map[string]spec{
		"data": {"data", "a", func(c Config) (bool, error) {
			return c.Data == "a", nil
		}},
		"log": {"log", "b", func(c Config) (bool, error) {
			return c.Log == "b", nil
		}},
		"resume false": {"resume", "false", func(c Config) (bool, error) {
			return !c.Resume, nil
		}},
		"resume true": {"resume", "true", func(c Config) (bool, error) {
			return c.Resume, nil
		}},
		"headers native": {"headers", []string{"a", "b"}, func(c Config) (bool, error) {
			return c.Headers[0] == "a" && c.Headers[1] == "b", nil
		}},
		"headers json": {"headers", `["c","d"]`, func(c Config) (bool, error) {
			return c.Headers[0] == "c" && c.Headers[1] == "d", nil
		}},
		"rate int": {"rate", 12, func(c Config) (bool, error) {
			return c.Rate == 12, nil
		}},
		"rate string": {"rate", "34", func(c Config) (bool, error) {
			return c.Rate == 34, nil
		}},
		"workers int": {"workers", 56, func(c Config) (bool, error) {
			return c.Workers == 56, nil
		}},
		"workers string": {"workers", "78", func(c Config) (bool, error) {
			return c.Workers == 78, nil
		}},
		"timeout int": {"timeout", 91, func(c Config) (bool, error) {
			return c.Timeout == 91, nil
		}},
		"timeout string": {"timeout", "23", func(c Config) (bool, error) {
			return c.Timeout == 23, nil
		}},
		"worker type": {"worker-type", "a", func(c Config) (bool, error) {
			return c.WorkerType == "a", nil
		}},
		"log data native": {"log-data", []string{"a", "b"}, func(c Config) (bool, error) {
			return c.LogData[0] == "a" && c.LogData[1] == "b", nil
		}},
		"log data string": {"log-data", `["c","d"]`, func(c Config) (bool, error) {
			return c.LogData[0] == "c" && c.LogData[1] == "d", nil
		}},
		"log output native": {"log-output", []string{"a", "b"}, func(c Config) (bool, error) {
			return c.LogOutput[0] == "a" && c.LogOutput[1] == "b", nil
		}},
		"log output string": {"log-output", `["c","d"]`, func(c Config) (bool, error) {
			return c.LogOutput[0] == "c" && c.LogOutput[1] == "d", nil
		}},
		"worker template native": {"worker-template", map[string]interface{}{"a": "b", "c": 1}, func(c Config) (bool, error) {
			return c.WorkerTemplate["a"] == "b" && c.WorkerTemplate["c"] == 1, nil
		}},
		"worker template json": {"worker-template", `{"d": "e", "f": 2}`, func(c Config) (bool, error) {
			return c.WorkerTemplate["d"] == "e" && c.WorkerTemplate["f"] == 2.0, nil // after json decode, all numbers are float64
		}},
		"payload template native": {"payload-template", map[string]interface{}{"g": "h", "i": 3}, func(c Config) (bool, error) {
			return c.PayloadTemplate["g"] == "h" && c.PayloadTemplate["i"] == 3, nil
		}},
		"payload template json": {"payload-template", `{"j": "k", "l": 4}`, func(c Config) (bool, error) {
			return c.PayloadTemplate["j"] == "k" && c.PayloadTemplate["l"] == 4.0, nil // after json decode, all numbers are float64
		}},
		"worker variants native": {"worker-variants", []map[string]string{{"a": "b"}, {"c": "d"}}, func(c Config) (bool, error) {
			return c.WorkerVariants[0]["a"] == "b" && c.WorkerVariants[1]["c"] == "d", nil
		}},
		"worker variants json": {"worker-variants", `[{"e":"f"},{"g":"h"}]`, func(c Config) (bool, error) {
			return c.WorkerVariants[0]["e"] == "f" && c.WorkerVariants[1]["g"] == "h", nil
		}},
		"payload variants native": {"payload-variants", []map[string]string{{"a": "b"}, {"c": "d"}}, func(c Config) (bool, error) {
			return c.PayloadVariants[0]["a"] == "b" && c.PayloadVariants[1]["c"] == "d", nil
		}},
		"payload variants json": {"payload-variants", `[{"e":"f"},{"g":"h"}]`, func(c Config) (bool, error) {
			return c.PayloadVariants[0]["e"] == "f" && c.PayloadVariants[1]["g"] == "h", nil
		}},
		"quiet": {"quiet", true, func(c Config) (bool, error) { return c.Quiet, nil }},
	}
	for name, test := range tests {
		run(name, test)
	}
}

func TestBlaster_Initialise(t *testing.T) {
	type spec struct {
		config   Config
		expected func(b *Blaster) (bool, error)
	}
	run := func(name string, test spec) {
		ctx, cancel := context.WithCancel(context.Background())
		b := New(ctx, cancel)
		b.RegisterWorkerType("w", func() Worker { return &ExampleWorker{} })
		if err := b.Initialise(ctx, test.config); err != nil {
			t.Fatal(err.Error())
		}
		success, err := test.expected(b)
		if err != nil {
			t.Fatal(err.Error())
		}
		if !success {
			t.Fatalf("Fail in %s", name)
		}
	}
	tests := map[string]spec{
		"resume": {Config{Resume: true}, func(b *Blaster) (bool, error) {
			return b.Resume == true, nil
		}},
		"rate": {Config{Rate: 100}, func(b *Blaster) (bool, error) {
			return b.Rate == 100, nil
		}},
		"workers": {Config{Workers: 100}, func(b *Blaster) (bool, error) {
			return b.Workers == 100, nil
		}},
		"worker-type": {Config{WorkerType: "w"}, func(b *Blaster) (bool, error) {
			return b.workerFunc != nil, nil
		}},
		"payload-template": {Config{PayloadTemplate: map[string]interface{}{"a": "{{ .b }}"}}, func(b *Blaster) (bool, error) {
			if b.payloadRenderer == nil {
				return false, nil
			}
			r, err := b.payloadRenderer.render(map[string]string{"b": "1"})
			if err != nil {
				return false, err
			}
			return r.(map[string]interface{})["a"] == "1", nil
		}},
		"worker-template": {Config{WorkerTemplate: map[string]interface{}{"c": "{{ .d }}"}}, func(b *Blaster) (bool, error) {
			if b.workerRenderer == nil {
				return false, nil
			}
			r, err := b.workerRenderer.render(map[string]string{"d": "2"})
			if err != nil {
				return false, err
			}
			return r.(map[string]interface{})["c"] == "2", nil
		}},
		"timeout": {Config{Timeout: 123}, func(b *Blaster) (bool, error) {
			return b.softTimeout == time.Millisecond*123 && b.hardTimeout == time.Millisecond*1123, nil
		}},
		"log-data": {Config{LogData: []string{"a", "b"}}, func(b *Blaster) (bool, error) {
			return b.LogData[0] == "a" && b.LogData[1] == "b", nil
		}},
		"log-output": {Config{LogOutput: []string{"c", "d"}}, func(b *Blaster) (bool, error) {
			return b.LogOutput[0] == "c" && b.LogOutput[1] == "d", nil
		}},
		"payload-variants": {Config{PayloadVariants: []map[string]string{{"a": "b"}, {"c": "d"}}}, func(b *Blaster) (bool, error) {
			return b.PayloadVariants[0]["a"] == "b" && b.PayloadVariants[1]["c"] == "d", nil
		}},
		"worker-variants": {Config{WorkerVariants: []map[string]string{{"a": "b"}, {"c": "d"}}}, func(b *Blaster) (bool, error) {
			return b.WorkerVariants[0]["a"] == "b" && b.WorkerVariants[1]["c"] == "d", nil
		}},
		"headers": {Config{Headers: []string{"a", "b"}}, func(b *Blaster) (bool, error) {
			return b.Headers[0] == "a" && b.Headers[1] == "b", nil
		}},
		"quiet": {Config{Quiet: true}, func(b *Blaster) (bool, error) {
			return b.Quiet, nil
		}},
	}
	for name, test := range tests {
		run(name, test)
	}
}
