package setup

import (
	"bytes"
	"encoding/json"
	"log"
	"strconv"

	consul "github.com/hashicorp/consul/api"
)

func (s *Setup) maybeConfigure() error {
	cc, err := s.ConsulMaster()
	if err != nil {
		return err
	}
	kv := cc.KV()

	_, _, err = kv.Get("arvados/service/API/port", nil)
	if err == nil {
		// already configured
		return nil
	}
	return s.Reconfigure()
}

func (s *Setup) Reconfigure() error {
	cc, err := s.ConsulMaster()
	if err != nil {
		return err
	}
	kv := cc.KV()

	var portmap map[string]int
	buf, err := json.Marshal(s.Ports)
	if err != nil {
		return err
	}
	err = json.Unmarshal(buf, &portmap)
	if err != nil {
		return err
	}
	for name, port := range portmap {
		pair := &consul.KVPair{
			Key:   "arvados/service/" + name + "/port",
			Value: []byte(strconv.Itoa(port)),
		}

		cur, _, err := kv.Get(pair.Key, nil)
		if err != nil {
			log.Print(err)
		} else if bytes.Compare(cur.Value, pair.Value) == 0 {
			continue
		}

		log.Printf("%q => %q", pair.Key, pair.Value)
		_, err = kv.Put(pair, nil)
		if err != nil {
			return err
		}
	}
	return nil
}
