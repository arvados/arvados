package arvadostest

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

var authSettings = make(map[string]string)

func ResetEnv() {
	for k, v := range authSettings {
		os.Setenv(k, v)
	}
}

func ParseAuthSettings(authScript []byte) {
	scanner := bufio.NewScanner(bytes.NewReader(authScript))
	for scanner.Scan() {
		line := scanner.Text()
		if 0 != strings.Index(line, "export ") {
			log.Printf("Ignoring: %v", line)
			continue
		}
		toks := strings.SplitN(strings.Replace(line, "export ", "", 1), "=", 2)
		if len(toks) == 2 {
			authSettings[toks[0]] = toks[1]
		} else {
			log.Fatalf("Could not parse: %v", line)
		}
	}
	log.Printf("authSettings: %v", authSettings)
}

var pythonTestDir string = ""

func chdirToPythonTests() {
	if pythonTestDir != "" {
		if err := os.Chdir(pythonTestDir); err != nil {
			log.Fatalf("chdir %s: %s", pythonTestDir, err)
		}
		return
	}
	for {
		if err := os.Chdir("sdk/python/tests"); err == nil {
			pythonTestDir, err = os.Getwd()
			return
		}
		if parent, err := os.Getwd(); err != nil || parent == "/" {
			log.Fatalf("sdk/python/tests/ not found in any ancestor")
		}
		if err := os.Chdir(".."); err != nil {
			log.Fatal(err)
		}
	}
}

func StartAPI() {
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	chdirToPythonTests()

	cmd := exec.Command("python", "run_test_server.py", "start", "--auth", "admin")
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}
	go io.Copy(os.Stderr, stderr)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err = cmd.Start(); err != nil {
		log.Fatal(err)
	}
	var authScript []byte
	if authScript, err = ioutil.ReadAll(stdout); err != nil {
		log.Fatal(err)
	}
	if err = cmd.Wait(); err != nil {
		log.Fatal(err)
	}
	ParseAuthSettings(authScript)
	ResetEnv()
}

func StopAPI() {
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	chdirToPythonTests()

	exec.Command("python", "run_test_server.py", "stop").Run()
}

// StartKeep is used to start keep servers
// with needMore = false and enforcePermissions = false
func StartKeep() {
	StartKeepWithParams(2, false)
}

// StartKeepWithParams is used to start keep servers while specifying
// numKeepServers and enforcePermissions parameters.
func StartKeepWithParams(numKeepServers int, enforcePermissions bool) {
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	chdirToPythonTests()

	cmdArgs := []string{"run_test_server.py", "start_keep"}
	if numKeepServers != 2 {
		cmdArgs = append(cmdArgs, "--num-keep-servers", strconv.Itoa(numKeepServers))
	}
	if enforcePermissions {
		cmdArgs = append(cmdArgs, "--keep-enforce-permissions")
	}

	cmd := exec.Command("python", cmdArgs...)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatalf("Setting up stderr pipe: %s", err)
	}
	go io.Copy(os.Stderr, stderr)
	if err := cmd.Run(); err != nil {
		panic(fmt.Sprintf("'python run_test_server.py start_keep' returned error %s", err))
	}
}

func StopKeep() {
	StopKeepServers(2)
}

// StopKeepServers is used to stop keep servers while specifying numKeepServers
func StopKeepServers(numKeepServers int) {
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	chdirToPythonTests()

	cmdArgs := []string{"run_test_server.py", "stop_keep"}

	if numKeepServers != 2 {
		cmdArgs = append(cmdArgs, "--num-keep-servers", strconv.Itoa(numKeepServers))
	}

	exec.Command("python", cmdArgs...)
}
