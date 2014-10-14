package main

import (
	"bufio"
	"flag"
	"errors"
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
	{
		content, err := ioutil.ReadAll(in)
		if err != nil && log_chan != nil {
			log_chan <- fmt.Sprintf("crunchstat: read %s: %s", filename, err)
		}
		return content, err
	}
}

func FindStat(stderr chan<- string, cgroup Cgroup, statgroup string, stat string, verbose bool) string {
	var path string
	path = fmt.Sprintf("%s/%s/%s/%s/%s", cgroup.root, statgroup, cgroup.parent, cgroup.cid, stat)
	if _, err := os.Stat(path); err != nil {
		path = fmt.Sprintf("%s/%s/%s/%s", cgroup.root, cgroup.parent, cgroup.cid, stat)
	}
	if _, err := os.Stat(path); err != nil {
		path = fmt.Sprintf("%s/%s/%s", cgroup.root, statgroup, stat)
	}
	if _, err := os.Stat(path); err != nil {
		path = fmt.Sprintf("%s/%s", cgroup.root, stat)
	}
	if _, err := os.Stat(path); err != nil {
		stderr <- fmt.Sprintf("crunchstat: did not find stats file (root %s, parent %s, cid %s, statgroup %s, stat %s)", cgroup.root, cgroup.parent, cgroup.cid, statgroup, stat)
		return ""
	}
	if verbose {
		stderr <- fmt.Sprintf("crunchstat: reading stats from %s", path)
	}
	return path
}

func GetContainerNetStats(stderr chan<- string, cgroup Cgroup) (io.Reader, error) {
	procsFilename := FindStat(stderr, cgroup, "cpuacct", "cgroup.procs", false)
	procsFile, err := os.Open(procsFilename)
	if err != nil {
		stderr <- fmt.Sprintf("crunchstat: open %s: %s", procsFilename, err)
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

type NetSample struct {
	sampleTime time.Time
	txBytes    int64
	rxBytes    int64
}

func DoNetworkStats(stderr chan<- string, cgroup Cgroup, lastStat map[string]NetSample) (map[string]NetSample) {
	sampleTime := time.Now()
	stats, err := GetContainerNetStats(stderr, cgroup)
	if err != nil { return lastStat }

	if lastStat == nil {
		lastStat = make(map[string]NetSample)
	}
	scanner := bufio.NewScanner(stats)
	Iface: for scanner.Scan() {
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
		nextSample := NetSample{}
		nextSample.sampleTime = sampleTime
		nextSample.txBytes = tx
		nextSample.rxBytes = rx
		var delta string
		if lastSample, ok := lastStat[ifName]; ok {
			interval := nextSample.sampleTime.Sub(lastSample.sampleTime).Seconds()
			delta = fmt.Sprintf(" -- interval %.4f seconds %d tx %d rx",
				interval,
				tx - lastSample.txBytes,
				rx - lastSample.rxBytes)
		}
		stderr <- fmt.Sprintf("crunchstat: net:%s %d tx %d rx%s",
			ifName, tx, rx, delta)
		lastStat[ifName] = nextSample
	}
	return lastStat
}

func PollCgroupStats(cgroup Cgroup, stderr chan string, poll int64, stop_poll_chan <-chan bool) {
	var last_user int64 = -1
	var last_sys int64 = -1
	var last_cpucount int64 = 0

	type Disk struct {
		last_read  int64
		next_read  int64
		last_write int64
		next_write int64
	}

	disk := make(map[string]*Disk)

	user_hz := float64(C.sysconf(C._SC_CLK_TCK))

	cpuacct_stat := FindStat(stderr, cgroup, "cpuacct", "cpuacct.stat", true)
	blkio_io_service_bytes := FindStat(stderr, cgroup, "blkio", "blkio.io_service_bytes", true)
	cpuset_cpus := FindStat(stderr, cgroup, "cpuset", "cpuset.cpus", true)
	memory_stat := FindStat(stderr, cgroup, "memory", "memory.stat", true)
	lastNetStat := DoNetworkStats(stderr, cgroup, nil)

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
		bedtime := time.Now()
		select {
		case <-stop_poll_chan:
			return
		case <-poll_chan:
			// Emit stats, then select again.
		}
		morning := time.Now()
		elapsed := morning.Sub(bedtime).Seconds()
		if cpuset_cpus != "" {
			b, err := OpenAndReadAll(cpuset_cpus, stderr)
			if err != nil {
				// cgroup probably gone -- skip other stats too.
				continue
			}
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
			last_cpucount = cpus
		}
		if cpuacct_stat != "" {
			b, err := OpenAndReadAll(cpuacct_stat, stderr)
			if err != nil {
				// Next time around, last_user would
				// be >1 interval old, so stats will
				// be incorrect. Start over instead.
				last_user = -1

				// cgroup probably gone -- skip other stats too.
				continue
			}
			var next_user int64
			var next_sys int64
			fmt.Sscanf(string(b), "user %d\nsystem %d", &next_user, &next_sys)

			delta := ""
			if elapsed > 0 && last_user != -1 {
				delta = fmt.Sprintf(" -- interval %.4f seconds %.4f user %.4f sys",
					elapsed,
					float64(next_user - last_user) / user_hz,
					float64(next_sys - last_sys) / user_hz)
			}
			stderr <- fmt.Sprintf("crunchstat: cpu %.4f user %.4f sys %d cpus%s",
				float64(next_user) / user_hz,
				float64(next_sys) / user_hz,
				last_cpucount,
				delta)
			last_user = next_user
			last_sys = next_sys
		}
		if blkio_io_service_bytes != "" {
			c, err := os.Open(blkio_io_service_bytes)
			if err != nil {
				stderr <- fmt.Sprintf("open %s: %s", blkio_io_service_bytes, err)
				// cgroup probably gone -- skip other stats too.
				continue
			}
			defer c.Close()
			b := bufio.NewScanner(c)
			var device, op string
			var next int64
			for b.Scan() {
				if _, err := fmt.Sscanf(string(b.Text()), "%s %s %d", &device, &op, &next); err != nil {
					continue
				}
				if disk[device] == nil {
					disk[device] = new(Disk)
				}
				if op == "Read" {
					disk[device].last_read = disk[device].next_read
					disk[device].next_read = next
					if disk[device].last_read > 0 && (disk[device].next_read != disk[device].last_read) {
						stderr <- fmt.Sprintf("crunchstat: blkio.io_service_bytes %s read %v", device, disk[device].next_read-disk[device].last_read)
					}
				}
				if op == "Write" {
					disk[device].last_write = disk[device].next_write
					disk[device].next_write = next
					if disk[device].last_write > 0 && (disk[device].next_write != disk[device].last_write) {
						stderr <- fmt.Sprintf("crunchstat: blkio.io_service_bytes %s write %v", device, disk[device].next_write-disk[device].last_write)
					}
				}
			}
		}

		if memory_stat != "" {
			c, err := os.Open(memory_stat)
			if err != nil {
				stderr <- fmt.Sprintf("open %s: %s", memory_stat, err)
				// cgroup probably gone -- skip other stats too.
				continue
			}
			b := bufio.NewScanner(c)
			var stat string
			var val int64
			for b.Scan() {
				if _, err := fmt.Sscanf(string(b.Text()), "%s %d", &stat, &val); err == nil {
					if stat == "rss" {
						stderr <- fmt.Sprintf("crunchstat: memory.stat rss %v", val)
					}
				}
			}
			c.Close()
		}

		lastNetStat = DoNetworkStats(stderr, cgroup, lastNetStat)
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
