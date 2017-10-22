package blast

import (
	"encoding/json"
	"io/ioutil"

	"github.com/pkg/errors"
)

type configDef struct {
	Data            string                 `json:"data"`
	Log             string                 `json:"log"`
	LogData         []string               `json:"log-data"`
	Resume          bool                   `json:"resume"`
	Rate            float64                `json:"rate"`
	Workers         int                    `json:"workers"`
	WorkerType      string                 `json:"worker-type"`
	WorkerTemplate  map[string]interface{} `json:"worker-template"`
	PayloadTemplate map[string]interface{} `json:"payload-template"`
	PayloadVariants []map[string]string    `json:"payload-variants"`
	WorkerVariants  []map[string]string    `json:"worker-variants"`
}

func (b *Blaster) loadConfig() error {
	bytes, err := ioutil.ReadFile("config.json")
	if err != nil {
		return errors.WithStack(err)
	}
	b.config = &configDef{}
	if err := json.Unmarshal(bytes, b.config); err != nil {
		return errors.WithStack(err)
	}
	if len(b.config.PayloadVariants) == 0 {
		// if payload-variants is empty, add a single variant with empty data
		b.config.PayloadVariants = []map[string]string{
			map[string]string{},
		}
	}
	b.rate = b.config.Rate
	return nil
}
