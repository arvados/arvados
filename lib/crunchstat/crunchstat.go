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
	"io/ioutil"
	"os"
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
	// CID of the container to monitor. If empty, read the CID
	// from CIDFile (first waiting until a non-empty file appears
	// at CIDFile). If CIDFile is also empty, report host
	// statistics.
	CID string

	// Path to a file we can read CID from.
	CIDFile string

	// Where cgroup accounting files live on this system, e.g.,
	// "/sys/fs/cgroup".
	CgroupRoot string

	// Parent cgroup, e.g., "docker".
	CgroupParent string

	// Interval between samples. Must be positive.
	PollPeriod time.Duration

	// Temporary directory, will be monitored for available, used & total space.
	TempDir string

	// Where to write statistics. Must not be nil.
	Logger logPrinter

	// When stats cross thresholds configured in the fields below,
	// they are reported to this logger.
	ThresholdLogger logPrinter

	// MemThresholds maps memory stat names to slices of thresholds.
	// When the corresponding stat exceeds a threshold, that will be logged.
	MemThresholds map[string][]Threshold

	kernelPageSize      int64
	reportedStatFile    map[string]string
	lastNetSample       map[string]ioSample
	lastDiskIOSample    map[string]ioSample
	lastCPUSample       cpuSample
	lastDiskSpaceSample diskSpaceSample
	lastMemSample       memSample
	maxDiskSpaceSample  diskSpaceSample
	maxMemSample        map[memoryKey]int64

	reportPIDs   map[string]int
	reportPIDsMu sync.Mutex

	done    chan struct{} // closed when we should stop reporting
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
	r.flushed = make(chan struct{})
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
	if r.maxDiskSpaceSample.hasData {
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

// Open the cgroup stats file in /sys/fs corresponding to the target
// cgroup, and return an io.ReadCloser. If no stats file is available,
// return nil.
//
// Log the file that was opened, if it isn't the same file opened on
// the last openStatFile for this stat.
//
// Log "not available" if no file is found and either this stat has
// been available in the past, or verbose==true.
//
// TODO: Instead of trying all options, choose a process in the
// container, and read /proc/PID/cgroup to determine the appropriate
// cgroup root for the given statgroup. (This will avoid falling back
// to host-level stats during container setup and teardown.)
func (r *Reporter) openStatFile(statgroup, stat string, verbose bool) (io.ReadCloser, error) {
	var paths []string
	if r.CID != "" {
		// Collect container's stats
		paths = []string{
			fmt.Sprintf("%s/%s/%s/%s/%s", r.CgroupRoot, statgroup, r.CgroupParent, r.CID, stat),
			fmt.Sprintf("%s/%s/%s/%s", r.CgroupRoot, r.CgroupParent, r.CID, stat),
		}
	} else {
		// Collect this host's stats
		paths = []string{
			fmt.Sprintf("%s/%s/%s", r.CgroupRoot, statgroup, stat),
			fmt.Sprintf("%s/%s", r.CgroupRoot, stat),
		}
	}
	var path string
	var file *os.File
	var err error
	for _, path = range paths {
		file, err = os.Open(path)
		if err == nil {
			break
		} else {
			path = ""
		}
	}
	if pathWas := r.reportedStatFile[stat]; pathWas != path {
		// Log whenever we start using a new/different cgroup
		// stat file for a given statistic. This typically
		// happens 1 to 3 times per statistic, depending on
		// whether we happen to collect stats [a] before any
		// processes have been created in the container and
		// [b] after all contained processes have exited.
		if path == "" && verbose {
			r.Logger.Printf("notice: stats not available: stat %s, statgroup %s, cid %s, parent %s, root %s\n", stat, statgroup, r.CID, r.CgroupParent, r.CgroupRoot)
		} else if pathWas != "" {
			r.Logger.Printf("notice: stats moved from %s to %s\n", r.reportedStatFile[stat], path)
		} else {
			r.Logger.Printf("notice: reading stats from %s\n", path)
		}
		r.reportedStatFile[stat] = path
	}
	return file, err
}

func (r *Reporter) getContainerNetStats() (io.Reader, error) {
	procsFile, err := r.openStatFile("cpuacct", "cgroup.procs", true)
	if err != nil {
		return nil, err
	}
	defer procsFile.Close()
	reader := bufio.NewScanner(procsFile)
	for reader.Scan() {
		taskPid := reader.Text()
		statsFilename := fmt.Sprintf("/proc/%s/net/dev", taskPid)
		stats, err := ioutil.ReadFile(statsFilename)
		if err != nil {
			r.Logger.Printf("notice: %v", err)
			continue
		}
		return strings.NewReader(string(stats)), nil
	}
	return nil, errors.New("Could not read stats for any proc in container")
}

type ioSample struct {
	sampleTime time.Time
	txBytes    int64
	rxBytes    int64
}

func (r *Reporter) doBlkIOStats() {
	c, err := r.openStatFile("blkio", "blkio.io_service_bytes", true)
	if err != nil {
		return
	}
	defer c.Close()
	b := bufio.NewScanner(c)
	var sampleTime = time.Now()
	newSamples := make(map[string]ioSample)
	for b.Scan() {
		var device, op string
		var val int64
		if _, err := fmt.Sscanf(string(b.Text()), "%s %s %d", &device, &op, &val); err != nil {
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
	c, err := r.openStatFile("memory", "memory.stat", true)
	if err != nil {
		return
	}
	defer c.Close()
	b := bufio.NewScanner(c)
	thisSample := memSample{time.Now(), make(map[string]int64)}
	for b.Scan() {
		var stat string
		var val int64
		if _, err := fmt.Sscanf(string(b.Text()), "%s %d", &stat, &val); err != nil {
			continue
		}
		thisSample.memStat[stat] = val
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
		buf, err := os.ReadFile("/proc/self/smaps")
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
		buf, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
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
	sampleTime := time.Now()
	stats, err := r.getContainerNetStats()
	if err != nil {
		return
	}

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
	cpusetFile, err := r.openStatFile("cpuset", "cpuset.cpus", true)
	if err != nil {
		return 0
	}
	defer cpusetFile.Close()
	b, err := r.readAllOrWarn(cpusetFile)
	if err != nil {
		return 0
	}
	sp := strings.Split(string(b), ",")
	cpus := int64(0)
	for _, v := range sp {
		var min, max int64
		n, _ := fmt.Sscanf(v, "%d-%d", &min, &max)
		if n == 2 {
			cpus += (max - min) + 1
		} else {
			cpus++
		}
	}
	return cpus
}

func (r *Reporter) doCPUStats() {
	statFile, err := r.openStatFile("cpuacct", "cpuacct.stat", true)
	if err != nil {
		return
	}
	defer statFile.Close()
	b, err := r.readAllOrWarn(statFile)
	if err != nil {
		return
	}

	var userTicks, sysTicks int64
	fmt.Sscanf(string(b), "user %d\nsystem %d", &userTicks, &sysTicks)
	userHz := float64(100)
	nextSample := cpuSample{
		hasData:    true,
		sampleTime: time.Now(),
		user:       float64(userTicks) / userHz,
		sys:        float64(sysTicks) / userHz,
		cpus:       r.getCPUCount(),
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
	r.reportedStatFile = make(map[string]string)

	if !r.waitForCIDFile() || !r.waitForCgroup() {
		return
	}

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

// If CID is empty, wait for it to appear in CIDFile. Return true if
// we get it before we learn (via r.done) that someone called Stop.
func (r *Reporter) waitForCIDFile() bool {
	if r.CID != "" || r.CIDFile == "" {
		return true
	}

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		cid, err := ioutil.ReadFile(r.CIDFile)
		if err == nil && len(cid) > 0 {
			r.CID = string(cid)
			return true
		}
		select {
		case <-ticker.C:
		case <-r.done:
			r.Logger.Printf("warning: CID never appeared in %+q: %v", r.CIDFile, err)
			return false
		}
	}
}

// Wait for the cgroup stats files to appear in cgroup_root. Return
// true if they appear before r.done indicates someone called Stop. If
// they don't appear within one poll interval, log a warning and keep
// waiting.
func (r *Reporter) waitForCgroup() bool {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	warningTimer := time.After(r.PollPeriod)
	for {
		c, err := r.openStatFile("cpuacct", "cgroup.procs", false)
		if err == nil {
			c.Close()
			return true
		}
		select {
		case <-ticker.C:
		case <-warningTimer:
			r.Logger.Printf("warning: cgroup stats files have not appeared after %v (config error?) -- still waiting...", r.PollPeriod)
		case <-r.done:
			r.Logger.Printf("warning: cgroup stats files never appeared for %v", r.CID)
			return false
		}
	}
}
