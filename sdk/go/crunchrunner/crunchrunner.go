package main

import (
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	//"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"errors"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

func getRecord(api arvadosclient.ArvadosClient, rsc, uuid string) (r arvadosclient.Dict) {
	r = make(arvadosclient.Dict)
	err := api.Get(rsc, uuid, nil, &r)
	if err != nil {
		log.Fatal(err)
	}
	return r
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

func setupCommand(cmd *exec.Cmd, taskp map[string]interface{}, keepmount, outdir string) error {
	var err error

	if taskp["task.vwd"] != nil {
		// Set up VWD symlinks in outdir
		// TODO
	}

	if taskp["task.stdin"] != nil {
		stdin, ok := taskp["task.stdin"].(string)
		if !ok {
			return errors.New("Could not cast task.stdin to string")
		}
		// Set up stdin redirection
		cmd.Stdin, err = os.Open(keepmount + "/" + stdin)
		if err != nil {
			log.Fatal(err)
		}
	}

	if taskp["task.stdout"] != nil {
		stdout, ok := taskp["task.stdout"].(string)
		if !ok {
			return errors.New("Could not cast task.stdout to string")
		}

		// Set up stdout redirection
		cmd.Stdout, err = os.Open(outdir + "/" + stdout)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		cmd.Stdout = os.Stdout
	}

	if taskp["task.env"] != nil {
		taskenv, ok := taskp["task.env"].(map[string]interface{})
		if !ok {
			return errors.New("Could not cast task.env to map")
		}

		// Set up subprocess environment
		cmd.Env = os.Environ()
		for k, v := range taskenv {
			var vstr string
			vstr, ok = v.(string)
			if !ok {
				return errors.New("Could not cast environment value to string")
			}
			cmd.Env = append(cmd.Env, k+"="+vstr)
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

func inCodes(code int, codes interface{}) bool {
	if codes != nil {
		codesArray, ok := codes.([]interface{})
		if !ok {
			return false
		}
		for _, c := range codesArray {
			var num float64
			num, ok = c.(float64)
			if ok && code == int(num) {
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

func runner(api arvadosclient.ArvadosClient,
	jobUuid, taskUuid, tmpdir, keepmount string, jobStruct,
	taskStruct arvadosclient.Dict) error {

	var err error
	var ok bool
	var jobp, taskp map[string]interface{}
	jobp, ok = jobStruct["script_parameters"].(map[string]interface{})
	if !ok {
		return errors.New("Could not cast job script_parameters to map")
	}

	taskp, ok = taskStruct["parameters"].(map[string]interface{})
	if !ok {
		return errors.New("Could not cast task parameters to map")
	}

	// If this is task 0 and there are multiple tasks, dispatch subtasks
	// and exit.
	if taskStruct["sequence"] == 0.0 {
		var tasks []interface{}
		tasks, ok = jobp["tasks"].([]interface{})
		if !ok {
			return errors.New("Could not cast tasks to array")
		}

		if len(tasks) == 1 {
			taskp = tasks[0].(map[string]interface{})
		} else {
			for task := range tasks {
				err := api.Call("POST", "job_tasks", "", "",
					arvadosclient.Dict{
						"job_uuid":                 jobUuid,
						"created_by_job_task_uuid": "",
						"sequence":                 1,
						"parameters":               task},
					nil)
				if err != nil {
					return TempFail{err}
				}
			}
			err = api.Call("PUT", "job_tasks", taskUuid, "",
				arvadosclient.Dict{
					"job_task": arvadosclient.Dict{
						"output":   "",
						"success":  true,
						"progress": 1.0}},
				nil)
			return nil
		}
	}

	// Set up subprocess
	var commandline []string
	var commandsarray []interface{}

	commandsarray, ok = taskp["command"].([]interface{})
	if !ok {
		return errors.New("Could not cast commands to array")
	}

	for _, c := range commandsarray {
		var cstr string
		cstr, ok = c.(string)
		if !ok {
			return errors.New("Could not cast command argument to string")
		}
		commandline = append(commandline, cstr)
	}
	cmd := exec.Command(commandline[0], commandline[1:]...)

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

	if inCodes(exitCode, taskp["task.successCodes"]) {
		status = success
	} else if inCodes(exitCode, taskp["task.permanentFailCodes"]) {
		status = permfail
	} else if inCodes(exitCode, taskp["task.temporaryFailCodes"]) {
		os.Exit(TASK_TEMPFAIL)
	} else if cmd.ProcessState.Success() {
		status = success
	} else {
		status = permfail
	}

	// Upload output directory
	// TODO

	// Set status
	err = api.Call("PUT", "job_tasks", taskUuid, "",
		arvadosclient.Dict{
			"job_task": arvadosclient.Dict{
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

	jobStruct := getRecord(api, "jobs", jobUuid)
	taskStruct := getRecord(api, "job_tasks", taskUuid)

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
