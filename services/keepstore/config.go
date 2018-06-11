// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/stats"
	"github.com/Sirupsen/logrus"
	"github.com/golang/protobuf/jsonpb"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Config struct {
	Debug  bool
	Listen string

	LogFormat string

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
	PullWorkers         int
	TrashWorkers        int
	EmptyTrashWorkers   int
	TLSCertificateFile  string
	TLSKeyFile          string

	Volumes VolumeList

	blobSigningKey  []byte
	systemAuthToken string
	debugLogf       func(string, ...interface{})

	ManagementToken string `doc: The secret key that must be provided by monitoring services
wishing to access the health check endpoint (/_health).`

	metrics
}

var (
	theConfig = DefaultConfig()
	formatter = map[string]logrus.Formatter{
		"text": &logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: rfc3339NanoFixed,
		},
		"json": &logrus.JSONFormatter{
			TimestampFormat: rfc3339NanoFixed,
		},
	}
	log = logrus.StandardLogger()
)

const rfc3339NanoFixed = "2006-01-02T15:04:05.000000000Z07:00"

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Listen:             ":25107",
		LogFormat:          "json",
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
	if cfg.Debug {
		log.Level = logrus.DebugLevel
		cfg.debugLogf = log.Printf
		cfg.debugLogf("debugging enabled")
	} else {
		log.Level = logrus.InfoLevel
		cfg.debugLogf = func(string, ...interface{}) {}
	}

	f := formatter[strings.ToLower(cfg.LogFormat)]
	if f == nil {
		return fmt.Errorf(`unsupported log format %q (try "text" or "json")`, cfg.LogFormat)
	}
	log.Formatter = f

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

type metrics struct {
	registry     *prometheus.Registry
	reqDuration  *prometheus.SummaryVec
	timeToStatus *prometheus.SummaryVec
	exportProm   http.Handler
}

func (*metrics) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (m *metrics) Fire(ent *logrus.Entry) error {
	if tts, ok := ent.Data["timeToStatus"].(stats.Duration); !ok {
	} else if method, ok := ent.Data["reqMethod"].(string); !ok {
	} else if code, ok := ent.Data["respStatusCode"].(int); !ok {
	} else {
		m.timeToStatus.WithLabelValues(strconv.Itoa(code), strings.ToLower(method)).Observe(time.Duration(tts).Seconds())
	}
	return nil
}

func (m *metrics) setup() {
	m.registry = prometheus.NewRegistry()
	m.timeToStatus = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name: "time_to_status_seconds",
		Help: "Summary of request TTFB.",
	}, []string{"code", "method"})
	m.reqDuration = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name: "request_duration_seconds",
		Help: "Summary of request duration.",
	}, []string{"code", "method"})
	m.registry.MustRegister(m.timeToStatus)
	m.registry.MustRegister(m.reqDuration)
	m.exportProm = promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{
		ErrorLog: log,
	})
	log.AddHook(m)
}

func (m *metrics) exportJSON(w http.ResponseWriter, req *http.Request) {
	jm := jsonpb.Marshaler{Indent: "  "}
	mfs, _ := m.registry.Gather()
	w.Write([]byte{'['})
	for i, mf := range mfs {
		if i > 0 {
			w.Write([]byte{','})
		}
		jm.Marshal(w, mf)
	}
	w.Write([]byte{']'})
}

func (m *metrics) Instrument(next http.Handler) http.Handler {
	return promhttp.InstrumentHandlerDuration(m.reqDuration, next)
}

// VolumeTypes is built up by init() funcs in the source files that
// define the volume types.
var VolumeTypes = []func() VolumeWithExamples{}

type VolumeList []Volume

// UnmarshalJSON -- given an array of objects -- deserializes each
// object as the volume type indicated by the object's Type field.
func (vl *VolumeList) UnmarshalJSON(data []byte) error {
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
		*vl = append(*vl, vol)
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
