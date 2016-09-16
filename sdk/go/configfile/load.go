package configfile

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

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
