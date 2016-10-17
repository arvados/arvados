package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
)

type Config struct {
	Listen string

	PIDFile string

	MaxBuffers  int
	MaxRequests int

	BlobSignatureTTL    arvados.Duration
	BlobSigningKeyFile  string
	RequireSignatures   bool
	SystemAuthTokenFile string
	EnableDelete        bool
	TrashLifetime       arvados.Duration
	TrashCheckInterval  arvados.Duration

	Volumes VolumeList

	blobSigningKey  []byte
	systemAuthToken string
}

var theConfig = DefaultConfig()

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Listen:             ":25107",
		MaxBuffers:         128,
		RequireSignatures:  true,
		BlobSignatureTTL:   arvados.Duration(14 * 24 * time.Hour),
		TrashLifetime:      arvados.Duration(14 * 24 * time.Hour),
		TrashCheckInterval: arvados.Duration(24 * time.Hour),
		Volumes:            []Volume{},
	}
}

// Start should be called exactly once: after setting all public
// fields, and before using the config.
func (cfg *Config) Start() error {
	if cfg.MaxBuffers < 0 {
		return fmt.Errorf("MaxBuffers must be greater than zero")
	}
	bufs = newBufferPool(cfg.MaxBuffers, BlockSize)

	if cfg.MaxRequests < 1 {
		cfg.MaxRequests = cfg.MaxBuffers * 2
		log.Printf("MaxRequests <1 or not specified; defaulting to MaxBuffers * 2 == %d", cfg.MaxRequests)
	}

	if cfg.BlobSigningKeyFile != "" {
		buf, err := ioutil.ReadFile(cfg.BlobSigningKeyFile)
		if err != nil {
			return fmt.Errorf("reading blob signing key file: %s", err)
		}
		cfg.blobSigningKey = bytes.TrimSpace(buf)
		if len(cfg.blobSigningKey) == 0 {
			return fmt.Errorf("blob signing key file %q is empty", cfg.BlobSigningKeyFile)
		}
	} else if cfg.RequireSignatures {
		return fmt.Errorf("cannot enable RequireSignatures (-enforce-permissions) without a blob signing key")
	} else {
		log.Println("Running without a blob signing key. Block locators " +
			"returned by this server will not be signed, and will be rejected " +
			"by a server that enforces permissions.")
		log.Println("To fix this, use the BlobSigningKeyFile config entry.")
	}

	if fn := cfg.SystemAuthTokenFile; fn != "" {
		buf, err := ioutil.ReadFile(fn)
		if err != nil {
			return fmt.Errorf("cannot read system auth token file %q: %s", fn, err)
		}
		cfg.systemAuthToken = strings.TrimSpace(string(buf))
	}

	if cfg.EnableDelete {
		log.Print("Trash/delete features are enabled. WARNING: this has not " +
			"been extensively tested. You should disable this unless you can afford to lose data.")
	}

	if len(cfg.Volumes) == 0 {
		if (&unixVolumeAdder{cfg}).Discover() == 0 {
			return fmt.Errorf("no volumes found")
		}
	}
	for _, v := range cfg.Volumes {
		if err := v.Start(); err != nil {
			return fmt.Errorf("volume %s: %s", v, err)
		}
		log.Printf("Using volume %v (writable=%v)", v, v.Writable())
	}
	return nil
}

// VolumeTypes is built up by init() funcs in the source files that
// define the volume types.
var VolumeTypes = []func() VolumeWithExamples{}

type VolumeList []Volume

// UnmarshalJSON, given an array of objects, deserializes each object
// as the volume type indicated by the object's Type field.
func (vols *VolumeList) UnmarshalJSON(data []byte) error {
	typeMap := map[string]func() VolumeWithExamples{}
	for _, factory := range VolumeTypes {
		t := factory().Type()
		if _, ok := typeMap[t]; ok {
			log.Fatal("volume type %+q is claimed by multiple VolumeTypes")
		}
		typeMap[t] = factory
	}

	var mapList []map[string]interface{}
	err := json.Unmarshal(data, &mapList)
	if err != nil {
		return err
	}
	for _, mapIn := range mapList {
		typeIn, ok := mapIn["Type"].(string)
		if !ok {
			return fmt.Errorf("invalid volume type %+v", mapIn["Type"])
		}
		factory, ok := typeMap[typeIn]
		if !ok {
			return fmt.Errorf("unsupported volume type %+q", typeIn)
		}
		data, err := json.Marshal(mapIn)
		if err != nil {
			return err
		}
		vol := factory()
		err = json.Unmarshal(data, vol)
		if err != nil {
			return err
		}
		*vols = append(*vols, vol)
	}
	return nil
}

// MarshalJSON adds a "Type" field to each volume corresponding to its
// Type().
func (vl *VolumeList) MarshalJSON() ([]byte, error) {
	data := []byte{'['}
	for _, vs := range *vl {
		j, err := json.Marshal(vs)
		if err != nil {
			return nil, err
		}
		if len(data) > 1 {
			data = append(data, byte(','))
		}
		t, err := json.Marshal(vs.Type())
		if err != nil {
			panic(err)
		}
		data = append(data, j[0])
		data = append(data, []byte(`"Type":`)...)
		data = append(data, t...)
		data = append(data, byte(','))
		data = append(data, j[1:]...)
	}
	return append(data, byte(']')), nil
}
