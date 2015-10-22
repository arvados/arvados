package main

import (
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
)

type TaskDef struct {
	command            []string          `json:"command"`
	env                map[string]string `json:"task.env"`
	stdin              string            `json:"task.stdin"`
	stdout             string            `json:"task.stdout"`
	vwd                map[string]string `json:"task.vwd"`
	successCodes       []int             `json:"task.successCodes"`
	permanentFailCodes []int             `json:"task.permanentFailCodes"`
	temporaryFailCodes []int             `json:"task.temporaryFailCodes"`
}

type Tasks struct {
	tasks []TaskDef `json:"script_parameters"`
}

type Job struct {
	script_parameters Tasks `json:"script_parameters"`
}

type Task struct {
	job_uuid                 string  `json:"job_uuid"`
	created_by_job_task_uuid string  `json:"created_by_job_task_uuid"`
	parameters               TaskDef `json:"parameters"`
	sequence                 int     `json:"sequence"`
	output                   string  `json:"output"`
	success                  bool    `json:"success"`
	progress                 float32 `json:"sequence"`
}

type IArvadosClient interface {
	Create(resourceType string, parameters arvadosclient.Dict, output interface{}) error
	Update(resourceType string, uuid string, parameters arvadosclient.Dict, output interface{}) (err error)
}

func setupDirectories(crunchtmpdir, taskUuid string) (tmpdir, outdir string, err error) {
	tmpdir = crunchtmpdir + "/tmpdir"
	err = os.Mkdir(tmpdir, 0700)
	if err != nil {
		return "", "", err
	}

	outdir = crunchtmpdir + "/outdir"
	err = os.Mkdir(outdir, 0700)
	if err != nil {
		return "", "", err
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

func setupCommand(cmd *exec.Cmd, taskp TaskDef, outdir string, replacements map[string]string) (stdin, stdout string, err error) {
	if taskp.vwd != nil {
		for k, v := range taskp.vwd {
			v = substitute(v, replacements)
			err = checkOutputFilename(outdir, k)
			if err != nil {
				return "", "", err
			}
			os.Symlink(v, outdir+"/"+k)
		}
	}

	if taskp.stdin != "" {
		// Set up stdin redirection
		stdin = substitute(taskp.stdin, replacements)
		cmd.Stdin, err = os.Open(stdin)
		if err != nil {
			return "", "", err
		}
	}

	if taskp.stdout != "" {
		err = checkOutputFilename(outdir, taskp.stdout)
		if err != nil {
			return "", "", err
		}
		// Set up stdout redirection
		stdout = outdir + "/" + taskp.stdout
		cmd.Stdout, err = os.Create(stdout)
		if err != nil {
			return "", "", err
		}
	} else {
		cmd.Stdout = os.Stdout
	}

	if taskp.env != nil {
		// Set up subprocess environment
		cmd.Env = os.Environ()
		for k, v := range taskp.env {
			v = substitute(v, replacements)
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}
	return stdin, stdout, nil
}

func setupSignals(cmd *exec.Cmd) chan os.Signal {
	// Set up signal handlers
	// Forward SIGINT, SIGTERM and SIGQUIT to inner process
	sigChan := make(chan os.Signal, 1)
	go func(sig <-chan os.Signal) {
		catch := <-sig
		cmd.Process.Signal(catch)
	}(sigChan)
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

func runner(api IArvadosClient,
	kc IKeepClient,
	jobUuid, taskUuid, crunchtmpdir, keepmount string,
	jobStruct Job, taskStruct Task) error {

	var err error
	taskp := taskStruct.parameters

	// If this is task 0 and there are multiple tasks, dispatch subtasks
	// and exit.
	if taskStruct.sequence == 0 {
		if len(jobStruct.script_parameters.tasks) == 1 {
			taskp = jobStruct.script_parameters.tasks[0]
		} else {
			for _, task := range jobStruct.script_parameters.tasks {
				err := api.Create("job_tasks",
					map[string]interface{}{
						"job_task": Task{job_uuid: jobUuid,
							created_by_job_task_uuid: taskUuid,
							sequence:                 1,
							parameters:               task}},
					nil)
				if err != nil {
					return TempFail{err}
				}
			}
			err = api.Update("job_tasks", taskUuid,
				map[string]interface{}{
					"job_task": Task{
						output:   "",
						success:  true,
						progress: 1.0}},
				nil)
			return nil
		}
	}

	var tmpdir, outdir string
	tmpdir, outdir, err = setupDirectories(crunchtmpdir, taskUuid)
	if err != nil {
		return TempFail{err}
	}

	replacements := map[string]string{
		"$(task.tmpdir)": tmpdir,
		"$(task.outdir)": outdir,
		"$(task.keep)":   keepmount}

	// Set up subprocess
	for k, v := range taskp.command {
		taskp.command[k] = substitute(v, replacements)
	}

	cmd := exec.Command(taskp.command[0], taskp.command[1:]...)

	cmd.Dir = outdir

	var stdin, stdout string
	stdin, stdout, err = setupCommand(cmd, taskp, outdir, replacements)
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
	log.Printf("Running %v%v%v", cmd.Args, stdin, stdout)

	err = cmd.Start()

	signals := setupSignals(cmd)
	err = cmd.Wait()
	signal.Stop(signals)

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

	if inCodes(exitCode, taskp.permanentFailCodes) {
		success = false
	} else if inCodes(exitCode, taskp.temporaryFailCodes) {
		return TempFail{fmt.Errorf("Process tempfail with exit code %v", exitCode)}
	} else if inCodes(exitCode, taskp.successCodes) || cmd.ProcessState.Success() {
		success = true
	} else {
		success = false
	}

	// Upload output directory
	manifest, err := WriteTree(kc, outdir)
	if err != nil {
		return TempFail{err}
	}

	// Set status
	err = api.Update("job_tasks", taskUuid,
		map[string]interface{}{
			"job_task": Task{
				output:   manifest,
				success:  success,
				progress: 1}},
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

	jobUuid := os.Getenv("JOB_UUID")
	taskUuid := os.Getenv("TASK_UUID")
	tmpdir := os.Getenv("TASK_WORK")
	keepmount := os.Getenv("TASK_KEEPMOUNT")

	var jobStruct Job
	var taskStruct Task

	err = api.Get("jobs", jobUuid, nil, &jobStruct)
	if err != nil {
		log.Fatal(err)
	}
	err = api.Get("job_tasks", taskUuid, nil, &taskStruct)
	if err != nil {
		log.Fatal(err)
	}

	var kc IKeepClient
	kc, err = keepclient.MakeKeepClient(&api)
	err = runner(api, kc, jobUuid, taskUuid, tmpdir, keepmount, jobStruct, taskStruct)

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
