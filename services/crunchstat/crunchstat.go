package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

/*
#include <unistd.h>
#include <sys/types.h>
#include <pwd.h>
#include <stdlib.h>
*/
import "C"

// The above block of magic allows us to look up user_hz via _SC_CLK_TCK.

type Cgroup struct {
	root   string
	parent string
	cid    string
}

func CopyPipeToChan(in io.Reader, out chan string, done chan<- bool) {
	s := bufio.NewScanner(in)
	for s.Scan() {
		out <- s.Text()
	}
	done <- true
}

func CopyChanToPipe(in <-chan string, out io.Writer) {
	for s := range in {
		fmt.Fprintln(out, s)
	}
}

func OpenAndReadAll(filename string, log_chan chan<- string) ([]byte, error) {
	in, err := os.Open(filename)
	if err != nil {
		if log_chan != nil {
			log_chan <- fmt.Sprintf("crunchstat: open %s: %s", filename, err)
		}
		return nil, err
	}
	defer in.Close()
	return ReadAllOrWarn(in, log_chan)
}

func ReadAllOrWarn(in *os.File, log_chan chan<- string) ([]byte, error) {
	content, err := ioutil.ReadAll(in)
	if err != nil && log_chan != nil {
		log_chan <- fmt.Sprintf("crunchstat: read %s: %s", in.Name(), err)
	}
	return content, err
}

var reportedStatFile = map[string]string{}

// Open the cgroup stats file in /sys/fs corresponding to the target
// cgroup, and return an *os.File. If no stats file is available,
// return nil.
//
// TODO: Instead of trying all options, choose a process in the
// container, and read /proc/PID/cgroup to determine the appropriate
// cgroup root for the given statgroup. (This will avoid falling back
// to host-level stats during container setup and teardown.)
func OpenStatFile(stderr chan<- string, cgroup Cgroup, statgroup string, stat string) (*os.File, error) {
	var paths = []string{}
	paths = append(paths, fmt.Sprintf("%s/%s/%s/%s/%s", cgroup.root, statgroup, cgroup.parent, cgroup.cid, stat))
	paths = append(paths, fmt.Sprintf("%s/%s/%s/%s", cgroup.root, cgroup.parent, cgroup.cid, stat))
	paths = append(paths, fmt.Sprintf("%s/%s/%s", cgroup.root, statgroup, stat))
	paths = append(paths, fmt.Sprintf("%s/%s", cgroup.root, stat))
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
	if pathWas, ok := reportedStatFile[stat]; !ok || pathWas != path {
		// Log whenever we start using a new/different cgroup
		// stat file for a given statistic. This typically
		// happens 1 to 3 times per statistic, depending on
		// whether we happen to collect stats [a] before any
		// processes have been created in the container and
		// [b] after all contained processes have exited.
		reportedStatFile[stat] = path
		if path == "" {
			stderr <- fmt.Sprintf("crunchstat: did not find stats file: stat %s, statgroup %s, cid %s, parent %s, root %s", stat, statgroup, cgroup.cid, cgroup.parent, cgroup.root)
		} else {
			stderr <- fmt.Sprintf("crunchstat: reading stats from %s", path)
		}
	}
	return file, err
}

func GetContainerNetStats(stderr chan<- string, cgroup Cgroup) (io.Reader, error) {
	procsFile, err := OpenStatFile(stderr, cgroup, "cpuacct", "cgroup.procs")
	if err != nil {
		return nil, err
	}
	defer procsFile.Close()
	reader := bufio.NewScanner(procsFile)
	for reader.Scan() {
		taskPid := reader.Text()
		statsFilename := fmt.Sprintf("/proc/%s/net/dev", taskPid)
		stats, err := OpenAndReadAll(statsFilename, stderr)
		if err != nil {
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

func DoBlkIoStats(stderr chan<- string, cgroup Cgroup, lastSample map[string]IoSample) {
	c, err := OpenStatFile(stderr, cgroup, "blkio", "blkio.io_service_bytes")
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
		if prev, ok := lastSample[dev]; ok {
			delta = fmt.Sprintf(" -- interval %.4f seconds %d write %d read",
				sample.sampleTime.Sub(prev.sampleTime).Seconds(),
				sample.txBytes-prev.txBytes,
				sample.rxBytes-prev.rxBytes)
		}
		stderr <- fmt.Sprintf("crunchstat: blkio:%s %d write %d read%s", dev, sample.txBytes, sample.rxBytes, delta)
		lastSample[dev] = sample
	}
}

type MemSample struct {
	sampleTime time.Time
	memStat    map[string]int64
}

func DoMemoryStats(stderr chan<- string, cgroup Cgroup) {
	c, err := OpenStatFile(stderr, cgroup, "memory", "memory.stat")
	if err != nil {
		return
	}
	defer c.Close()
	b := bufio.NewScanner(c)
	thisSample := MemSample{time.Now(), make(map[string]int64)}
	wantStats := [...]string{"cache", "pgmajfault", "rss"}
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
	stderr <- fmt.Sprintf("crunchstat: mem%s", outstat.String())
}

func DoNetworkStats(stderr chan<- string, cgroup Cgroup, lastSample map[string]IoSample) {
	sampleTime := time.Now()
	stats, err := GetContainerNetStats(stderr, cgroup)
	if err != nil {
		return
	}

	scanner := bufio.NewScanner(stats)
Iface:
	for scanner.Scan() {
		var ifName string
		var rx, tx int64
		words := bufio.NewScanner(strings.NewReader(scanner.Text()))
		words.Split(bufio.ScanWords)
		wordIndex := 0
		for words.Scan() {
			word := words.Text()
			switch wordIndex {
			case 0:
				ifName = strings.TrimRight(word, ":")
			case 1:
				if _, err := fmt.Sscanf(word, "%d", &rx); err != nil {
					continue Iface
				}
			case 9:
				if _, err := fmt.Sscanf(word, "%d", &tx); err != nil {
					continue Iface
				}
			}
			wordIndex++
		}
		if ifName == "lo" || ifName == "" || wordIndex != 17 {
			// Skip loopback interface and lines with wrong format
			continue
		}
		nextSample := IoSample{}
		nextSample.sampleTime = sampleTime
		nextSample.txBytes = tx
		nextSample.rxBytes = rx
		var delta string
		if lastSample, ok := lastSample[ifName]; ok {
			interval := nextSample.sampleTime.Sub(lastSample.sampleTime).Seconds()
			delta = fmt.Sprintf(" -- interval %.4f seconds %d tx %d rx",
				interval,
				tx-lastSample.txBytes,
				rx-lastSample.rxBytes)
		}
		stderr <- fmt.Sprintf("crunchstat: net:%s %d tx %d rx%s",
			ifName, tx, rx, delta)
		lastSample[ifName] = nextSample
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
func GetCpuCount(stderr chan<- string, cgroup Cgroup) int64 {
	cpusetFile, err := OpenStatFile(stderr, cgroup, "cpuset", "cpuset.cpus")
	if err != nil {
		return 0
	}
	defer cpusetFile.Close()
	b, err := ReadAllOrWarn(cpusetFile, stderr)
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

func DoCpuStats(stderr chan<- string, cgroup Cgroup, lastSample *CpuSample) {
	statFile, err := OpenStatFile(stderr, cgroup, "cpuacct", "cpuacct.stat")
	if err != nil {
		return
	}
	defer statFile.Close()
	b, err := ReadAllOrWarn(statFile, stderr)
	if err != nil {
		return
	}

	nextSample := CpuSample{true, time.Now(), 0, 0, GetCpuCount(stderr, cgroup)}
	var userTicks, sysTicks int64
	fmt.Sscanf(string(b), "user %d\nsystem %d", &userTicks, &sysTicks)
	user_hz := float64(C.sysconf(C._SC_CLK_TCK))
	nextSample.user = float64(userTicks) / user_hz
	nextSample.sys = float64(sysTicks) / user_hz

	delta := ""
	if lastSample.hasData {
		delta = fmt.Sprintf(" -- interval %.4f seconds %.4f user %.4f sys",
			nextSample.sampleTime.Sub(lastSample.sampleTime).Seconds(),
			nextSample.user-lastSample.user,
			nextSample.sys-lastSample.sys)
	}
	stderr <- fmt.Sprintf("crunchstat: cpu %.4f user %.4f sys %d cpus%s",
		nextSample.user, nextSample.sys, nextSample.cpus, delta)
	*lastSample = nextSample
}

func PollCgroupStats(cgroup Cgroup, stderr chan string, poll int64, stop_poll_chan <-chan bool) {
	var lastNetSample = map[string]IoSample{}
	var lastDiskSample = map[string]IoSample{}
	var lastCpuSample = CpuSample{}

	poll_chan := make(chan bool, 1)
	go func() {
		// Send periodic poll events.
		poll_chan <- true
		for {
			time.Sleep(time.Duration(poll) * time.Millisecond)
			poll_chan <- true
		}
	}()
	for {
		select {
		case <-stop_poll_chan:
			return
		case <-poll_chan:
			// Emit stats, then select again.
		}
		DoMemoryStats(stderr, cgroup)
		DoCpuStats(stderr, cgroup, &lastCpuSample)
		DoBlkIoStats(stderr, cgroup, lastDiskSample)
		DoNetworkStats(stderr, cgroup, lastNetSample)
	}
}

func run(logger *log.Logger) error {

	var (
		cgroup_root    string
		cgroup_parent  string
		cgroup_cidfile string
		wait           int64
		poll           int64
	)

	flag.StringVar(&cgroup_root, "cgroup-root", "", "Root of cgroup tree")
	flag.StringVar(&cgroup_parent, "cgroup-parent", "", "Name of container parent under cgroup")
	flag.StringVar(&cgroup_cidfile, "cgroup-cid", "", "Path to container id file")
	flag.Int64Var(&wait, "wait", 5, "Maximum time (in seconds) to wait for cid file to show up")
	flag.Int64Var(&poll, "poll", 1000, "Polling frequency, in milliseconds")

	flag.Parse()

	if cgroup_root == "" {
		logger.Fatal("Must provide -cgroup-root")
	}

	stderr_chan := make(chan string, 1)
	defer close(stderr_chan)
	finish_chan := make(chan bool)
	defer close(finish_chan)

	go CopyChanToPipe(stderr_chan, os.Stderr)

	var cmd *exec.Cmd

	if len(flag.Args()) > 0 {
		// Set up subprocess
		cmd = exec.Command(flag.Args()[0], flag.Args()[1:]...)

		logger.Print("Running ", flag.Args())

		// Child process will use our stdin and stdout pipes
		// (we close our copies below)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout

		// Forward SIGINT and SIGTERM to inner process
		term := make(chan os.Signal, 1)
		go func(sig <-chan os.Signal) {
			catch := <-sig
			if cmd.Process != nil {
				cmd.Process.Signal(catch)
			}
			logger.Print("caught signal: ", catch)
		}(term)
		signal.Notify(term, syscall.SIGTERM)
		signal.Notify(term, syscall.SIGINT)

		// Funnel stderr through our channel
		stderr_pipe, err := cmd.StderrPipe()
		if err != nil {
			logger.Fatal(err)
		}
		go CopyPipeToChan(stderr_pipe, stderr_chan, finish_chan)

		// Run subprocess
		if err := cmd.Start(); err != nil {
			logger.Fatal(err)
		}

		// Close stdin/stdout in this (parent) process
		os.Stdin.Close()
		os.Stdout.Close()
	}

	// Read the cid file
	var container_id string
	if cgroup_cidfile != "" {
		// wait up to 'wait' seconds for the cid file to appear
		ok := false
		var i time.Duration
		for i = 0; i < time.Duration(wait)*time.Second; i += (100 * time.Millisecond) {
			cid, err := OpenAndReadAll(cgroup_cidfile, nil)
			if err == nil && len(cid) > 0 {
				ok = true
				container_id = string(cid)
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if !ok {
			logger.Printf("Could not read cid file %s", cgroup_cidfile)
		}
	}

	stop_poll_chan := make(chan bool, 1)
	cgroup := Cgroup{cgroup_root, cgroup_parent, container_id}
	go PollCgroupStats(cgroup, stderr_chan, poll, stop_poll_chan)

	// When the child exits, tell the polling goroutine to stop.
	defer func() { stop_poll_chan <- true }()

	// Wait for CopyPipeToChan to consume child's stderr pipe
	<-finish_chan

	return cmd.Wait()
}

func main() {
	logger := log.New(os.Stderr, "crunchstat: ", 0)
	if err := run(logger); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			// The program has exited with an exit code != 0

			// This works on both Unix and
			// Windows. Although package syscall is
			// generally platform dependent, WaitStatus is
			// defined for both Unix and Windows and in
			// both cases has an ExitStatus() method with
			// the same signature.
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				os.Exit(status.ExitStatus())
			}
		} else {
			logger.Fatalf("cmd.Wait: %v", err)
		}
	}
}
