package main

import (
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	. "gopkg.in/check.v1"
	"io"
	"io/ioutil"
	"log"
	"os"
	"syscall"
	"testing"
	"time"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	TestingT(t)
}

type TestSuite struct{}

// Gocheck boilerplate
var _ = Suite(&TestSuite{})

type ArvTestClient struct {
	c        *C
	manifest string
	success  bool
}

func (t ArvTestClient) Create(resourceType string, parameters arvadosclient.Dict, output interface{}) error {
	return nil
}

func (t ArvTestClient) Update(resourceType string, uuid string, parameters arvadosclient.Dict, output interface{}) (err error) {
	t.c.Check(resourceType, Equals, "job_tasks")
	t.c.Check(parameters, DeepEquals, arvadosclient.Dict{"job_task": Task{
		output:   t.manifest,
		success:  t.success,
		progress: 1}})
	return nil
}

func (s *TestSuite) TestSimpleRun(c *C) {
	tmpdir, _ := ioutil.TempDir("", "")
	defer func() {
		os.RemoveAll(tmpdir)
	}()

	err := runner(ArvTestClient{c, "", true},
		KeepTestClient{},
		"zzzz-8i9sb-111111111111111",
		"zzzz-ot0gb-111111111111111",
		tmpdir,
		"",
		Job{script_parameters: Tasks{[]TaskDef{TaskDef{
			command: []string{"echo", "foo"}}}}},
		Task{sequence: 0})
	c.Check(err, IsNil)
}

func checkOutput(c *C, tmpdir string) {
	file, err := os.Open(tmpdir + "/outdir/output.txt")
	c.Assert(err, IsNil)

	data := make([]byte, 100)
	var count int
	err = nil
	offset := 0
	for err == nil {
		count, err = file.Read(data[offset:])
		offset += count
	}
	c.Assert(err, Equals, io.EOF)
	c.Check(string(data[0:offset]), Equals, "foo\n")
}

func (s *TestSuite) TestSimpleRunSubtask(c *C) {
	tmpdir, _ := ioutil.TempDir("", "")
	defer func() {
		os.RemoveAll(tmpdir)
	}()

	err := runner(ArvTestClient{c,
		". d3b07384d113edec49eaa6238ad5ff00+4 0:4:output.txt\n", true},
		KeepTestClient{},
		"zzzz-8i9sb-111111111111111",
		"zzzz-ot0gb-111111111111111",
		tmpdir,
		"",
		Job{script_parameters: Tasks{[]TaskDef{
			TaskDef{command: []string{"echo", "bar"}},
			TaskDef{command: []string{"echo", "foo"}}}}},
		Task{parameters: TaskDef{
			command: []string{"echo", "foo"},
			stdout:  "output.txt"},
			sequence: 1})
	c.Check(err, IsNil)

	checkOutput(c, tmpdir)
}

func (s *TestSuite) TestRedirect(c *C) {
	tmpfile, _ := ioutil.TempFile("", "")
	tmpfile.Write([]byte("foo\n"))
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	tmpdir, _ := ioutil.TempDir("", "")
	defer func() {
		os.RemoveAll(tmpdir)
	}()

	err := runner(ArvTestClient{c,
		". d3b07384d113edec49eaa6238ad5ff00+4 0:4:output.txt\n", true},
		KeepTestClient{},
		"zzzz-8i9sb-111111111111111",
		"zzzz-ot0gb-111111111111111",
		tmpdir,
		"",
		Job{script_parameters: Tasks{[]TaskDef{TaskDef{
			command: []string{"cat"},
			stdout:  "output.txt",
			stdin:   tmpfile.Name()}}}},
		Task{sequence: 0})
	c.Check(err, IsNil)

	checkOutput(c, tmpdir)
}

func (s *TestSuite) TestEnv(c *C) {
	tmpdir, _ := ioutil.TempDir("", "")
	defer func() {
		os.RemoveAll(tmpdir)
	}()

	err := runner(ArvTestClient{c, ". d3b07384d113edec49eaa6238ad5ff00+4 0:4:output.txt\n", true},
		KeepTestClient{},
		"zzzz-8i9sb-111111111111111",
		"zzzz-ot0gb-111111111111111",
		tmpdir,
		"",
		Job{script_parameters: Tasks{[]TaskDef{TaskDef{
			command: []string{"/bin/sh", "-c", "echo $BAR"},
			stdout:  "output.txt",
			env:     map[string]string{"BAR": "foo"}}}}},
		Task{sequence: 0})
	c.Check(err, IsNil)
	checkOutput(c, tmpdir)
}

func (s *TestSuite) TestEnvSubstitute(c *C) {
	tmpdir, _ := ioutil.TempDir("", "")
	defer func() {
		os.RemoveAll(tmpdir)
	}()

	err := runner(ArvTestClient{c, ". d3b07384d113edec49eaa6238ad5ff00+4 0:4:output.txt\n", true},
		KeepTestClient{},
		"zzzz-8i9sb-111111111111111",
		"zzzz-ot0gb-111111111111111",
		tmpdir,
		"foo\n",
		Job{script_parameters: Tasks{[]TaskDef{TaskDef{
			command: []string{"/bin/sh", "-c", "echo $BAR"},
			stdout:  "output.txt",
			env:     map[string]string{"BAR": "$(task.keep)"}}}}},
		Task{sequence: 0})
	c.Check(err, IsNil)
	checkOutput(c, tmpdir)
}

func (s *TestSuite) TestEnvReplace(c *C) {
	tmpdir, _ := ioutil.TempDir("", "")
	defer func() {
		os.RemoveAll(tmpdir)
	}()

	err := runner(ArvTestClient{c, ". d3b07384d113edec49eaa6238ad5ff00+4 0:4:output.txt\n", true},
		KeepTestClient{},
		"zzzz-8i9sb-111111111111111",
		"zzzz-ot0gb-111111111111111",
		tmpdir,
		"",
		Job{script_parameters: Tasks{[]TaskDef{TaskDef{
			command: []string{"/bin/sh", "-c", "echo $PATH"},
			stdout:  "output.txt",
			env:     map[string]string{"PATH": "foo"}}}}},
		Task{sequence: 0})
	c.Check(err, IsNil)
	checkOutput(c, tmpdir)
}

type SubtaskTestClient struct {
	c     *C
	parms []Task
	i     int
}

func (t *SubtaskTestClient) Create(resourceType string, parameters arvadosclient.Dict, output interface{}) error {
	t.c.Check(resourceType, Equals, "job_tasks")
	t.c.Check(parameters, DeepEquals, arvadosclient.Dict{"job_task": t.parms[t.i]})
	t.i += 1
	return nil
}

func (t SubtaskTestClient) Update(resourceType string, uuid string, parameters arvadosclient.Dict, output interface{}) (err error) {
	return nil
}

func (s *TestSuite) TestScheduleSubtask(c *C) {

	api := SubtaskTestClient{c, []Task{
		Task{job_uuid: "zzzz-8i9sb-111111111111111",
			created_by_job_task_uuid: "zzzz-ot0gb-111111111111111",
			sequence:                 1,
			parameters: TaskDef{
				command: []string{"echo", "bar"}}},
		Task{job_uuid: "zzzz-8i9sb-111111111111111",
			created_by_job_task_uuid: "zzzz-ot0gb-111111111111111",
			sequence:                 1,
			parameters: TaskDef{
				command: []string{"echo", "foo"}}}},
		0}

	tmpdir, _ := ioutil.TempDir("", "")
	defer func() {
		os.RemoveAll(tmpdir)
	}()

	err := runner(&api, KeepTestClient{},
		"zzzz-8i9sb-111111111111111",
		"zzzz-ot0gb-111111111111111",
		tmpdir,
		"",
		Job{script_parameters: Tasks{[]TaskDef{
			TaskDef{command: []string{"echo", "bar"}},
			TaskDef{command: []string{"echo", "foo"}}}}},
		Task{sequence: 0})
	c.Check(err, IsNil)

}

func (s *TestSuite) TestRunFail(c *C) {
	tmpdir, _ := ioutil.TempDir("", "")
	defer func() {
		os.RemoveAll(tmpdir)
	}()

	err := runner(ArvTestClient{c, "", false}, KeepTestClient{},
		"zzzz-8i9sb-111111111111111",
		"zzzz-ot0gb-111111111111111",
		tmpdir,
		"",
		Job{script_parameters: Tasks{[]TaskDef{TaskDef{
			command: []string{"/bin/sh", "-c", "exit 1"}}}}},
		Task{sequence: 0})
	c.Check(err, FitsTypeOf, PermFail{})
}

func (s *TestSuite) TestRunSuccessCode(c *C) {
	tmpdir, _ := ioutil.TempDir("", "")
	defer func() {
		os.RemoveAll(tmpdir)
	}()

	err := runner(ArvTestClient{c, "", true}, KeepTestClient{},
		"zzzz-8i9sb-111111111111111",
		"zzzz-ot0gb-111111111111111",
		tmpdir,
		"",
		Job{script_parameters: Tasks{[]TaskDef{TaskDef{
			command:      []string{"/bin/sh", "-c", "exit 1"},
			successCodes: []int{0, 1}}}}},
		Task{sequence: 0})
	c.Check(err, IsNil)
}

func (s *TestSuite) TestRunFailCode(c *C) {
	tmpdir, _ := ioutil.TempDir("", "")
	defer func() {
		os.RemoveAll(tmpdir)
	}()

	err := runner(ArvTestClient{c, "", false}, KeepTestClient{},
		"zzzz-8i9sb-111111111111111",
		"zzzz-ot0gb-111111111111111",
		tmpdir,
		"",
		Job{script_parameters: Tasks{[]TaskDef{TaskDef{
			command:            []string{"/bin/sh", "-c", "exit 0"},
			permanentFailCodes: []int{0, 1}}}}},
		Task{sequence: 0})
	c.Check(err, FitsTypeOf, PermFail{})
}

func (s *TestSuite) TestRunTempFailCode(c *C) {
	tmpdir, _ := ioutil.TempDir("", "")
	defer func() {
		os.RemoveAll(tmpdir)
	}()

	err := runner(ArvTestClient{c, "", false}, KeepTestClient{},
		"zzzz-8i9sb-111111111111111",
		"zzzz-ot0gb-111111111111111",
		tmpdir,
		"",
		Job{script_parameters: Tasks{[]TaskDef{TaskDef{
			command:            []string{"/bin/sh", "-c", "exit 1"},
			temporaryFailCodes: []int{1}}}}},
		Task{sequence: 0})
	c.Check(err, FitsTypeOf, TempFail{})
}

func (s *TestSuite) TestVwd(c *C) {
	tmpfile, _ := ioutil.TempFile("", "")
	tmpfile.Write([]byte("foo\n"))
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	tmpdir, _ := ioutil.TempDir("", "")
	defer func() {
		os.RemoveAll(tmpdir)
	}()

	err := runner(ArvTestClient{c, ". d3b07384d113edec49eaa6238ad5ff00+4 0:4:output.txt\n", true},
		KeepTestClient{},
		"zzzz-8i9sb-111111111111111",
		"zzzz-ot0gb-111111111111111",
		tmpdir,
		"",
		Job{script_parameters: Tasks{[]TaskDef{TaskDef{
			command: []string{"ls", "output.txt"},
			vwd: map[string]string{
				"output.txt": tmpfile.Name()}}}}},
		Task{sequence: 0})
	c.Check(err, IsNil)
	checkOutput(c, tmpdir)
}

func (s *TestSuite) TestSubstitutionStdin(c *C) {
	keepmount, _ := ioutil.TempDir("", "")
	ioutil.WriteFile(keepmount+"/"+"file1.txt", []byte("foo\n"), 0600)
	defer func() {
		os.RemoveAll(keepmount)
	}()

	log.Print("Keepmount is ", keepmount)

	tmpdir, _ := ioutil.TempDir("", "")
	defer func() {
		os.RemoveAll(tmpdir)
	}()

	log.Print("tmpdir is ", tmpdir)

	err := runner(ArvTestClient{c,
		". d3b07384d113edec49eaa6238ad5ff00+4 0:4:output.txt\n", true},
		KeepTestClient{},
		"zzzz-8i9sb-111111111111111",
		"zzzz-ot0gb-111111111111111",
		tmpdir,
		keepmount,
		Job{script_parameters: Tasks{[]TaskDef{TaskDef{
			command: []string{"cat"},
			stdout:  "output.txt",
			stdin:   "$(task.keep)/file1.txt"}}}},
		Task{sequence: 0})
	c.Check(err, IsNil)
	checkOutput(c, tmpdir)
}

func (s *TestSuite) TestSubstitutionCommandLine(c *C) {
	keepmount, _ := ioutil.TempDir("", "")
	ioutil.WriteFile(keepmount+"/"+"file1.txt", []byte("foo\n"), 0600)
	defer func() {
		os.RemoveAll(keepmount)
	}()

	tmpdir, _ := ioutil.TempDir("", "")
	defer func() {
		os.RemoveAll(tmpdir)
	}()

	err := runner(ArvTestClient{c,
		". d3b07384d113edec49eaa6238ad5ff00+4 0:4:output.txt\n", true},
		KeepTestClient{},
		"zzzz-8i9sb-111111111111111",
		"zzzz-ot0gb-111111111111111",
		tmpdir,
		keepmount,
		Job{script_parameters: Tasks{[]TaskDef{TaskDef{
			command: []string{"cat", "$(task.keep)/file1.txt"},
			stdout:  "output.txt"}}}},
		Task{sequence: 0})
	c.Check(err, IsNil)

	checkOutput(c, tmpdir)
}

func (s *TestSuite) TestSignal(c *C) {
	tmpdir, _ := ioutil.TempDir("", "")
	defer func() {
		os.RemoveAll(tmpdir)
	}()

	go func() {
		time.Sleep(1 * time.Second)
		self, _ := os.FindProcess(os.Getpid())
		self.Signal(syscall.SIGINT)
	}()

	err := runner(ArvTestClient{c,
		"", false},
		KeepTestClient{},
		"zzzz-8i9sb-111111111111111",
		"zzzz-ot0gb-111111111111111",
		tmpdir,
		"",
		Job{script_parameters: Tasks{[]TaskDef{TaskDef{
			command: []string{"sleep", "4"}}}}},
		Task{sequence: 0})
	c.Check(err, FitsTypeOf, PermFail{})

}

func (s *TestSuite) TestQuoting(c *C) {
	tmpdir, _ := ioutil.TempDir("", "")
	defer func() {
		os.RemoveAll(tmpdir)
	}()

	err := runner(ArvTestClient{c,
		"./s\\040ub:dir d3b07384d113edec49eaa6238ad5ff00+4 0:4::e\\040vil\n", true},
		KeepTestClient{},
		"zzzz-8i9sb-111111111111111",
		"zzzz-ot0gb-111111111111111",
		tmpdir,
		"",
		Job{script_parameters: Tasks{[]TaskDef{TaskDef{
			command: []string{"echo", "foo"},
			stdout:  "s ub:dir/:e vi\nl"}}}},
		Task{sequence: 0})
	c.Check(err, IsNil)
}
