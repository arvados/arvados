package main

import (
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	//"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

type TaskDef struct {
	commands           []string          `json:"commands"`
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

func setupDirectories(tmpdir string) (outdir string, err error) {
	err = os.Chdir(tmpdir)
	if err != nil {
		return "", err
	}

	err = os.Mkdir("tmpdir", 0700)
	if err != nil {
		return "", err
	}

	err = os.Mkdir("outdir", 0700)
	if err != nil {
		return "", err
	}

	os.Chdir("outdir")
	if err != nil {
		return "", err
	}

	outdir, err = os.Getwd()
	if err != nil {
		return "", err
	}

	return outdir, nil
}

func setupCommand(cmd *exec.Cmd, taskp TaskDef, keepmount, outdir string) error {
	var err error

	//if taskp.vwd != nil {
	// Set up VWD symlinks in outdir
	// TODO
	//}

	if taskp.stdin != "" {
		// Set up stdin redirection
		cmd.Stdin, err = os.Open(keepmount + "/" + taskp.stdin)
		if err != nil {
			log.Fatal(err)
		}
	}

	if taskp.stdout != "" {
		// Set up stdout redirection
		cmd.Stdout, err = os.Create(outdir + "/" + taskp.stdout)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		cmd.Stdout = os.Stdout
	}

	if taskp.env != nil {
		// Set up subprocess environment
		cmd.Env = os.Environ()
		for k, v := range taskp.env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}
	return nil
}

func setupSignals(cmd *exec.Cmd) {
	// Set up signal handlers
	// Forward SIGINT, SIGTERM and SIGQUIT to inner process
	sigChan := make(chan os.Signal, 1)
	go func(sig <-chan os.Signal) {
		catch := <-sig
		if cmd.Process != nil {
			cmd.Process.Signal(catch)
		}
	}(sigChan)
	signal.Notify(sigChan, syscall.SIGTERM)
	signal.Notify(sigChan, syscall.SIGINT)
	signal.Notify(sigChan, syscall.SIGQUIT)
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

type TempFail struct{ InnerError error }
type PermFail struct{}

func (s TempFail) Error() string {
	return s.InnerError.Error()
}

func (s PermFail) Error() string {
	return "PermFail"
}

func runner(api arvadosclient.IArvadosClient,
	jobUuid, taskUuid, tmpdir, keepmount string,
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

	// Set up subprocess
	cmd := exec.Command(taskp.commands[0], taskp.commands[1:]...)

	var outdir string
	outdir, err = setupDirectories(tmpdir)
	if err != nil {
		return TempFail{err}
	}

	cmd.Dir = outdir

	err = setupCommand(cmd, taskp, keepmount, outdir)
	if err != nil {
		return err
	}

	setupSignals(cmd)

	// Run subprocess and wait for it to complete
	log.Printf("Running %v", cmd.Args)

	err = cmd.Run()

	if err != nil {
		return TempFail{err}
	}

	const success = 1
	const permfail = 2
	const tempfail = 2
	var status int

	exitCode := cmd.ProcessState.Sys().(syscall.WaitStatus).ExitStatus()

	if inCodes(exitCode, taskp.successCodes) {
		status = success
	} else if inCodes(exitCode, taskp.permanentFailCodes) {
		status = permfail
	} else if inCodes(exitCode, taskp.temporaryFailCodes) {
		os.Exit(TASK_TEMPFAIL)
	} else if cmd.ProcessState.Success() {
		status = success
	} else {
		status = permfail
	}

	// Upload output directory
	// TODO

	// Set status
	err = api.Update("job_tasks", taskUuid,
		map[string]interface{}{
			"job_task": map[string]interface{}{
				"output":   "",
				"success":  status == success,
				"progress": 1.0}},
		nil)
	if err != nil {
		return TempFail{err}
	}

	if status == success {
		return nil
	} else {
		return PermFail{}
	}
}

func main() {
	syscall.Umask(0077)

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

	err = runner(api, jobUuid, taskUuid, tmpdir, keepmount, jobStruct, taskStruct)

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
