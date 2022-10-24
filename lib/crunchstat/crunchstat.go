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
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

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
	Logger interface {
		Printf(fmt string, args ...interface{})
	}

	reportedStatFile    map[string]string
	lastNetSample       map[string]ioSample
	lastDiskIOSample    map[string]ioSample
	lastCPUSample       cpuSample
	lastDiskSpaceSample diskSpaceSample

	reportPIDs   map[string]int
	reportPIDsMu sync.Mutex

	done    chan struct{} // closed when we should stop reporting
	flushed chan struct{} // closed when we have made our last report
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
// Nothing will be logged after Stop returns.
func (r *Reporter) Stop() {
	close(r.done)
	<-r.flushed
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

func (r *Reporter) doMemoryStats() {
	c, err := r.openStatFile("memory", "memory.stat", true)
	if err != nil {
		return
	}
	defer c.Close()
	b := bufio.NewScanner(c)
	thisSample := memSample{time.Now(), make(map[string]int64)}
	wantStats := [...]string{"cache", "swap", "pgmajfault", "rss"}
	for b.Scan() {
		var stat string
		var val int64
		if _, err := fmt.Sscanf(string(b.Text()), "%s %d", &stat, &val); err != nil {
			continue
		}
		thisSample.memStat[stat] = val
	}
	var outstat bytes.Buffer
	for _, key := range wantStats {
		// Use "total_X" stats (entire hierarchy) if enabled,
		// otherwise just the single cgroup -- see
		// https://www.kernel.org/doc/Documentation/cgroup-v1/memory.txt
		if val, ok := thisSample.memStat["total_"+key]; ok {
			fmt.Fprintf(&outstat, " %d %s", val, key)
		} else if val, ok := thisSample.memStat[key]; ok {
			fmt.Fprintf(&outstat, " %d %s", val, key)
		}
	}
	r.Logger.Printf("mem%s\n", outstat.String())

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
		rss, err := strconv.Atoi(string(fields[22]))
		if err != nil {
			continue
		}
		procmem += fmt.Sprintf(" %d %s", rss, procname)
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

// Report stats periodically until we learn (via r.done) that someone
// called Stop.
func (r *Reporter) run() {
	defer close(r.flushed)

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

	ticker := time.NewTicker(r.PollPeriod)
	for {
		r.doMemoryStats()
		r.doCPUStats()
		r.doBlkIOStats()
		r.doNetworkStats()
		r.doDiskSpaceStats()
		select {
		case <-r.done:
			return
		case <-ticker.C:
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
