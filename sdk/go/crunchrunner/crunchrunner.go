// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
)

type TaskDef struct {
	Command            []string          `json:"command"`
	Env                map[string]string `json:"task.env"`
	Stdin              string            `json:"task.stdin"`
	Stdout             string            `json:"task.stdout"`
	Stderr             string            `json:"task.stderr"`
	Vwd                map[string]string `json:"task.vwd"`
	SuccessCodes       []int             `json:"task.successCodes"`
	PermanentFailCodes []int             `json:"task.permanentFailCodes"`
	TemporaryFailCodes []int             `json:"task.temporaryFailCodes"`
	KeepTmpOutput      bool              `json:"task.keepTmpOutput"`
}

type Tasks struct {
	Tasks []TaskDef `json:"tasks"`
}

type Job struct {
	ScriptParameters Tasks `json:"script_parameters"`
}

type Task struct {
	JobUUID              string  `json:"job_uuid"`
	CreatedByJobTaskUUID string  `json:"created_by_job_task_uuid"`
	Parameters           TaskDef `json:"parameters"`
	Sequence             int     `json:"sequence"`
	Output               string  `json:"output"`
	Success              bool    `json:"success"`
	Progress             float32 `json:"sequence"`
}

type IArvadosClient interface {
	Create(resourceType string, parameters arvadosclient.Dict, output interface{}) error
	Update(resourceType string, uuid string, parameters arvadosclient.Dict, output interface{}) (err error)
}

func setupDirectories(crunchtmpdir, taskUUID string, keepTmp bool) (tmpdir, outdir string, err error) {
	tmpdir = crunchtmpdir + "/tmpdir"
	err = os.Mkdir(tmpdir, 0700)
	if err != nil {
		return "", "", err
	}

	if keepTmp {
		outdir = os.Getenv("TASK_KEEPMOUNT_TMP")
	} else {
		outdir = crunchtmpdir + "/outdir"
		err = os.Mkdir(outdir, 0700)
		if err != nil {
			return "", "", err
		}
	}

	return tmpdir, outdir, nil
}

func checkOutputFilename(outdir, fn string) error {
	if strings.HasPrefix(fn, "/") || strings.HasSuffix(fn, "/") {
		return fmt.Errorf("Path must not start or end with '/'")
	}
	if strings.Index("../", fn) != -1 {
		return fmt.Errorf("Path must not contain '../'")
	}

	sl := strings.LastIndex(fn, "/")
	if sl != -1 {
		os.MkdirAll(outdir+"/"+fn[0:sl], 0777)
	}
	return nil
}

func copyFile(dst, src string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func setupCommand(cmd *exec.Cmd, taskp TaskDef, outdir string, replacements map[string]string) (stdin, stdout, stderr string, err error) {
	if taskp.Vwd != nil {
		for k, v := range taskp.Vwd {
			v = substitute(v, replacements)
			err = checkOutputFilename(outdir, k)
			if err != nil {
				return "", "", "", err
			}
			if taskp.KeepTmpOutput {
				err = copyFile(v, outdir+"/"+k)
			} else {
				err = os.Symlink(v, outdir+"/"+k)
			}
			if err != nil {
				return "", "", "", err
			}
		}
	}

	if taskp.Stdin != "" {
		// Set up stdin redirection
		stdin = substitute(taskp.Stdin, replacements)
		cmd.Stdin, err = os.Open(stdin)
		if err != nil {
			return "", "", "", err
		}
	}

	if taskp.Stdout != "" {
		err = checkOutputFilename(outdir, taskp.Stdout)
		if err != nil {
			return "", "", "", err
		}
		// Set up stdout redirection
		stdout = outdir + "/" + taskp.Stdout
		cmd.Stdout, err = os.Create(stdout)
		if err != nil {
			return "", "", "", err
		}
	} else {
		cmd.Stdout = os.Stdout
	}

	if taskp.Stderr != "" {
		err = checkOutputFilename(outdir, taskp.Stderr)
		if err != nil {
			return "", "", "", err
		}
		// Set up stderr redirection
		stderr = outdir + "/" + taskp.Stderr
		cmd.Stderr, err = os.Create(stderr)
		if err != nil {
			return "", "", "", err
		}
	} else {
		cmd.Stderr = os.Stderr
	}

	if taskp.Env != nil {
		// Set up subprocess environment
		cmd.Env = os.Environ()
		for k, v := range taskp.Env {
			v = substitute(v, replacements)
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}
	return stdin, stdout, stderr, nil
}

// Set up signal handlers.  Go sends signal notifications to a "signal
// channel".
func setupSignals(cmd *exec.Cmd) chan os.Signal {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM)
	signal.Notify(sigChan, syscall.SIGINT)
	signal.Notify(sigChan, syscall.SIGQUIT)
	return sigChan
}

func inCodes(code int, codes []int) bool {
	if codes != nil {
		for _, c := range codes {
			if code == c {
				return true
			}
		}
	}
	return false
}

const TASK_TEMPFAIL = 111

type TempFail struct{ error }
type PermFail struct{}

func (s PermFail) Error() string {
	return "PermFail"
}

func substitute(inp string, subst map[string]string) string {
	for k, v := range subst {
		inp = strings.Replace(inp, k, v, -1)
	}
	return inp
}

func getKeepTmp(outdir string) (manifest string, err error) {
	fn, err := os.Open(outdir + "/" + ".arvados#collection")
	if err != nil {
		return "", err
	}
	defer fn.Close()

	buf, err := ioutil.ReadAll(fn)
	if err != nil {
		return "", err
	}
	collection := arvados.Collection{}
	err = json.Unmarshal(buf, &collection)
	return collection.ManifestText, err
}

func runner(api IArvadosClient,
	kc IKeepClient,
	jobUUID, taskUUID, crunchtmpdir, keepmount string,
	jobStruct Job, taskStruct Task) error {

	var err error
	taskp := taskStruct.Parameters

	// If this is task 0 and there are multiple tasks, dispatch subtasks
	// and exit.
	if taskStruct.Sequence == 0 {
		if len(jobStruct.ScriptParameters.Tasks) == 1 {
			taskp = jobStruct.ScriptParameters.Tasks[0]
		} else {
			for _, task := range jobStruct.ScriptParameters.Tasks {
				err := api.Create("job_tasks",
					map[string]interface{}{
						"job_task": Task{
							JobUUID:              jobUUID,
							CreatedByJobTaskUUID: taskUUID,
							Sequence:             1,
							Parameters:           task}},
					nil)
				if err != nil {
					return TempFail{err}
				}
			}
			err = api.Update("job_tasks", taskUUID,
				map[string]interface{}{
					"job_task": map[string]interface{}{
						"output":   "",
						"success":  true,
						"progress": 1.0}},
				nil)
			return nil
		}
	}

	var tmpdir, outdir string
	tmpdir, outdir, err = setupDirectories(crunchtmpdir, taskUUID, taskp.KeepTmpOutput)
	if err != nil {
		return TempFail{err}
	}

	replacements := map[string]string{
		"$(task.tmpdir)": tmpdir,
		"$(task.outdir)": outdir,
		"$(task.keep)":   keepmount}

	log.Printf("crunchrunner: $(task.tmpdir)=%v", tmpdir)
	log.Printf("crunchrunner: $(task.outdir)=%v", outdir)
	log.Printf("crunchrunner: $(task.keep)=%v", keepmount)

	// Set up subprocess
	for k, v := range taskp.Command {
		taskp.Command[k] = substitute(v, replacements)
	}

	cmd := exec.Command(taskp.Command[0], taskp.Command[1:]...)

	cmd.Dir = outdir

	var stdin, stdout, stderr string
	stdin, stdout, stderr, err = setupCommand(cmd, taskp, outdir, replacements)
	if err != nil {
		return err
	}

	// Run subprocess and wait for it to complete
	if stdin != "" {
		stdin = " < " + stdin
	}
	if stdout != "" {
		stdout = " > " + stdout
	}
	if stderr != "" {
		stderr = " 2> " + stderr
	}
	log.Printf("Running %v%v%v%v", cmd.Args, stdin, stdout, stderr)

	var caughtSignal os.Signal
	sigChan := setupSignals(cmd)

	err = cmd.Start()
	if err != nil {
		signal.Stop(sigChan)
		return TempFail{err}
	}

	finishedSignalNotify := make(chan struct{})
	go func(sig <-chan os.Signal) {
		for sig := range sig {
			caughtSignal = sig
			cmd.Process.Signal(caughtSignal)
		}
		close(finishedSignalNotify)
	}(sigChan)

	err = cmd.Wait()
	signal.Stop(sigChan)

	close(sigChan)
	<-finishedSignalNotify

	if caughtSignal != nil {
		log.Printf("Caught signal %v", caughtSignal)
		return PermFail{}
	}

	if err != nil {
		// Run() returns ExitError on non-zero exit code, but we handle
		// that down below.  So only return if it's not ExitError.
		if _, ok := err.(*exec.ExitError); !ok {
			return TempFail{err}
		}
	}

	var success bool

	exitCode := cmd.ProcessState.Sys().(syscall.WaitStatus).ExitStatus()

	log.Printf("Completed with exit code %v", exitCode)

	if inCodes(exitCode, taskp.PermanentFailCodes) {
		success = false
	} else if inCodes(exitCode, taskp.TemporaryFailCodes) {
		return TempFail{fmt.Errorf("Process tempfail with exit code %v", exitCode)}
	} else if inCodes(exitCode, taskp.SuccessCodes) || cmd.ProcessState.Success() {
		success = true
	} else {
		success = false
	}

	// Upload output directory
	var manifest string
	if taskp.KeepTmpOutput {
		manifest, err = getKeepTmp(outdir)
	} else {
		manifest, err = WriteTree(kc, outdir)
	}
	if err != nil {
		return TempFail{err}
	}

	// Set status
	err = api.Update("job_tasks", taskUUID,
		map[string]interface{}{
			"job_task": Task{
				Output:   manifest,
				Success:  success,
				Progress: 1}},
		nil)
	if err != nil {
		return TempFail{err}
	}

	if success {
		return nil
	} else {
		return PermFail{}
	}
}

func main() {
	api, err := arvadosclient.MakeArvadosClient()
	if err != nil {
		log.Fatal(err)
	}

	jobUUID := os.Getenv("JOB_UUID")
	taskUUID := os.Getenv("TASK_UUID")
	tmpdir := os.Getenv("TASK_WORK")
	keepmount := os.Getenv("TASK_KEEPMOUNT")

	var jobStruct Job
	var taskStruct Task

	err = api.Get("jobs", jobUUID, nil, &jobStruct)
	if err != nil {
		log.Fatal(err)
	}
	err = api.Get("job_tasks", taskUUID, nil, &taskStruct)
	if err != nil {
		log.Fatal(err)
	}

	var kc IKeepClient
	kc, err = keepclient.MakeKeepClient(api)
	if err != nil {
		log.Fatal(err)
	}

	syscall.Umask(0022)
	err = runner(api, kc, jobUUID, taskUUID, tmpdir, keepmount, jobStruct, taskStruct)

	if err == nil {
		os.Exit(0)
	} else if _, ok := err.(TempFail); ok {
		log.Print(err)
		os.Exit(TASK_TEMPFAIL)
	} else if _, ok := err.(PermFail); ok {
		os.Exit(1)
	} else {
		log.Fatal(err)
	}
}
