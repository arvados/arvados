[comment]: # (Copyright © The Arvados Authors. All rights reserved.)
[comment]: # ()
[comment]: # (SPDX-License-Identifier: CC-BY-SA-3.0)

# Running Tests

Arvados includes a script at `build/run-tests.sh` which tests (nearly) all of the components in the source tree. This is the script that [Arvados CI tests](https://ci.arvados.org) use, so running it locally is the most consistent entry point to all Arvados tests.

This document assumes you have [installed a development environment](Prerequisites.md) following that guide.

## Running interactively

Most developers want to run tests with `--temp` and `--interactive`:

```sh
$ mkdir -p tmp/run-tests
$ build/run-tests.sh --temp "$PWD/tmp/run-tests" --interactive
```

This will display help with a list of commands and test targets. When you run with a fresh temp directory, the tool will probably prompt you to `install deps`. You should do this to install dependencies to the temp directory.

### Dealing with state

Before you change `run-tests.sh` itself—including pulling changes from other developers—you should end any interactive sessions.

If you make changes to a low-library library or SDK and want to see how it affects dependent tests, `install` your changed component, then `test` the dependents.

If you make changes to a cluster component and want to see how it affects tests, `reset` the test cluster, then `test` the components you're interested in.

If you want to clean your `--temp` directory—because you pulled a bad dependency or just want to recover some disk space—it is safe to end any interactive sessions, remove it, then `mkdir` it again.

### Running individual test cases

#### Golang

Most Go packages use gocheck. Use gocheck command line args like `-check.f` to select tests and `-check.v` to show more output.

    What next? test lib/controller/router -check.f=RouterSuite -check.v
    ======= test lib/controller/router
    PASS: request_test.go:135: RouterSuite.TestAttrsInBody	0.000s
    PASS: request_test.go:164: RouterSuite.TestBoolParam	0.000s
    PASS: router_test.go:55: RouterSuite.TestOptions	0.002s
    PASS: request_test.go:209: RouterSuite.TestStringOrArrayParam	0.000s
    OK: 4 passed
    PASS
    ok  	git.arvados.org/arvados.git/lib/controller/router	0.012s
    ======= test lib/controller/router -- 1s

#### Python

If what you really want to do is focus on failing or newly-added tests, consider passing the appropriate switches to do that:

      -x, --exitfirst       Exit instantly on first error or failed test
      --lf, --last-failed   Rerun only the tests that failed at the last run (or
                            all if none failed)
      --ff, --failed-first  Run all tests, but run the last failures first. This
                            may re-order tests and thus lead to repeated fixture
                            setup/teardown.
      --nf, --new-first     Run tests from new files first, then the rest of the
                            tests sorted by file mtime

If you want to manually select tests:

      FILENAME              Run tests from FILENAME, relative to the source root
      FILENAME::CLASSNAME   Run tests from CLASSNAME
      FILENAME::FUNCNAME, FILENAME::CLASSNAME::FUNCNAME
                            Run only the named test function
      -k EXPRESSION         Only run tests which match the given substring
                            expression. An expression is a Python evaluable
                            expression where all names are substring-matched
                            against test names and their parent classes.
                            Example: -k 'test_method or test_other' matches all
                            test functions and classes whose name contains
                            'test_method' or 'test_other', while -k 'not
                            test_method' matches those that don't contain
                            'test_method' in their names. -k 'not test_method
                            and not test_other' will eliminate the matches.
                            Additionally keywords are matched to classes and
                            functions containing extra names in their
                            'extra_keyword_matches' set, as well as functions
                            which have names assigned directly to them. The
                            matching is case-insensitive.
      -m MARKEXPR           Only run tests matching given mark expression. For
                            example: -m 'mark1 and not mark2'.

For even more options, refer to the [pytest command line reference](https://docs.pytest.org/en/stable/reference/reference.html#command-line-flags).

Example:

    What next? test sdk/python --disable-warnings --tb=no --no-showlocals tests/test_keep_client.py::KeepDiskCacheTestCase
    ======= test sdk/python
    […pip output…]
    ========================================================== test session starts ==========================================================
    platform linux -- Python 3.10.19, pytest-9.0.2, pluggy-1.6.0
    rootdir: /home/brett/Curii/arvados/sdk/python
    configfile: pytest.ini
    collected 9 items

    tests/test_keep_client.py F........                                                                                               [100%]

    ======================================================== short test summary info ========================================================
    FAILED tests/test_keep_client.py::KeepDiskCacheTestCase::test_disk_cache_cap - AssertionError: True is not false
    ====================================================== 1 failed, 8 passed in 0.16s ======================================================
    ======= sdk/python tests -- FAILED
    ======= test sdk/python -- 2s
    Failures (1):
    Fail: sdk/python tests (2s)
    What next? test sdk/python --disable-warnings --tb=no --no-showlocals --lf
    ======= test sdk/python
    […pip output…]
    ========================================================== test session starts ==========================================================
    platform linux -- Python 3.10.19, pytest-9.0.2, pluggy-1.6.0
    rootdir: /home/brett/Curii/arvados/sdk/python
    configfile: pytest.ini
    testpaths: tests
    collected 964 items / 963 deselected / 1 selected
    run-last-failure: rerun previous 1 failure

    tests/test_keep_client.py F                                                                                                       [100%]

    ======================================================== short test summary info ========================================================
    FAILED tests/test_keep_client.py::KeepDiskCacheTestCase::test_disk_cache_cap - AssertionError: True is not false
    ============================================= 1 failed, 963 deselected, 1 warning in 0.43s ==============================================
    ======= sdk/python tests -- FAILED
    ======= test sdk/python -- 2s
    Failures (1):
    Fail: sdk/python tests (2s)

#### RailsAPI

Rails parses `TESTOPTS` and passes them to the test runner:

    What next? test services/api TESTOPTS=--name=/.*signed.locators.*/
    [...]
    # Running:

    ....

    Finished in 1.080084s, 3.7034 runs/s, 461.0751 assertions/s.

##### Controlling Rails test order

Rails tests start off with a line like this

    Run options: -v -d --seed 57089

The seed value determines the order tests are run. To reproduce reproduce an order-dependent test failure, specify the same seed as a previous failed run:

    What next? test services/api TESTOPTS="-v -d --seed 57089"

## Environment variables

The following variables affect test setup and execution:

Variable    | Value
------------|-----------------------------------------------------------------
`CONFIGSRC` | A directory with an Arvados cluster `config.yml`. Tests will read `Clusters.zzzzz.PostgreSQL.Connection` from that file to determine how to connect to the test database. If not set, the tests will use default connection settings.
`WORKSPACE` | A directory with an Arvados Git checkout. Defaults to what Git reports for `run-tests.sh` itself.

`run-tests.sh` cleans `ARVADOS` variables from the environment to help ensure consistent test execution. These variables are helpful, but you must pass them as arguments to `run-tests.sh` and not in the environment directly.

Variable               | Value
-----------------------|------------------------------------------------------
`ARVADOS_DEBUG`        | If 1, lots of components will log more information.
`ARVADOS_TEST_PRIVESC` | A literal string. If `sudo`, various tests that need to perform privileged operations with run with `sudo` to get them. Otherwise, those tests are skipped.

## Scripting run-tests

If you run `run-tests.sh` without `--interactive`, by default it runs all the tests and reports their results. This is how CI runs. Run the script with `--help` to see the options you can use to control this behavior. Common options include:

Option           | Behavior
-----------------|------------------------------------------------------------
`--only`         | Run a single set of tests
`--skip`         | Skip a set of tests during a full run
`NAME_test=ARGS` | Pass arguments to a set of tests

## Running Workbench tests in Docker

If you do not have a full development environment, Workbench tests can be run in Docker. The `services/workbench2` subfolder includes Makefile targets that preinstall the necessary dependencies in a Docker container using Ansible.

With Docker and Ansible installed (see `arvados/tools/ansible/README.md`), run this command from within the `arvados/services/workbench2` directory:

    make workbench-docker-image

You can verify the docker image was built by looking for `arvados/workbench` in `docker image ls`

Then, start the interactive tests with this command:

    make interactive-tests-in-docker

Non-interactive (headless) tests can be run with the targets:

    # Both e2e & component tests
    make tests-in-docker

    # Integration (e2e) only
    make integration-tests-in-docker

    # Unit (component) only
    make unit-tests-in-docker

### Troubleshooting

#### Missing X server or \$DISPLAY

Run:

    xhost +local:root

#### No version of Cypress is installed / other error starting Cypress

Recreate the home volume which re-installs Cypress and other persisted dependencies by running:

    make clean-docker-volume
    make workbench-docker-volume
