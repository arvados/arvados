package crunchstat

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// This magically allows us to look up user_hz via _SC_CLK_TCK:

/*
#include <unistd.h>
#include <sys/types.h>
#include <pwd.h>
#include <stdlib.h>
*/
import "C"

// A Reporter gathers statistics for a cgroup and writes them to a
// log.Logger.
type Reporter struct {
	// CID of the container to monitor. If empty, read the CID
	// from CIDFile.
	CID string
	// Where cgroup special files live on this system
	CgroupRoot   string
	CgroupParent string
	// Path to a file we can read CID from. If CIDFile is empty or
	// nonexistent, wait for it to appear.
	CIDFile string

	// Interval between samples
	Poll time.Duration

	// Where to write statistics.
	Logger *log.Logger

	reportedStatFile map[string]string
	lastNetSample    map[string]IoSample
	lastDiskSample   map[string]IoSample
	lastCPUSample    CpuSample

	done chan struct{}
}

// Wait (if necessary) for the CID to appear in CIDFile, then start
// reporting statistics.
//
// Start should not be called more than once on a Reporter.
//
// Public data fields should not be changed after calling Start.
func (r *Reporter) Start() {
	r.done = make(chan struct{})
	go r.run()
}

// Stop reporting statistics. Do not call more than once, or before
// calling Start.
//
// Nothing will be logged after Stop returns.
func (r *Reporter) Stop() {
	close(r.done)
}

func (r *Reporter) readAllOrWarn(in *os.File) ([]byte, error) {
	content, err := ioutil.ReadAll(in)
	if err != nil {
		r.Logger.Print(err)
	}
	return content, err
}

// Open the cgroup stats file in /sys/fs corresponding to the target
// cgroup, and return an *os.File. If no stats file is available,
// return nil.
//
// TODO: Instead of trying all options, choose a process in the
// container, and read /proc/PID/cgroup to determine the appropriate
// cgroup root for the given statgroup. (This will avoid falling back
// to host-level stats during container setup and teardown.)
func (r *Reporter) openStatFile(statgroup string, stat string) (*os.File, error) {
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
	if pathWas, ok := r.reportedStatFile[stat]; !ok || pathWas != path {
		// Log whenever we start using a new/different cgroup
		// stat file for a given statistic. This typically
		// happens 1 to 3 times per statistic, depending on
		// whether we happen to collect stats [a] before any
		// processes have been created in the container and
		// [b] after all contained processes have exited.
		if path == "" {
			r.Logger.Printf("notice: stats not available: stat %s, statgroup %s, cid %s, parent %s, root %s\n", stat, statgroup, r.CID, r.CgroupParent, r.CgroupRoot)
		} else if ok {
			r.Logger.Printf("notice: stats moved from %s to %s\n", r.reportedStatFile[stat], path)
		} else {
			r.Logger.Printf("notice: reading stats from %s\n", path)
		}
		r.reportedStatFile[stat] = path
	}
	return file, err
}

func (r *Reporter) getContainerNetStats() (io.Reader, error) {
	procsFile, err := r.openStatFile("cpuacct", "cgroup.procs")
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
			r.Logger.Print(err)
			continue
		}
		return strings.NewReader(string(stats)), nil
	}
	return nil, errors.New("Could not read stats for any proc in container")
}

type IoSample struct {
	sampleTime time.Time
	txBytes    int64
	rxBytes    int64
}

func (r *Reporter) DoBlkIoStats() {
	c, err := r.openStatFile("blkio", "blkio.io_service_bytes")
	if err != nil {
		return
	}
	defer c.Close()
	b := bufio.NewScanner(c)
	var sampleTime = time.Now()
	newSamples := make(map[string]IoSample)
	for b.Scan() {
		var device, op string
		var val int64
		if _, err := fmt.Sscanf(string(b.Text()), "%s %s %d", &device, &op, &val); err != nil {
			continue
		}
		var thisSample IoSample
		var ok bool
		if thisSample, ok = newSamples[device]; !ok {
			thisSample = IoSample{sampleTime, -1, -1}
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
		if prev, ok := r.lastDiskSample[dev]; ok {
			delta = fmt.Sprintf(" -- interval %.4f seconds %d write %d read",
				sample.sampleTime.Sub(prev.sampleTime).Seconds(),
				sample.txBytes-prev.txBytes,
				sample.rxBytes-prev.rxBytes)
		}
		r.Logger.Printf("blkio:%s %d write %d read%s\n", dev, sample.txBytes, sample.rxBytes, delta)
		r.lastDiskSample[dev] = sample
	}
}

type MemSample struct {
	sampleTime time.Time
	memStat    map[string]int64
}

func (r *Reporter) DoMemoryStats() {
	c, err := r.openStatFile("memory", "memory.stat")
	if err != nil {
		return
	}
	defer c.Close()
	b := bufio.NewScanner(c)
	thisSample := MemSample{time.Now(), make(map[string]int64)}
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
		if val, ok := thisSample.memStat[key]; ok {
			outstat.WriteString(fmt.Sprintf(" %d %s", val, key))
		}
	}
	r.Logger.Printf("mem%s\n", outstat.String())
}

func (r *Reporter) DoNetworkStats() {
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
		nextSample := IoSample{}
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

type CpuSample struct {
	hasData    bool // to distinguish the zero value from real data
	sampleTime time.Time
	user       float64
	sys        float64
	cpus       int64
}

// Return the number of CPUs available in the container. Return 0 if
// we can't figure out the real number of CPUs.
func (r *Reporter) GetCpuCount() int64 {
	cpusetFile, err := r.openStatFile("cpuset", "cpuset.cpus")
	if err != nil {
		return 0
	}
	defer cpusetFile.Close()
	b, err := r.readAllOrWarn(cpusetFile)
	sp := strings.Split(string(b), ",")
	cpus := int64(0)
	for _, v := range sp {
		var min, max int64
		n, _ := fmt.Sscanf(v, "%d-%d", &min, &max)
		if n == 2 {
			cpus += (max - min) + 1
		} else {
			cpus += 1
		}
	}
	return cpus
}

func (r *Reporter) DoCpuStats() {
	statFile, err := r.openStatFile("cpuacct", "cpuacct.stat")
	if err != nil {
		return
	}
	defer statFile.Close()
	b, err := r.readAllOrWarn(statFile)
	if err != nil {
		return
	}

	nextSample := CpuSample{true, time.Now(), 0, 0, r.GetCpuCount()}
	var userTicks, sysTicks int64
	fmt.Sscanf(string(b), "user %d\nsystem %d", &userTicks, &sysTicks)
	user_hz := float64(C.sysconf(C._SC_CLK_TCK))
	nextSample.user = float64(userTicks) / user_hz
	nextSample.sys = float64(sysTicks) / user_hz

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

// Report stats periodically until someone closes or sends to r.done.
func (r *Reporter) run() {
	if !r.waitForCIDFile() {
		return
	}

	r.reportedStatFile = make(map[string]string)
	r.lastNetSample = make(map[string]IoSample)
	r.lastDiskSample = make(map[string]IoSample)

	ticker := time.NewTicker(r.Poll)
	for {
		r.DoMemoryStats()
		r.DoCpuStats()
		r.DoBlkIoStats()
		r.DoNetworkStats()
		select {
		case <-r.done:
			return
		case <-ticker.C:
		}
	}
}

// If CID is empty, wait for it to appear in CIDFile. Return true if
// we get it before someone calls Stop().
func (r *Reporter) waitForCIDFile() bool {
	if r.CID != "" {
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
			r.Logger.Printf("CID never appeared in %+q: %v", r.CIDFile, err)
			return false
		}
	}
}
