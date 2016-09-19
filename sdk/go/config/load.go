package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

// LoadFile loads configuration from the file given by configPath and
// decodes it into cfg.
//
// Currently, only JSON is supported. Support for YAML is anticipated.
func LoadFile(cfg interface{}, configPath string) error {
	buf, err := ioutil.ReadFile(configPath)
	if err != nil {
		return err
	}
	err = json.Unmarshal(buf, cfg)
	if err != nil {
		return fmt.Errorf("Error decoding config %q: %v", configPath, err)
	}
	return nil
}
