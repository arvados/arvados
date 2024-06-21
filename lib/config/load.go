// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package config

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"dario.cat/mergo"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"github.com/ghodss/yaml"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"golang.org/x/sys/unix"
)

//go:embed config.default.yml
var DefaultYAML []byte

var ErrNoClustersDefined = errors.New("config does not define any clusters")

type Loader struct {
	Stdin          io.Reader
	Logger         logrus.FieldLogger
	SkipDeprecated bool // Don't load deprecated config keys
	SkipLegacy     bool // Don't load legacy config files
	SkipAPICalls   bool // Don't do checks that call RailsAPI/controller

	Path                    string
	KeepstorePath           string
	KeepWebPath             string
	CrunchDispatchSlurmPath string
	WebsocketPath           string
	KeepproxyPath           string
	KeepBalancePath         string

	configdata []byte
	// UTC time for configdata: either the modtime of the file we
	// read configdata from, or the time when we read configdata
	// from a pipe.
	sourceTimestamp time.Time
	// UTC time when configdata was read.
	loadTimestamp time.Time
}

// NewLoader returns a new Loader with Stdin and Logger set to the
// given values, and all config paths set to their default values.
func NewLoader(stdin io.Reader, logger logrus.FieldLogger) *Loader {
	ldr := &Loader{Stdin: stdin, Logger: logger}
	// Calling SetupFlags on a throwaway FlagSet has the side
	// effect of assigning default values to the configurable
	// fields.
	ldr.SetupFlags(flag.NewFlagSet("", flag.ContinueOnError))
	return ldr
}

// SetupFlags configures a flagset so arguments like -config X can be
// used to change the loader's Path fields.
//
//	ldr := NewLoader(os.Stdin, logrus.New())
//	flagset := flag.NewFlagSet("", flag.ContinueOnError)
//	ldr.SetupFlags(flagset)
//	// ldr.Path == "/etc/arvados/config.yml"
//	flagset.Parse([]string{"-config", "/tmp/c.yaml"})
//	// ldr.Path == "/tmp/c.yaml"
func (ldr *Loader) SetupFlags(flagset *flag.FlagSet) {
	flagset.StringVar(&ldr.Path, "config", arvados.DefaultConfigFile, "Site configuration `file` (default may be overridden by setting an ARVADOS_CONFIG environment variable)")
	if !ldr.SkipLegacy {
		flagset.StringVar(&ldr.KeepstorePath, "legacy-keepstore-config", defaultKeepstoreConfigPath, "Legacy keepstore configuration `file`")
		flagset.StringVar(&ldr.KeepWebPath, "legacy-keepweb-config", defaultKeepWebConfigPath, "Legacy keep-web configuration `file`")
		flagset.StringVar(&ldr.CrunchDispatchSlurmPath, "legacy-crunch-dispatch-slurm-config", defaultCrunchDispatchSlurmConfigPath, "Legacy crunch-dispatch-slurm configuration `file`")
		flagset.StringVar(&ldr.WebsocketPath, "legacy-ws-config", defaultWebsocketConfigPath, "Legacy arvados-ws configuration `file`")
		flagset.StringVar(&ldr.KeepproxyPath, "legacy-keepproxy-config", defaultKeepproxyConfigPath, "Legacy keepproxy configuration `file`")
		flagset.StringVar(&ldr.KeepBalancePath, "legacy-keepbalance-config", defaultKeepBalanceConfigPath, "Legacy keep-balance configuration `file`")
		flagset.BoolVar(&ldr.SkipLegacy, "skip-legacy", false, "Don't load legacy config files")
	}
}

// MungeLegacyConfigArgs checks args for a -config flag whose argument
// is a regular file (or a symlink to one), but doesn't have a
// top-level "Clusters" key and therefore isn't a valid cluster
// configuration file. If it finds such a flag, it replaces -config
// with legacyConfigArg (e.g., "-legacy-keepstore-config").
//
// This is used by programs that still need to accept "-config" as a
// way to specify a per-component config file until their config has
// been migrated.
//
// If any errors are encountered while reading or parsing a config
// file, the given args are not munged. We presume the same errors
// will be encountered again and reported later on when trying to load
// cluster configuration from the same file, regardless of which
// struct we end up using.
func (ldr *Loader) MungeLegacyConfigArgs(lgr logrus.FieldLogger, args []string, legacyConfigArg string) []string {
	munged := append([]string(nil), args...)
	for i := 0; i < len(args); i++ {
		if !strings.HasPrefix(args[i], "-") || strings.SplitN(strings.TrimPrefix(args[i], "-"), "=", 2)[0] != "config" {
			continue
		}
		var operand string
		if strings.Contains(args[i], "=") {
			operand = strings.SplitN(args[i], "=", 2)[1]
		} else if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			i++
			operand = args[i]
		} else {
			continue
		}
		if fi, err := os.Stat(operand); err != nil || !fi.Mode().IsRegular() {
			continue
		}
		f, err := os.Open(operand)
		if err != nil {
			continue
		}
		defer f.Close()
		buf, err := ioutil.ReadAll(f)
		if err != nil {
			continue
		}
		var cfg arvados.Config
		err = yaml.Unmarshal(buf, &cfg)
		if err != nil {
			continue
		}
		if len(cfg.Clusters) == 0 {
			lgr.Warnf("%s is not a cluster config file -- interpreting %s as %s (please migrate your config!)", operand, "-config", legacyConfigArg)
			if operand == args[i] {
				munged[i-1] = legacyConfigArg
			} else {
				munged[i] = legacyConfigArg + "=" + operand
			}
		}
	}

	// Disable legacy config loading for components other than the
	// one that was specified
	if legacyConfigArg != "-legacy-keepstore-config" {
		ldr.KeepstorePath = ""
	}
	if legacyConfigArg != "-legacy-crunch-dispatch-slurm-config" {
		ldr.CrunchDispatchSlurmPath = ""
	}
	if legacyConfigArg != "-legacy-ws-config" {
		ldr.WebsocketPath = ""
	}
	if legacyConfigArg != "-legacy-keepweb-config" {
		ldr.KeepWebPath = ""
	}
	if legacyConfigArg != "-legacy-keepproxy-config" {
		ldr.KeepproxyPath = ""
	}
	if legacyConfigArg != "-legacy-keepbalance-config" {
		ldr.KeepBalancePath = ""
	}

	return munged
}

func (ldr *Loader) loadBytes(path string) (buf []byte, sourceTime, loadTime time.Time, err error) {
	loadTime = time.Now().UTC()
	if path == "-" {
		buf, err = ioutil.ReadAll(ldr.Stdin)
		sourceTime = loadTime
		return
	}
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return
	}
	sourceTime = fi.ModTime().UTC()
	buf, err = ioutil.ReadAll(f)
	return
}

func (ldr *Loader) Load() (*arvados.Config, error) {
	if ldr.configdata == nil {
		buf, sourceTime, loadTime, err := ldr.loadBytes(ldr.Path)
		if err != nil {
			return nil, err
		}
		ldr.configdata = buf
		ldr.sourceTimestamp = sourceTime
		ldr.loadTimestamp = loadTime
	}

	// FIXME: We should reject YAML if the same key is used twice
	// in a map/object, like {foo: bar, foo: baz}. Maybe we'll get
	// this fixed free when we upgrade ghodss/yaml to a version
	// that uses go-yaml v3.

	// Load the config into a dummy map to get the cluster ID
	// keys, discarding the values; then set up defaults for each
	// cluster ID; then load the real config on top of the
	// defaults.
	var dummy struct {
		Clusters map[string]struct{}
	}
	err := yaml.Unmarshal(ldr.configdata, &dummy)
	if err != nil {
		return nil, err
	}
	if len(dummy.Clusters) == 0 {
		return nil, ErrNoClustersDefined
	}

	// We can't merge deep structs here; instead, we unmarshal the
	// default & loaded config files into generic maps, merge
	// those, and then json-encode+decode the result into the
	// config struct type.
	var merged map[string]interface{}
	for id := range dummy.Clusters {
		var src map[string]interface{}
		err = yaml.Unmarshal(bytes.Replace(DefaultYAML, []byte(" xxxxx:"), []byte(" "+id+":"), -1), &src)
		if err != nil {
			return nil, fmt.Errorf("loading defaults for %s: %s", id, err)
		}
		err = mergo.Merge(&merged, src, mergo.WithOverride)
		if err != nil {
			return nil, fmt.Errorf("merging defaults for %s: %s", id, err)
		}
	}
	var src map[string]interface{}
	err = yaml.Unmarshal(ldr.configdata, &src)
	if err != nil {
		return nil, fmt.Errorf("loading config data: %s", err)
	}
	ldr.logExtraKeys(merged, src, "")
	removeSampleKeys(merged)
	// We merge the loaded config into the default, overriding any existing keys.
	// Make sure we do not override a default with a key that has a 'null' value.
	removeNullKeys(src)
	err = mergo.Merge(&merged, src, mergo.WithOverride)
	if err != nil {
		return nil, fmt.Errorf("merging config data: %s", err)
	}

	// map[string]interface{} => json => arvados.Config
	var cfg arvados.Config
	var errEnc error
	pr, pw := io.Pipe()
	go func() {
		errEnc = json.NewEncoder(pw).Encode(merged)
		pw.Close()
	}()
	err = json.NewDecoder(pr).Decode(&cfg)
	if errEnc != nil {
		err = errEnc
	}
	if err != nil {
		return nil, fmt.Errorf("transcoding config data: %s", err)
	}

	var loadFuncs []func(*arvados.Config) error
	if !ldr.SkipDeprecated {
		loadFuncs = append(loadFuncs,
			ldr.applyDeprecatedConfig,
			ldr.applyDeprecatedVolumeDriverParameters,
		)
	}
	if !ldr.SkipLegacy {
		// legacy file is required when either:
		// * a non-default location was specified
		// * no primary config was loaded, and this is the
		// legacy config file for the current component
		loadFuncs = append(loadFuncs,
			ldr.loadOldEnvironmentVariables,
			ldr.loadOldKeepstoreConfig,
			ldr.loadOldKeepWebConfig,
			ldr.loadOldCrunchDispatchSlurmConfig,
			ldr.loadOldWebsocketConfig,
			ldr.loadOldKeepproxyConfig,
			ldr.loadOldKeepBalanceConfig,
		)
	}
	loadFuncs = append(loadFuncs,
		ldr.setImplicitStorageClasses,
		ldr.setLoopbackInstanceType,
	)
	for _, f := range loadFuncs {
		err = f(&cfg)
		if err != nil {
			return nil, err
		}
	}

	// Preprocess/automate some configs
	for id, cc := range cfg.Clusters {
		ldr.autofillPreemptible("Clusters."+id, &cc)

		if strings.Count(cc.Users.AnonymousUserToken, "/") == 3 {
			// V2 token, strip it to just a secret
			tmp := strings.Split(cc.Users.AnonymousUserToken, "/")
			cc.Users.AnonymousUserToken = tmp[2]
		}

		cfg.Clusters[id] = cc
	}

	// Check for known mistakes
	for id, cc := range cfg.Clusters {
		for remote := range cc.RemoteClusters {
			if remote == "*" || remote == "SAMPLE" {
				continue
			}
			err = ldr.checkClusterID(fmt.Sprintf("Clusters.%s.RemoteClusters.%s", id, remote), remote, true)
			if err != nil {
				return nil, err
			}
		}
		for _, err = range []error{
			ldr.checkClusterID(fmt.Sprintf("Clusters.%s", id), id, false),
			ldr.checkClusterID(fmt.Sprintf("Clusters.%s.Login.LoginCluster", id), cc.Login.LoginCluster, true),
			ldr.checkToken(fmt.Sprintf("Clusters.%s.ManagementToken", id), cc.ManagementToken, true, false),
			ldr.checkToken(fmt.Sprintf("Clusters.%s.SystemRootToken", id), cc.SystemRootToken, true, false),
			ldr.checkToken(fmt.Sprintf("Clusters.%s.Users.AnonymousUserToken", id), cc.Users.AnonymousUserToken, false, true),
			ldr.checkToken(fmt.Sprintf("Clusters.%s.Collections.BlobSigningKey", id), cc.Collections.BlobSigningKey, true, false),
			checkKeyConflict(fmt.Sprintf("Clusters.%s.PostgreSQL.Connection", id), cc.PostgreSQL.Connection),
			ldr.checkEnum("Containers.LocalKeepLogsToContainerLog", cc.Containers.LocalKeepLogsToContainerLog, "none", "all", "errors"),
			ldr.checkEmptyKeepstores(cc),
			ldr.checkUnlistedKeepstores(cc),
			ldr.checkLocalKeepBlobBuffers(cc),
			ldr.checkStorageClasses(cc),
			ldr.checkCUDAVersions(cc),
			// TODO: check non-empty Rendezvous on
			// services other than Keepstore
		} {
			if err != nil {
				return nil, err
			}
		}
	}
	cfg.SourceTimestamp = ldr.sourceTimestamp
	cfg.SourceSHA256 = fmt.Sprintf("%x", sha256.Sum256(ldr.configdata))
	return &cfg, nil
}

var acceptableClusterIDRe = regexp.MustCompile(`^[a-z0-9]{5}$`)

func (ldr *Loader) checkClusterID(label, clusterID string, emptyStringOk bool) error {
	if emptyStringOk && clusterID == "" {
		return nil
	} else if !acceptableClusterIDRe.MatchString(clusterID) {
		return fmt.Errorf("%s: cluster ID should be 5 lowercase alphanumeric characters", label)
	}
	return nil
}

var acceptableTokenRe = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
var acceptableTokenLength = 32

func (ldr *Loader) checkToken(label, token string, mandatory bool, acceptV2 bool) error {
	if len(token) == 0 {
		if !mandatory {
			// when a token is not mandatory, the acceptable length and content is only checked if its length is non-zero
			return nil
		} else {
			if ldr.Logger != nil {
				ldr.Logger.Warnf("%s: secret token is not set (use %d+ random characters from a-z, A-Z, 0-9)", label, acceptableTokenLength)
			}
		}
	} else if !acceptableTokenRe.MatchString(token) {
		if !acceptV2 {
			return fmt.Errorf("%s: unacceptable characters in token (only a-z, A-Z, 0-9 are acceptable)", label)
		}
		// Test for a proper V2 token
		tmp := strings.SplitN(token, "/", 3)
		if len(tmp) != 3 {
			return fmt.Errorf("%s: unacceptable characters in token (only a-z, A-Z, 0-9 are acceptable)", label)
		}
		if !strings.HasPrefix(token, "v2/") {
			return fmt.Errorf("%s: unacceptable characters in token (only a-z, A-Z, 0-9 are acceptable)", label)
		}
		if !acceptableTokenRe.MatchString(tmp[2]) {
			return fmt.Errorf("%s: unacceptable characters in V2 token secret (only a-z, A-Z, 0-9 are acceptable)", label)
		}
		if len(tmp[2]) < acceptableTokenLength {
			ldr.Logger.Warnf("%s: secret is too short (should be at least %d characters)", label, acceptableTokenLength)
		}
	} else if len(token) < acceptableTokenLength {
		if ldr.Logger != nil {
			ldr.Logger.Warnf("%s: token is too short (should be at least %d characters)", label, acceptableTokenLength)
		}
	}
	return nil
}

func (ldr *Loader) checkEnum(label, value string, accepted ...string) error {
	for _, s := range accepted {
		if s == value {
			return nil
		}
	}
	return fmt.Errorf("%s: unacceptable value %q: must be one of %q", label, value, accepted)
}

func (ldr *Loader) setLoopbackInstanceType(cfg *arvados.Config) error {
	for id, cc := range cfg.Clusters {
		if !cc.Containers.CloudVMs.Enable || cc.Containers.CloudVMs.Driver != "loopback" {
			continue
		}
		if len(cc.InstanceTypes) == 1 {
			continue
		}
		if len(cc.InstanceTypes) > 1 {
			return fmt.Errorf("Clusters.%s.InstanceTypes: cannot use multiple InstanceTypes with loopback driver", id)
		}
		// No InstanceTypes configured. Fill in implicit
		// default.
		hostram, err := getHostRAM()
		if err != nil {
			return err
		}
		scratch, err := getFilesystemSize(os.TempDir())
		if err != nil {
			return err
		}
		cc.InstanceTypes = arvados.InstanceTypeMap{"localhost": {
			Name:            "localhost",
			ProviderType:    "localhost",
			VCPUs:           runtime.NumCPU(),
			RAM:             hostram,
			Scratch:         scratch,
			IncludedScratch: scratch,
			Price:           1.0,
		}}
		cfg.Clusters[id] = cc
	}
	return nil
}

func getFilesystemSize(path string) (arvados.ByteSize, error) {
	var stat unix.Statfs_t
	err := unix.Statfs(path, &stat)
	if err != nil {
		return 0, err
	}
	return arvados.ByteSize(stat.Blocks * uint64(stat.Bsize)), nil
}

var reMemTotal = regexp.MustCompile(`(^|\n)MemTotal: *(\d+) kB\n`)

func getHostRAM() (arvados.ByteSize, error) {
	buf, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	m := reMemTotal.FindSubmatch(buf)
	if m == nil {
		return 0, errors.New("error parsing /proc/meminfo: no MemTotal")
	}
	kb, err := strconv.ParseInt(string(m[2]), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("error parsing /proc/meminfo: %q: %w", m[2], err)
	}
	return arvados.ByteSize(kb) * 1024, nil
}

func (ldr *Loader) setImplicitStorageClasses(cfg *arvados.Config) error {
cluster:
	for id, cc := range cfg.Clusters {
		if len(cc.StorageClasses) > 0 {
			continue cluster
		}
		for _, vol := range cc.Volumes {
			if len(vol.StorageClasses) > 0 {
				continue cluster
			}
		}
		// No explicit StorageClasses config info at all; fill
		// in implicit defaults.
		for id, vol := range cc.Volumes {
			vol.StorageClasses = map[string]bool{"default": true}
			cc.Volumes[id] = vol
		}
		cc.StorageClasses = map[string]arvados.StorageClassConfig{"default": {Default: true}}
		cfg.Clusters[id] = cc
	}
	return nil
}

func (ldr *Loader) checkLocalKeepBlobBuffers(cc arvados.Cluster) error {
	kbb := cc.Containers.LocalKeepBlobBuffersPerVCPU
	if kbb == 0 {
		return nil
	}
	for uuid, vol := range cc.Volumes {
		if len(vol.AccessViaHosts) > 0 {
			ldr.Logger.Warnf("LocalKeepBlobBuffersPerVCPU is %d but will not be used because at least one volume (%s) uses AccessViaHosts -- suggest changing to 0", kbb, uuid)
			return nil
		}
		if !vol.ReadOnly && vol.Replication < cc.Collections.DefaultReplication {
			ldr.Logger.Warnf("LocalKeepBlobBuffersPerVCPU is %d but will not be used because at least one volume (%s) has lower replication than DefaultReplication (%d < %d) -- suggest changing to 0", kbb, uuid, vol.Replication, cc.Collections.DefaultReplication)
			return nil
		}
	}
	return nil
}

func (ldr *Loader) checkStorageClasses(cc arvados.Cluster) error {
	classOnVolume := map[string]bool{}
	for volid, vol := range cc.Volumes {
		if len(vol.StorageClasses) == 0 {
			return fmt.Errorf("%s: volume has no StorageClasses listed", volid)
		}
		for classid := range vol.StorageClasses {
			if _, ok := cc.StorageClasses[classid]; !ok {
				return fmt.Errorf("%s: volume refers to storage class %q that is not defined in StorageClasses", volid, classid)
			}
			classOnVolume[classid] = true
		}
	}
	haveDefault := false
	for classid, sc := range cc.StorageClasses {
		if !classOnVolume[classid] && len(cc.Volumes) > 0 {
			ldr.Logger.Warnf("there are no volumes providing storage class %q", classid)
		}
		if sc.Default {
			haveDefault = true
		}
	}
	if !haveDefault {
		return fmt.Errorf("there is no default storage class (at least one entry in StorageClasses must have Default: true)")
	}
	return nil
}

func (ldr *Loader) checkCUDAVersions(cc arvados.Cluster) error {
	for _, it := range cc.InstanceTypes {
		if it.CUDA.DeviceCount == 0 {
			continue
		}

		_, err := strconv.ParseFloat(it.CUDA.DriverVersion, 64)
		if err != nil {
			return fmt.Errorf("InstanceType %q has invalid CUDA.DriverVersion %q, expected format X.Y (%v)", it.Name, it.CUDA.DriverVersion, err)
		}
		_, err = strconv.ParseFloat(it.CUDA.HardwareCapability, 64)
		if err != nil {
			return fmt.Errorf("InstanceType %q has invalid CUDA.HardwareCapability %q, expected format X.Y (%v)", it.Name, it.CUDA.HardwareCapability, err)
		}
	}
	return nil
}

func checkKeyConflict(label string, m map[string]string) error {
	saw := map[string]bool{}
	for k := range m {
		k = strings.ToLower(k)
		if saw[k] {
			return fmt.Errorf("%s: multiple entries for %q (fix by using same capitalization as default/example file)", label, k)
		}
		saw[k] = true
	}
	return nil
}

func removeNullKeys(m map[string]interface{}) {
	for k, v := range m {
		if v == nil {
			delete(m, k)
		}
		if v, _ := v.(map[string]interface{}); v != nil {
			removeNullKeys(v)
		}
	}
}

func removeSampleKeys(m map[string]interface{}) {
	delete(m, "SAMPLE")
	for _, v := range m {
		if v, _ := v.(map[string]interface{}); v != nil {
			removeSampleKeys(v)
		}
	}
}

func (ldr *Loader) logExtraKeys(expected, supplied map[string]interface{}, prefix string) {
	if ldr.Logger == nil {
		return
	}
	for k, vsupp := range supplied {
		if k == "SAMPLE" {
			// entry will be dropped in removeSampleKeys anyway
			continue
		}
		vexp, ok := expected[k]
		if expected["SAMPLE"] != nil {
			// use the SAMPLE entry's keys as the
			// "expected" map when checking vsupp
			// recursively.
			vexp = expected["SAMPLE"]
		} else if !ok {
			// check for a case-insensitive match
			hint := ""
			for ek := range expected {
				if strings.EqualFold(k, ek) {
					hint = " (perhaps you meant " + ek + "?)"
					// If we don't delete this, it
					// will end up getting merged,
					// unpredictably
					// merging/overriding the
					// default.
					delete(supplied, k)
					break
				}
			}
			ldr.Logger.Warnf("deprecated or unknown config entry: %s%s%s", prefix, k, hint)
			continue
		}
		if vsupp, ok := vsupp.(map[string]interface{}); !ok {
			// if vsupp is a map but vexp isn't map, this
			// will be caught elsewhere; see TestBadType.
			continue
		} else if vexp, ok := vexp.(map[string]interface{}); !ok {
			ldr.Logger.Warnf("unexpected object in config entry: %s%s", prefix, k)
		} else {
			ldr.logExtraKeys(vexp, vsupp, prefix+k+".")
		}
	}
}

func (ldr *Loader) autofillPreemptible(label string, cc *arvados.Cluster) {
	if factor := cc.Containers.PreemptiblePriceFactor; factor > 0 {
		for name, it := range cc.InstanceTypes {
			if !it.Preemptible {
				it.Preemptible = true
				it.Price = it.Price * factor
				it.Name = name + ".preemptible"
				if it2, exists := cc.InstanceTypes[it.Name]; exists && it2 != it {
					ldr.Logger.Warnf("%s.InstanceTypes[%s]: already exists, so not automatically adding a preemptible variant of %s", label, it.Name, name)
					continue
				}
				cc.InstanceTypes[it.Name] = it
			}
		}
	}

}

// RegisterMetrics registers metrics showing the timestamp and content
// hash of the currently loaded config.
//
// Must not be called more than once for a given registry. Must not be
// called before Load(). Metrics are not updated by subsequent calls
// to Load().
func (ldr *Loader) RegisterMetrics(reg *prometheus.Registry) {
	hash := fmt.Sprintf("%x", sha256.Sum256(ldr.configdata))
	vec := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "arvados",
		Subsystem: "config",
		Name:      "source_timestamp_seconds",
		Help:      "Timestamp of config file when it was loaded.",
	}, []string{"sha256"})
	vec.WithLabelValues(hash).Set(float64(ldr.sourceTimestamp.UnixNano()) / 1e9)
	reg.MustRegister(vec)

	vec = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "arvados",
		Subsystem: "config",
		Name:      "load_timestamp_seconds",
		Help:      "Time when config file was loaded.",
	}, []string{"sha256"})
	vec.WithLabelValues(hash).Set(float64(ldr.loadTimestamp.UnixNano()) / 1e9)
	reg.MustRegister(vec)
}

// Load an SSH private key from the given confvalue, which is either
// the literal key or an absolute path to a file containing the key.
func LoadSSHKey(confvalue string) (ssh.Signer, error) {
	if fnm := strings.TrimPrefix(confvalue, "file://"); fnm != confvalue && strings.HasPrefix(fnm, "/") {
		keydata, err := os.ReadFile(fnm)
		if err != nil {
			return nil, err
		}
		return ssh.ParsePrivateKey(keydata)
	} else {
		return ssh.ParsePrivateKey([]byte(confvalue))
	}
}
