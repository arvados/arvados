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

func FindStat(cgroup_root string, cgroup_parent string, container_id string, statgroup string, stat string) string {
	var path string
	path = fmt.Sprintf("%s/%s/%s/%s/%s.%s", cgroup_root, statgroup, cgroup_parent, container_id, statgroup, stat)
	if _, err := os.Stat(path); err == nil {
		return path
	}
	path = fmt.Sprintf("%s/%s/%s/%s.%s", cgroup_root, cgroup_parent, container_id, statgroup, stat)
	if _, err := os.Stat(path); err == nil {
		return path
	}
	path = fmt.Sprintf("%s/%s/%s.%s", cgroup_root, statgroup, statgroup, stat)
	if _, err := os.Stat(path); err == nil {
		return path
	}
	path = fmt.Sprintf("%s/%s.%s", cgroup_root, statgroup, stat)
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return ""
}

func PollCgroupStats(cgroup_root string, cgroup_parent string, container_id string, stderr chan string, poll int64, stop_poll_chan <-chan bool) {
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

	cpuacct_stat := FindStat(cgroup_root, cgroup_parent, container_id, "cpuacct", "stat")
	blkio_io_service_bytes := FindStat(cgroup_root, cgroup_parent, container_id, "blkio", "io_service_bytes")
	cpuset_cpus := FindStat(cgroup_root, cgroup_parent, container_id, "cpuset", "cpus")
	memory_stat := FindStat(cgroup_root, cgroup_parent, container_id, "memory", "stat")

	if cpuacct_stat != "" {
		stderr <- fmt.Sprintf("crunchstat: reading stats from %s", cpuacct_stat)
	}
	if blkio_io_service_bytes != "" {
		stderr <- fmt.Sprintf("crunchstat: reading stats from %s", blkio_io_service_bytes)
	}
	if cpuset_cpus != "" {
		stderr <- fmt.Sprintf("crunchstat: reading stats from %s", cpuset_cpus)
	}
	if memory_stat != "" {
		stderr <- fmt.Sprintf("crunchstat: reading stats from %s", memory_stat)
	}

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
	go PollCgroupStats(cgroup_root, cgroup_parent, container_id, stderr_chan, poll, stop_poll_chan)

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
