package main

import (
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	. "gopkg.in/check.v1"
	"io/ioutil"
	"os"
	"testing"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	TestingT(t)
}

type TestSuite struct{}

// Gocheck boilerplate
var _ = Suite(&TestSuite{})

type ArvTestClient struct {
}

func (t ArvTestClient) Create(resourceType string, parameters arvadosclient.Dict, output interface{}) error {
	return nil
}

func (t ArvTestClient) Delete(resource string, uuid string, parameters arvadosclient.Dict, output interface{}) (err error) {
	return nil
}

func (t ArvTestClient) Update(resourceType string, uuid string, parameters arvadosclient.Dict, output interface{}) (err error) {
	return nil
}

func (t ArvTestClient) Get(resourceType string, uuid string, parameters arvadosclient.Dict, output interface{}) (err error) {
	return nil
}

func (t ArvTestClient) List(resource string, parameters arvadosclient.Dict, output interface{}) (err error) {
	return nil
}

func (s *TestSuite) TestSimpleRun(c *C) {

	api := ArvTestClient{}

	tmpdir, _ := ioutil.TempDir("", "")
	defer func() {
		os.RemoveAll(tmpdir)
	}()

	err := runner(api,
		"zzzz-8i9sb-111111111111111",
		"zzzz-ot0gb-111111111111111",
		tmpdir,
		"",
		Job{script_parameters: Tasks{[]TaskDef{TaskDef{
			commands: []string{"echo", "foo"}}}}},
		Task{sequence: 0})
	c.Check(err, IsNil)

}

func checkOutput(c *C, tmpdir string) {
	file, err := os.Open(tmpdir + "/zzzz-ot0gb-111111111111111/output.txt")
	c.Assert(err, IsNil)

	data := make([]byte, 100)
	var count int
	count, err = file.Read(data)
	c.Assert(err, IsNil)
	c.Check(string(data[0:count]), Equals, "foo\n")
}

func (s *TestSuite) TestSimpleRunSubtask(c *C) {

	api := ArvTestClient{}

	tmpdir, _ := ioutil.TempDir("", "")
	defer func() {
		os.RemoveAll(tmpdir)
	}()

	err := runner(api,
		"zzzz-8i9sb-111111111111111",
		"zzzz-ot0gb-111111111111111",
		tmpdir,
		"",
		Job{script_parameters: Tasks{[]TaskDef{
			TaskDef{commands: []string{"echo", "bar"}},
			TaskDef{commands: []string{"echo", "foo"}}}}},
		Task{parameters: TaskDef{
			commands: []string{"echo", "foo"},
			stdout:   "output.txt"},
			sequence: 1})
	c.Check(err, IsNil)

	checkOutput(c, tmpdir)
}

func (s *TestSuite) TestRedirect(c *C) {

	api := ArvTestClient{}

	tmpfile, _ := ioutil.TempFile("", "")
	tmpfile.Write([]byte("foo\n"))
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	tmpdir, _ := ioutil.TempDir("", "")
	defer func() {
		os.RemoveAll(tmpdir)
	}()

	err := runner(api,
		"zzzz-8i9sb-111111111111111",
		"zzzz-ot0gb-111111111111111",
		tmpdir,
		"",
		Job{script_parameters: Tasks{[]TaskDef{TaskDef{
			commands: []string{"cat"},
			stdout:   "output.txt",
			stdin:    tmpfile.Name()}}}},
		Task{sequence: 0})
	c.Check(err, IsNil)

	checkOutput(c, tmpdir)
}

func (s *TestSuite) TestEnv(c *C) {

	api := ArvTestClient{}

	tmpdir, _ := ioutil.TempDir("", "")
	defer func() {
		os.RemoveAll(tmpdir)
	}()

	err := runner(api,
		"zzzz-8i9sb-111111111111111",
		"zzzz-ot0gb-111111111111111",
		tmpdir,
		"",
		Job{script_parameters: Tasks{[]TaskDef{TaskDef{
			commands: []string{"/bin/sh", "-c", "echo $BAR"},
			stdout:   "output.txt",
			env:      map[string]string{"BAR": "foo"}}}}},
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

func (t SubtaskTestClient) Delete(resource string, uuid string, parameters arvadosclient.Dict, output interface{}) (err error) {
	return nil
}

func (t SubtaskTestClient) Update(resourceType string, uuid string, parameters arvadosclient.Dict, output interface{}) (err error) {
	return nil
}

func (t SubtaskTestClient) Get(resourceType string, uuid string, parameters arvadosclient.Dict, output interface{}) (err error) {
	return nil
}

func (t SubtaskTestClient) List(resource string, parameters arvadosclient.Dict, output interface{}) (err error) {
	return nil
}

func (s *TestSuite) TestScheduleSubtask(c *C) {

	api := SubtaskTestClient{c, []Task{
		Task{job_uuid: "zzzz-8i9sb-111111111111111",
			created_by_job_task_uuid: "zzzz-ot0gb-111111111111111",
			sequence:                 1,
			parameters: TaskDef{
				commands: []string{"echo", "bar"}}},
		Task{job_uuid: "zzzz-8i9sb-111111111111111",
			created_by_job_task_uuid: "zzzz-ot0gb-111111111111111",
			sequence:                 1,
			parameters: TaskDef{
				commands: []string{"echo", "foo"}}}},
		0}

	tmpdir, _ := ioutil.TempDir("", "")
	defer func() {
		os.RemoveAll(tmpdir)
	}()

	err := runner(&api,
		"zzzz-8i9sb-111111111111111",
		"zzzz-ot0gb-111111111111111",
		tmpdir,
		"",
		Job{script_parameters: Tasks{[]TaskDef{
			TaskDef{commands: []string{"echo", "bar"}},
			TaskDef{commands: []string{"echo", "foo"}}}}},
		Task{sequence: 0})
	c.Check(err, IsNil)

}

func (s *TestSuite) TestRunFail(c *C) {

	api := ArvTestClient{}

	tmpdir, _ := ioutil.TempDir("", "")
	defer func() {
		os.RemoveAll(tmpdir)
	}()

	err := runner(api,
		"zzzz-8i9sb-111111111111111",
		"zzzz-ot0gb-111111111111111",
		tmpdir,
		"",
		Job{script_parameters: Tasks{[]TaskDef{TaskDef{
			commands: []string{"/bin/sh", "-c", "exit 1"}}}}},
		Task{sequence: 0})
	c.Check(err, FitsTypeOf, PermFail{})
}

func (s *TestSuite) TestRunSuccessCode(c *C) {

	api := ArvTestClient{}

	tmpdir, _ := ioutil.TempDir("", "")
	defer func() {
		os.RemoveAll(tmpdir)
	}()

	err := runner(api,
		"zzzz-8i9sb-111111111111111",
		"zzzz-ot0gb-111111111111111",
		tmpdir,
		"",
		Job{script_parameters: Tasks{[]TaskDef{TaskDef{
			commands:     []string{"/bin/sh", "-c", "exit 1"},
			successCodes: []int{0, 1}}}}},
		Task{sequence: 0})
	c.Check(err, IsNil)
}

func (s *TestSuite) TestRunFailCode(c *C) {
	api := ArvTestClient{}

	tmpdir, _ := ioutil.TempDir("", "")
	defer func() {
		os.RemoveAll(tmpdir)
	}()

	err := runner(api,
		"zzzz-8i9sb-111111111111111",
		"zzzz-ot0gb-111111111111111",
		tmpdir,
		"",
		Job{script_parameters: Tasks{[]TaskDef{TaskDef{
			commands:           []string{"/bin/sh", "-c", "exit 0"},
			permanentFailCodes: []int{0, 1}}}}},
		Task{sequence: 0})
	c.Check(err, FitsTypeOf, PermFail{})
}

func (s *TestSuite) TestRunTempFailCode(c *C) {
	api := ArvTestClient{}

	tmpdir, _ := ioutil.TempDir("", "")
	defer func() {
		os.RemoveAll(tmpdir)
	}()

	err := runner(api,
		"zzzz-8i9sb-111111111111111",
		"zzzz-ot0gb-111111111111111",
		tmpdir,
		"",
		Job{script_parameters: Tasks{[]TaskDef{TaskDef{
			commands:           []string{"/bin/sh", "-c", "exit 1"},
			temporaryFailCodes: []int{1}}}}},
		Task{sequence: 0})
	c.Check(err, FitsTypeOf, TempFail{})
}

func (s *TestSuite) TestVwd(c *C) {
	api := ArvTestClient{}

	tmpfile, _ := ioutil.TempFile("", "")
	tmpfile.Write([]byte("foo\n"))
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	tmpdir, _ := ioutil.TempDir("", "")
	defer func() {
		os.RemoveAll(tmpdir)
	}()

	err := runner(api,
		"zzzz-8i9sb-111111111111111",
		"zzzz-ot0gb-111111111111111",
		tmpdir,
		"",
		Job{script_parameters: Tasks{[]TaskDef{TaskDef{
			commands: []string{"ls", "output.txt"},
			vwd: map[string]string{
				"output.txt": tmpfile.Name()}}}}},
		Task{sequence: 0})
	c.Check(err, IsNil)
	checkOutput(c, tmpdir)
}
