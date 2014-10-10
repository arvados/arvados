package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
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
			log_chan <- fmt.Sprintf("open %s: %s", filename, err)
		}
		return nil, err
	}
	defer in.Close()
	{
		content, err := ioutil.ReadAll(in)
		if err != nil && log_chan != nil {
			log_chan <- fmt.Sprintf("read %s: %s", filename, err)
		}
		return content, err
	}
}

func FindStat(stderr chan<- string, cgroup Cgroup, statgroup string, stat string) string {
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
	stderr <- fmt.Sprintf("crunchstat: reading stats from %s", path)
	return path
}

func SetNetworkNamespace(stderr chan<- string, procsFilename string) (string) {
	// Not supported yet -- we'll just report host-wide network stats.
	return "host"

	if procsFilename == "" { return "host" }
	procsFile, err := os.Open(procsFilename)
	if err != nil {
		stderr <- fmt.Sprintf("crunchstat: open %s: %s", procsFilename, err)
		return "host"
	}
	defer procsFile.Close()
	reader := bufio.NewScanner(procsFile)
	for reader.Scan() {
		taskPid := reader.Text()
		netnsFilename := fmt.Sprintf("/proc/%s/ns/net", taskPid)
		netnsFile, err := os.Open(netnsFilename)
		if err != nil {
			stderr <- fmt.Sprintf("crunchstat: open %s: %s", netnsFilename, err)
			continue
		}
		defer netnsFile.Close()

		// syscall.Setns() doesn't exist yet, and doesn't work
		// from a multithreaded program yet.
		//
		// if _, err2 := syscall.Setns(netnsFile.Fd()); err != nil {
		// 	stderr <- fmt.Sprintf("crunchstat: Setns: %s", err2)
		// 	continue
		// }
		return "task"
	}
	return "host"
}

type NetSample struct {
	sampleTime time.Time
	txBytes    int64
	rxBytes    int64
}

func DoNetworkStats(stderr chan<- string, procsFilename string, lastStat map[string]NetSample) (map[string]NetSample) {
	statScope := SetNetworkNamespace(stderr, procsFilename)

	ifDirs, err := filepath.Glob("/sys/class/net/*")
	if err != nil {
		stderr <- fmt.Sprintf("crunchstat: could not list interfaces", err)
		return lastStat
	}
	if lastStat == nil {
		lastStat = make(map[string]NetSample)
	}
	for _, ifDir := range ifDirs {
		ifName := filepath.Base(ifDir)
		tx_s, tx_err := OpenAndReadAll(fmt.Sprintf("/sys/class/net/%s/statistics/tx_bytes", ifName), stderr)
		rx_s, rx_err := OpenAndReadAll(fmt.Sprintf("/sys/class/net/%s/statistics/rx_bytes", ifName), stderr)
		if rx_err != nil || tx_err != nil {
			return nil
		}
		nextSample := NetSample{}
		nextSample.sampleTime = time.Now()
		fmt.Sscanf(string(tx_s), "%d", &nextSample.txBytes)
		fmt.Sscanf(string(rx_s), "%d", &nextSample.rxBytes)
		if lastSample, ok := lastStat[ifName]; ok {
			stderr <- fmt.Sprintf("crunchstat: %s net %s tx %d rx %d interval %.4f",
				statScope,
				ifName,
				nextSample.txBytes - lastSample.txBytes,
				nextSample.rxBytes - lastSample.rxBytes,
				nextSample.sampleTime.Sub(lastSample.sampleTime).Seconds())
		}
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

	cpuacct_stat := FindStat(stderr, cgroup, "cpuacct", "cpuacct.stat")
	blkio_io_service_bytes := FindStat(stderr, cgroup, "blkio", "blkio.io_service_bytes")
	cpuset_cpus := FindStat(stderr, cgroup, "cpuset", "cpuset.cpus")
	memory_stat := FindStat(stderr, cgroup, "memory", "memory.stat")
	procs := FindStat(stderr, cgroup, "cpuacct", "cgroup.procs")
	lastNetStat := DoNetworkStats(stderr, procs, nil)

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

			if elapsed > 0 && last_user != -1 {
				user_diff := next_user - last_user
				sys_diff := next_sys - last_sys
				// {*_diff} == {1/user_hz}-second
				// ticks of CPU core consumed in an
				// {elapsed}-second interval.
				//
				// We report this as CPU core usage
				// (i.e., 1.0 == one pegged core). We
				// also report the number of cores
				// (maximum possible usage).
				user := float64(user_diff) / elapsed / user_hz
				sys := float64(sys_diff) / elapsed / user_hz

				stderr <- fmt.Sprintf("crunchstat: cpuacct.stat user %.4f sys %.4f cpus %d interval %.4f", user, sys, last_cpucount, elapsed)
			}

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

		lastNetStat = DoNetworkStats(stderr, procs, lastNetStat)
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
