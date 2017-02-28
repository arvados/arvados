package setup

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
)

func atomicWriteFile(name string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(path.Dir(name), 0755); err != nil {
		return err
	}
	tmp, err := ioutil.TempFile(path.Dir(name), path.Base(name)+"~")
	if err != nil {
		return err
	}
	defer func() {
		if tmp != nil {
			os.Remove(tmp.Name())
		}
	}()
	_, err = tmp.Write(data)
	if err != nil {
		return err
	}
	err = tmp.Close()
	if err != nil {
		return err
	}
	err = os.Chmod(tmp.Name(), mode)
	if err != nil {
		return err
	}
	err = os.Rename(tmp.Name(), name)
	if err != nil {
		return err
	}
	tmp = nil
	return nil
}

func atomicWriteJSON(name string, data interface{}, mode os.FileMode) error {
	j, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteFile(name, j, mode)
}
