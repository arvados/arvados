// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

// Package crunchstat reports resource usage (CPU, memory, disk,
// network) for a cgroup.
package crunchstat

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// crunchstat collects all memory statistics, but only reports these.
var memoryStats = [...]string{"cache", "swap", "pgmajfault", "rss"}

type logPrinter interface {
	Printf(fmt string, args ...interface{})
}

// A Reporter gathers statistics for a cgroup and writes them to a
// log.Logger.
type Reporter struct {
	// Func that returns the pid of a process inside the desired
	// cgroup. Reporter will call Pid periodically until it
	// returns a positive number, then start reporting stats for
	// the cgroup that process belongs to.
	//
	// Pid is used when cgroups v2 is available. For cgroups v1,
	// see below.
	Pid func() int

	// Interval between samples. Must be positive.
	PollPeriod time.Duration

	// Temporary directory, will be monitored for available, used
	// & total space.
	TempDir string

	// Where to write statistics. Must not be nil.
	Logger logPrinter

	// When stats cross thresholds configured in the fields below,
	// they are reported to this logger.
	ThresholdLogger logPrinter

	// MemThresholds maps memory stat names to slices of thresholds.
	// When the corresponding stat exceeds a threshold, that will be logged.
	MemThresholds map[string][]Threshold

	// Filesystem to read /proc entries and cgroup stats from.
	// Non-nil for testing, nil for real root filesystem.
	FS fs.FS

	// Enable debug messages.
	Debug bool

	// available cgroup hierarchies
	statFiles struct {
		cpusetCpus        string // v1,v2 (via /proc/$PID/cpuset)
		cpuacctStat       string // v1 (via /proc/$PID/cgroup => cpuacct)
		cpuStat           string // v2
		ioServiceBytes    string // v1 (via /proc/$PID/cgroup => blkio)
		ioStat            string // v2
		memoryStat        string // v1 and v2 (but v2 is missing some entries)
		memoryCurrent     string // v2
		memorySwapCurrent string // v2
		netDev            string // /proc/$PID/net/dev
	}

	kernelPageSize      int64
	lastNetSample       map[string]ioSample
	lastDiskIOSample    map[string]ioSample
	lastCPUSample       cpuSample
	lastDiskSpaceSample diskSpaceSample
	lastMemSample       memSample
	maxDiskSpaceSample  diskSpaceSample
	maxMemSample        map[memoryKey]int64

	// process returned by Pid(), whose cgroup stats we are
	// reporting
	pid int

	// individual processes whose memory size we are reporting
	reportPIDs   map[string]int
	reportPIDsMu sync.Mutex

	done    chan struct{} // closed when we should stop reporting
	ready   chan struct{} // have pid and stat files
	flushed chan struct{} // closed when we have made our last report
}

type Threshold struct {
	percentage int64
	threshold  int64
	total      int64
}

func NewThresholdFromPercentage(total int64, percentage int64) Threshold {
	return Threshold{
		percentage: percentage,
		threshold:  total * percentage / 100,
		total:      total,
	}
}

func NewThresholdsFromPercentages(total int64, percentages []int64) (thresholds []Threshold) {
	for _, percentage := range percentages {
		thresholds = append(thresholds, NewThresholdFromPercentage(total, percentage))
	}
	return
}

// memoryKey is a key into Reporter.maxMemSample.
// Initialize it with just statName to get the host/cgroup maximum.
// Initialize it with all fields to get that process' maximum.
type memoryKey struct {
	processID   int
	processName string
	statName    string
}

// Start starts monitoring in a new goroutine, and returns
// immediately.
//
// The monitoring goroutine waits for a non-empty CIDFile to appear
// (unless CID is non-empty). Then it waits for the accounting files
// to appear for the monitored container. Then it collects and reports
// statistics until Stop is called.
//
// Callers should not call Start more than once.
//
// Callers should not modify public data fields after calling Start.
func (r *Reporter) Start() {
	r.done = make(chan struct{})
	r.ready = make(chan struct{})
	r.flushed = make(chan struct{})
	if r.FS == nil {
		r.FS = os.DirFS("/")
	}
	go r.run()
}

// ReportPID starts reporting stats for a specified process.
func (r *Reporter) ReportPID(name string, pid int) {
	r.reportPIDsMu.Lock()
	defer r.reportPIDsMu.Unlock()
	if r.reportPIDs == nil {
		r.reportPIDs = map[string]int{name: pid}
	} else {
		r.reportPIDs[name] = pid
	}
}

// Stop reporting. Do not call more than once, or before calling
// Start.
//
// Nothing will be logged after Stop returns unless you call a Log* method.
func (r *Reporter) Stop() {
	close(r.done)
	<-r.flushed
}

var v1keys = map[string]bool{
	"blkio":   true,
	"cpuacct": true,
	"cpuset":  true,
	"memory":  true,
}

// Find cgroup hierarchies in /proc/mounts, e.g.,
//
//	{
//		"blkio": "/sys/fs/cgroup/blkio",
//		"unified": "/sys/fs/cgroup/unified",
//	}
func (r *Reporter) cgroupMounts() map[string]string {
	procmounts, err := fs.ReadFile(r.FS, "proc/mounts")
	if err != nil {
		r.Logger.Printf("error reading /proc/mounts: %s", err)
		return nil
	}
	mounts := map[string]string{}
	for _, line := range bytes.Split(procmounts, []byte{'\n'}) {
		fields := bytes.SplitN(line, []byte{' '}, 6)
		if len(fields) != 6 {
			continue
		}
		switch string(fields[2]) {
		case "cgroup2":
			// cgroup /sys/fs/cgroup/unified cgroup2 rw,nosuid,nodev,noexec,relatime 0 0
			mounts["unified"] = string(fields[1])
		case "cgroup":
			// cgroup /sys/fs/cgroup/blkio cgroup rw,nosuid,nodev,noexec,relatime,blkio 0 0
			options := bytes.Split(fields[3], []byte{','})
			for _, option := range options {
				option := string(option)
				if v1keys[option] {
					mounts[option] = string(fields[1])
					break
				}
			}
		}
	}
	return mounts
}

// generate map of cgroup controller => path for r.pid.
//
// the "unified" controller represents cgroups v2.
func (r *Reporter) cgroupPaths(mounts map[string]string) map[string]string {
	if len(mounts) == 0 {
		return nil
	}
	procdir := fmt.Sprintf("proc/%d", r.pid)
	buf, err := fs.ReadFile(r.FS, procdir+"/cgroup")
	if err != nil {
		r.Logger.Printf("error reading cgroup file: %s", err)
		return nil
	}
	paths := map[string]string{}
	for _, line := range bytes.Split(buf, []byte{'\n'}) {
		// The entry for cgroup v2 is always in the format
		// "0::$PATH" --
		// https://docs.kernel.org/admin-guide/cgroup-v2.html
		if bytes.HasPrefix(line, []byte("0::/")) && mounts["unified"] != "" {
			paths["unified"] = mounts["unified"] + string(line[3:])
			continue
		}
		// cgroups v1 entries look like
		// "6:cpu,cpuacct:/user.slice"
		fields := bytes.SplitN(line, []byte{':'}, 3)
		if len(fields) != 3 {
			continue
		}
		for _, key := range bytes.Split(fields[1], []byte{','}) {
			key := string(key)
			if mounts[key] != "" {
				paths[key] = mounts[key] + string(fields[2])
			}
		}
	}
	// In unified mode, /proc/$PID/cgroup doesn't have a cpuset
	// entry, but we still need it -- there's no cpuset.cpus file
	// in the cgroup2 subtree indicated by the 0::$PATH entry. We
	// have to get the right path from /proc/$PID/cpuset.
	if _, found := paths["cpuset"]; !found && mounts["unified"] != "" {
		buf, _ := fs.ReadFile(r.FS, procdir+"/cpuset")
		cpusetPath := string(bytes.TrimRight(buf, "\n"))
		paths["cpuset"] = mounts["unified"] + cpusetPath
	}
	return paths
}

func (r *Reporter) findStatFiles() {
	mounts := r.cgroupMounts()
	paths := r.cgroupPaths(mounts)
	done := map[*string]bool{}
	for _, try := range []struct {
		statFile *string
		pathkey  string
		file     string
	}{
		{&r.statFiles.cpusetCpus, "cpuset", "cpuset.cpus.effective"},
		{&r.statFiles.cpusetCpus, "cpuset", "cpuset.cpus"},
		{&r.statFiles.cpuacctStat, "cpuacct", "cpuacct.stat"},
		{&r.statFiles.cpuStat, "unified", "cpu.stat"},
		// blkio.throttle.io_service_bytes must precede
		// blkio.io_service_bytes -- on ubuntu1804, the latter
		// is present but reports 0
		{&r.statFiles.ioServiceBytes, "blkio", "blkio.throttle.io_service_bytes"},
		{&r.statFiles.ioServiceBytes, "blkio", "blkio.io_service_bytes"},
		{&r.statFiles.ioStat, "unified", "io.stat"},
		{&r.statFiles.memoryStat, "unified", "memory.stat"},
		{&r.statFiles.memoryStat, "memory", "memory.stat"},
		{&r.statFiles.memoryCurrent, "unified", "memory.current"},
		{&r.statFiles.memorySwapCurrent, "unified", "memory.swap.current"},
	} {
		startpath, ok := paths[try.pathkey]
		if !ok || done[try.statFile] {
			continue
		}
		// /proc/$PID/cgroup says cgroup path is
		// /exa/mple/exa/mple, however, sometimes the file we
		// need is not under that path, it's only available in
		// a parent cgroup's dir.  So we start at
		// /sys/fs/cgroup/unified/exa/mple/exa/mple/ and walk
		// up to /sys/fs/cgroup/unified/ until we find the
		// desired file.
		//
		// This might mean our reported stats include more
		// cgroups in the cgroup tree, but it's the best we
		// can do.
		for path := startpath; path != "" && path != "/" && (path == startpath || strings.HasPrefix(path, mounts[try.pathkey])); path, _ = filepath.Split(strings.TrimRight(path, "/")) {
			target := strings.TrimLeft(filepath.Join(path, try.file), "/")
			buf, err := fs.ReadFile(r.FS, target)
			if err != nil || len(buf) == 0 || bytes.Equal(buf, []byte{'\n'}) {
				if r.Debug {
					if os.IsNotExist(err) {
						// don't stutter
						err = os.ErrNotExist
					}
					r.Logger.Printf("skip /%s: %s", target, err)
				}
				continue
			}
			*try.statFile = target
			done[try.statFile] = true
			r.Logger.Printf("using /%s", target)
			break
		}
	}

	netdev := fmt.Sprintf("proc/%d/net/dev", r.pid)
	if buf, err := fs.ReadFile(r.FS, netdev); err == nil && len(buf) > 0 {
		r.statFiles.netDev = netdev
		r.Logger.Printf("using /%s", netdev)
	}
}

func (r *Reporter) reportMemoryMax(logger logPrinter, source, statName string, value, limit int64) {
	var units string
	switch statName {
	case "pgmajfault":
		units = "faults"
	default:
		units = "bytes"
	}
	if limit > 0 {
		percentage := 100 * value / limit
		logger.Printf("Maximum %s memory %s usage was %d%%, %d/%d %s",
			source, statName, percentage, value, limit, units)
	} else {
		logger.Printf("Maximum %s memory %s usage was %d %s",
			source, statName, value, units)
	}
}

func (r *Reporter) LogMaxima(logger logPrinter, memLimits map[string]int64) {
	if r.lastCPUSample.hasData {
		logger.Printf("Total CPU usage was %f user and %f sys on %d CPUs",
			r.lastCPUSample.user, r.lastCPUSample.sys, r.lastCPUSample.cpus)
	}
	for disk, sample := range r.lastDiskIOSample {
		logger.Printf("Total disk I/O on %s was %d bytes written and %d bytes read",
			disk, sample.txBytes, sample.rxBytes)
	}
	if r.maxDiskSpaceSample.total > 0 {
		percentage := 100 * r.maxDiskSpaceSample.used / r.maxDiskSpaceSample.total
		logger.Printf("Maximum disk usage was %d%%, %d/%d bytes",
			percentage, r.maxDiskSpaceSample.used, r.maxDiskSpaceSample.total)
	}
	for _, statName := range memoryStats {
		value, ok := r.maxMemSample[memoryKey{statName: "total_" + statName}]
		if !ok {
			value, ok = r.maxMemSample[memoryKey{statName: statName}]
		}
		if ok {
			r.reportMemoryMax(logger, "container", statName, value, memLimits[statName])
		}
	}
	for ifname, sample := range r.lastNetSample {
		logger.Printf("Total network I/O on %s was %d bytes written and %d bytes read",
			ifname, sample.txBytes, sample.rxBytes)
	}
}

func (r *Reporter) LogProcessMemMax(logger logPrinter) {
	for memKey, value := range r.maxMemSample {
		if memKey.processName == "" {
			continue
		}
		r.reportMemoryMax(logger, memKey.processName, memKey.statName, value, 0)
	}
}

func (r *Reporter) readAllOrWarn(in io.Reader) ([]byte, error) {
	content, err := ioutil.ReadAll(in)
	if err != nil {
		r.Logger.Printf("warning: %v", err)
	}
	return content, err
}

type ioSample struct {
	sampleTime time.Time
	txBytes    int64
	rxBytes    int64
}

func (r *Reporter) doBlkIOStats() {
	var sampleTime = time.Now()
	newSamples := make(map[string]ioSample)

	if r.statFiles.ioStat != "" {
		statfile, err := fs.ReadFile(r.FS, r.statFiles.ioStat)
		if err != nil {
			return
		}
		for _, line := range bytes.Split(statfile, []byte{'\n'}) {
			// 254:16 rbytes=72163328 wbytes=117370880 rios=3811 wios=3906 dbytes=0 dios=0
			words := bytes.Split(line, []byte{' '})
			if len(words) < 2 {
				continue
			}
			thisSample := ioSample{sampleTime, -1, -1}
			for _, kv := range words[1:] {
				if bytes.HasPrefix(kv, []byte("rbytes=")) {
					fmt.Sscanf(string(kv[7:]), "%d", &thisSample.rxBytes)
				} else if bytes.HasPrefix(kv, []byte("wbytes=")) {
					fmt.Sscanf(string(kv[7:]), "%d", &thisSample.txBytes)
				}
			}
			if thisSample.rxBytes >= 0 && thisSample.txBytes >= 0 {
				newSamples[string(words[0])] = thisSample
			}
		}
	} else if r.statFiles.ioServiceBytes != "" {
		statfile, err := fs.ReadFile(r.FS, r.statFiles.ioServiceBytes)
		if err != nil {
			return
		}
		for _, line := range bytes.Split(statfile, []byte{'\n'}) {
			var device, op string
			var val int64
			if _, err := fmt.Sscanf(string(line), "%s %s %d", &device, &op, &val); err != nil {
				continue
			}
			var thisSample ioSample
			var ok bool
			if thisSample, ok = newSamples[device]; !ok {
				thisSample = ioSample{sampleTime, -1, -1}
			}
			switch op {
			case "Read":
				thisSample.rxBytes = val
			case "Write":
				thisSample.txBytes = val
			}
			newSamples[device] = thisSample
		}
	}

	for dev, sample := range newSamples {
		if sample.txBytes < 0 || sample.rxBytes < 0 {
			continue
		}
		delta := ""
		if prev, ok := r.lastDiskIOSample[dev]; ok {
			delta = fmt.Sprintf(" -- interval %.4f seconds %d write %d read",
				sample.sampleTime.Sub(prev.sampleTime).Seconds(),
				sample.txBytes-prev.txBytes,
				sample.rxBytes-prev.rxBytes)
		}
		r.Logger.Printf("blkio:%s %d write %d read%s\n", dev, sample.txBytes, sample.rxBytes, delta)
		r.lastDiskIOSample[dev] = sample
	}
}

type memSample struct {
	sampleTime time.Time
	memStat    map[string]int64
}

func (r *Reporter) getMemSample() {
	thisSample := memSample{time.Now(), make(map[string]int64)}

	// memory.stat contains "pgmajfault" in cgroups v1 and v2. It
	// also contains "rss", "swap", and "cache" in cgroups v1.
	c, err := r.FS.Open(r.statFiles.memoryStat)
	if err != nil {
		return
	}
	defer c.Close()
	b := bufio.NewScanner(c)
	for b.Scan() {
		var stat string
		var val int64
		if _, err := fmt.Sscanf(string(b.Text()), "%s %d", &stat, &val); err != nil {
			continue
		}
		thisSample.memStat[stat] = val
	}

	// In cgroups v2, we need to read "memory.current" and
	// "memory.swap.current" as well.
	for stat, fnm := range map[string]string{
		// memory.current includes cache. We don't get
		// separate rss/cache values, so we call
		// memory usage "rss" for compatibility, and
		// omit "cache".
		"rss":  r.statFiles.memoryCurrent,
		"swap": r.statFiles.memorySwapCurrent,
	} {
		if fnm == "" {
			continue
		}
		buf, err := fs.ReadFile(r.FS, fnm)
		if err != nil {
			continue
		}
		var val int64
		_, err = fmt.Sscanf(string(buf), "%d", &val)
		if err != nil {
			continue
		}
		thisSample.memStat[stat] = val
	}
	for stat, val := range thisSample.memStat {
		maxKey := memoryKey{statName: stat}
		if val > r.maxMemSample[maxKey] {
			r.maxMemSample[maxKey] = val
		}
	}
	r.lastMemSample = thisSample

	if r.ThresholdLogger != nil {
		for statName, thresholds := range r.MemThresholds {
			statValue, ok := thisSample.memStat["total_"+statName]
			if !ok {
				statValue, ok = thisSample.memStat[statName]
				if !ok {
					continue
				}
			}
			var index int
			var statThreshold Threshold
			for index, statThreshold = range thresholds {
				if statValue < statThreshold.threshold {
					break
				} else if statThreshold.percentage > 0 {
					r.ThresholdLogger.Printf("Container using over %d%% of memory (%s %d/%d bytes)",
						statThreshold.percentage, statName, statValue, statThreshold.total)
				} else {
					r.ThresholdLogger.Printf("Container using over %d of memory (%s %s bytes)",
						statThreshold.threshold, statName, statValue)
				}
			}
			r.MemThresholds[statName] = thresholds[index:]
		}
	}
}

func (r *Reporter) reportMemSample() {
	var outstat bytes.Buffer
	for _, key := range memoryStats {
		// Use "total_X" stats (entire hierarchy) if enabled,
		// otherwise just the single cgroup -- see
		// https://www.kernel.org/doc/Documentation/cgroup-v1/memory.txt
		if val, ok := r.lastMemSample.memStat["total_"+key]; ok {
			fmt.Fprintf(&outstat, " %d %s", val, key)
		} else if val, ok := r.lastMemSample.memStat[key]; ok {
			fmt.Fprintf(&outstat, " %d %s", val, key)
		}
	}
	r.Logger.Printf("mem%s\n", outstat.String())
}

func (r *Reporter) doProcmemStats() {
	if r.kernelPageSize == 0 {
		// assign "don't try again" value in case we give up
		// and return without assigning the real value
		r.kernelPageSize = -1
		buf, err := fs.ReadFile(r.FS, "proc/self/smaps")
		if err != nil {
			r.Logger.Printf("error reading /proc/self/smaps: %s", err)
			return
		}
		m := regexp.MustCompile(`\nKernelPageSize:\s*(\d+) kB\n`).FindSubmatch(buf)
		if len(m) != 2 {
			r.Logger.Printf("error parsing /proc/self/smaps: KernelPageSize not found")
			return
		}
		size, err := strconv.ParseInt(string(m[1]), 10, 64)
		if err != nil {
			r.Logger.Printf("error parsing /proc/self/smaps: KernelPageSize %q: %s", m[1], err)
			return
		}
		r.kernelPageSize = size * 1024
	} else if r.kernelPageSize < 0 {
		// already failed to determine page size, don't keep
		// trying/logging
		return
	}

	r.reportPIDsMu.Lock()
	defer r.reportPIDsMu.Unlock()
	procnames := make([]string, 0, len(r.reportPIDs))
	for name := range r.reportPIDs {
		procnames = append(procnames, name)
	}
	sort.Strings(procnames)
	procmem := ""
	for _, procname := range procnames {
		pid := r.reportPIDs[procname]
		buf, err := fs.ReadFile(r.FS, fmt.Sprintf("proc/%d/stat", pid))
		if err != nil {
			continue
		}
		// If the executable name contains a ')' char,
		// /proc/$pid/stat will look like '1234 (exec name)) S
		// 123 ...' -- the last ')' is the end of the 2nd
		// field.
		paren := bytes.LastIndexByte(buf, ')')
		if paren < 0 {
			continue
		}
		fields := bytes.SplitN(buf[paren:], []byte{' '}, 24)
		if len(fields) < 24 {
			continue
		}
		// rss is the 24th field in .../stat, and fields[0]
		// here is the last char ')' of the 2nd field, so
		// rss is fields[22]
		rss, err := strconv.ParseInt(string(fields[22]), 10, 64)
		if err != nil {
			continue
		}
		value := rss * r.kernelPageSize
		procmem += fmt.Sprintf(" %d %s", value, procname)
		maxKey := memoryKey{pid, procname, "rss"}
		if value > r.maxMemSample[maxKey] {
			r.maxMemSample[maxKey] = value
		}
	}
	if procmem != "" {
		r.Logger.Printf("procmem%s\n", procmem)
	}
}

func (r *Reporter) doNetworkStats() {
	if r.statFiles.netDev == "" {
		return
	}
	sampleTime := time.Now()
	stats, err := r.FS.Open(r.statFiles.netDev)
	if err != nil {
		return
	}
	defer stats.Close()
	scanner := bufio.NewScanner(stats)
	for scanner.Scan() {
		var ifName string
		var rx, tx int64
		words := strings.Fields(scanner.Text())
		if len(words) != 17 {
			// Skip lines with wrong format
			continue
		}
		ifName = strings.TrimRight(words[0], ":")
		if ifName == "lo" || ifName == "" {
			// Skip loopback interface and lines with wrong format
			continue
		}
		if tx, err = strconv.ParseInt(words[9], 10, 64); err != nil {
			continue
		}
		if rx, err = strconv.ParseInt(words[1], 10, 64); err != nil {
			continue
		}
		nextSample := ioSample{}
		nextSample.sampleTime = sampleTime
		nextSample.txBytes = tx
		nextSample.rxBytes = rx
		var delta string
		if prev, ok := r.lastNetSample[ifName]; ok {
			interval := nextSample.sampleTime.Sub(prev.sampleTime).Seconds()
			delta = fmt.Sprintf(" -- interval %.4f seconds %d tx %d rx",
				interval,
				tx-prev.txBytes,
				rx-prev.rxBytes)
		}
		r.Logger.Printf("net:%s %d tx %d rx%s\n", ifName, tx, rx, delta)
		r.lastNetSample[ifName] = nextSample
	}
}

type diskSpaceSample struct {
	hasData    bool
	sampleTime time.Time
	total      uint64
	used       uint64
	available  uint64
}

func (r *Reporter) doDiskSpaceStats() {
	s := syscall.Statfs_t{}
	err := syscall.Statfs(r.TempDir, &s)
	if err != nil {
		return
	}
	bs := uint64(s.Bsize)
	nextSample := diskSpaceSample{
		hasData:    true,
		sampleTime: time.Now(),
		total:      s.Blocks * bs,
		used:       (s.Blocks - s.Bfree) * bs,
		available:  s.Bavail * bs,
	}
	if nextSample.used > r.maxDiskSpaceSample.used {
		r.maxDiskSpaceSample = nextSample
	}

	var delta string
	if r.lastDiskSpaceSample.hasData {
		prev := r.lastDiskSpaceSample
		interval := nextSample.sampleTime.Sub(prev.sampleTime).Seconds()
		delta = fmt.Sprintf(" -- interval %.4f seconds %d used",
			interval,
			int64(nextSample.used-prev.used))
	}
	r.Logger.Printf("statfs %d available %d used %d total%s\n",
		nextSample.available, nextSample.used, nextSample.total, delta)
	r.lastDiskSpaceSample = nextSample
}

type cpuSample struct {
	hasData    bool // to distinguish the zero value from real data
	sampleTime time.Time
	user       float64
	sys        float64
	cpus       int64
}

// Return the number of CPUs available in the container. Return 0 if
// we can't figure out the real number of CPUs.
func (r *Reporter) getCPUCount() int64 {
	buf, err := fs.ReadFile(r.FS, r.statFiles.cpusetCpus)
	if err != nil {
		return 0
	}
	cpus := int64(0)
	for _, v := range bytes.Split(buf, []byte{','}) {
		var min, max int64
		n, _ := fmt.Sscanf(string(v), "%d-%d", &min, &max)
		if n == 2 {
			cpus += (max - min) + 1
		} else {
			cpus++
		}
	}
	return cpus
}

func (r *Reporter) doCPUStats() {
	var nextSample cpuSample
	if r.statFiles.cpuStat != "" {
		// v2
		f, err := r.FS.Open(r.statFiles.cpuStat)
		if err != nil {
			return
		}
		defer f.Close()
		nextSample = cpuSample{
			hasData:    true,
			sampleTime: time.Now(),
			cpus:       r.getCPUCount(),
		}
		for {
			var stat string
			var val int64
			n, err := fmt.Fscanf(f, "%s %d\n", &stat, &val)
			if err != nil || n != 2 {
				break
			}
			if stat == "user_usec" {
				nextSample.user = float64(val) / 1000000
			} else if stat == "system_usec" {
				nextSample.sys = float64(val) / 1000000
			}
		}
	} else if r.statFiles.cpuacctStat != "" {
		// v1
		b, err := fs.ReadFile(r.FS, r.statFiles.cpuacctStat)
		if err != nil {
			return
		}

		var userTicks, sysTicks int64
		fmt.Sscanf(string(b), "user %d\nsystem %d", &userTicks, &sysTicks)
		userHz := float64(100)
		nextSample = cpuSample{
			hasData:    true,
			sampleTime: time.Now(),
			user:       float64(userTicks) / userHz,
			sys:        float64(sysTicks) / userHz,
			cpus:       r.getCPUCount(),
		}
	}

	delta := ""
	if r.lastCPUSample.hasData {
		delta = fmt.Sprintf(" -- interval %.4f seconds %.4f user %.4f sys",
			nextSample.sampleTime.Sub(r.lastCPUSample.sampleTime).Seconds(),
			nextSample.user-r.lastCPUSample.user,
			nextSample.sys-r.lastCPUSample.sys)
	}
	r.Logger.Printf("cpu %.4f user %.4f sys %d cpus%s\n",
		nextSample.user, nextSample.sys, nextSample.cpus, delta)
	r.lastCPUSample = nextSample
}

func (r *Reporter) doAllStats() {
	r.reportMemSample()
	r.doProcmemStats()
	r.doCPUStats()
	r.doBlkIOStats()
	r.doNetworkStats()
	r.doDiskSpaceStats()
}

// Report stats periodically until we learn (via r.done) that someone
// called Stop.
func (r *Reporter) run() {
	defer close(r.flushed)

	r.maxMemSample = make(map[memoryKey]int64)

	if !r.waitForPid() {
		return
	}
	r.findStatFiles()
	close(r.ready)

	r.lastNetSample = make(map[string]ioSample)
	r.lastDiskIOSample = make(map[string]ioSample)

	if len(r.TempDir) == 0 {
		// Temporary dir not provided, try to get it from the environment.
		r.TempDir = os.Getenv("TMPDIR")
	}
	if len(r.TempDir) > 0 {
		r.Logger.Printf("notice: monitoring temp dir %s\n", r.TempDir)
	}

	r.getMemSample()
	r.doAllStats()

	if r.PollPeriod < 1 {
		r.PollPeriod = time.Second * 10
	}

	memTicker := time.NewTicker(time.Second)
	mainTicker := time.NewTicker(r.PollPeriod)
	for {
		select {
		case <-r.done:
			return
		case <-memTicker.C:
			r.getMemSample()
		case <-mainTicker.C:
			r.doAllStats()
		}
	}
}

// Wait for Pid() to return a real pid.  Return true if this succeeds
// before Stop is called.
func (r *Reporter) waitForPid() bool {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	warningTimer := time.After(r.PollPeriod)
	for {
		r.pid = r.Pid()
		if r.pid > 0 {
			break
		}
		select {
		case <-ticker.C:
		case <-warningTimer:
			r.Logger.Printf("warning: Pid() did not return a process ID after %v (config error?) -- still waiting...", r.PollPeriod)
		case <-r.done:
			r.Logger.Printf("warning: Pid() never returned a process ID")
			return false
		}
	}
	return true
}

func (r *Reporter) dumpSourceFiles(destdir string) error {
	select {
	case <-r.done:
		return errors.New("reporter was never ready")
	case <-r.ready:
	}
	todo := []string{
		fmt.Sprintf("proc/%d/cgroup", r.pid),
		fmt.Sprintf("proc/%d/cpuset", r.pid),
		"proc/mounts",
		"proc/self/smaps",
		r.statFiles.cpusetCpus,
		r.statFiles.cpuacctStat,
		r.statFiles.cpuStat,
		r.statFiles.ioServiceBytes,
		r.statFiles.ioStat,
		r.statFiles.memoryStat,
		r.statFiles.memoryCurrent,
		r.statFiles.memorySwapCurrent,
		r.statFiles.netDev,
	}
	for _, path := range todo {
		if path == "" {
			continue
		}
		err := r.createParentsAndCopyFile(destdir, path)
		if err != nil {
			return err
		}
	}
	r.reportPIDsMu.Lock()
	r.reportPIDsMu.Unlock()
	for _, pid := range r.reportPIDs {
		path := fmt.Sprintf("proc/%d/stat", pid)
		err := r.createParentsAndCopyFile(destdir, path)
		if err != nil {
			return err
		}
	}
	if proc, err := os.FindProcess(r.pid); err != nil || proc.Signal(syscall.Signal(0)) != nil {
		return fmt.Errorf("process %d no longer exists, snapshot is probably broken", r.pid)
	}
	return nil
}

func (r *Reporter) createParentsAndCopyFile(destdir, path string) error {
	buf, err := fs.ReadFile(r.FS, path)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	if parent, _ := filepath.Split(path); parent != "" {
		err = os.MkdirAll(destdir+"/"+parent, 0777)
		if err != nil {
			return fmt.Errorf("mkdir %s: %s", destdir+"/"+parent, err)
		}
	}
	destfile := destdir + "/" + path
	r.Logger.Printf("copy %s to %s -- size %d", path, destfile, len(buf))
	return os.WriteFile(destfile, buf, 0777)
}
