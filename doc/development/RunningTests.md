[comment]: # (Copyright © The Arvados Authors. All rights reserved.)
[comment]: # ()
[comment]: # (SPDX-License-Identifier: CC-BY-SA-3.0)

# Running tests

The arvados git repository has a `run-tests.sh` script which tests (nearly) all of the components in the source tree. Jenkins at https://ci.arvados.org uses this exact script, so running it *before pushing a new main* is a good way to predict whether Jenkins will fail your build and start pestering you by IRC.

## Background

You have [installed prerequisites](Prerequisites.md) following that guide. You have the arvados source tree at `~/arvados` and you might have local modifications.

## Environment

Your locale must use utf-8. Set environment variable LANG=C.UTF-8 if necessary to ensure the “locale” command reports UTF-8.

## If you’re Jenkins

Run all tests from a clean slate (slow, but more immune to leaks)

    WORKSPACE=~/arvados ~/arvados/build/run-tests.sh

## If you’re a developer

Cache everything needed to run test suites:

    mkdir ~/.cache/arvados-build
    WORKSPACE=~/arvados ~/arvados/build/run-tests.sh --temp ~/.cache/arvados-build --only install

Start interactive mode:

    $ WORKSPACE=~/arvados ~/arvados/build/run-tests.sh --temp ~/.cache/arvados-build --interactive

Start interactive mode and enabled debug output:

    $ WORKSPACE=~/arvados ~/arvados/build/run-tests.sh ARVADOS_DEBUG=1 --temp ~/.cache/arvados-build --interactive

When prompted, choose a test suite to run:

    == Interactive commands:
    test TARGET
    test TARGET:py3        (test with python3)
    test TARGET -check.vv  (pass arguments to test)
    install TARGET
    install env            (go/python libs)
    install deps           (go/python libs + arvados components needed for integration tests)
    reset                  (...services used by integration tests)
    exit
    == Test targets:
    cmd/arvados-client              lib/dispatchcloud/container     sdk/go/auth                     sdk/pam:py3                     services/fuse                   tools/crunchstat-summary
    cmd/arvados-server              lib/dispatchcloud/scheduler     sdk/go/blockdigest              sdk/python                      services/fuse:py3               tools/crunchstat-summary:py3
    lib/cli                         lib/dispatchcloud/ssh_executor  sdk/go/crunchrunner             sdk/python:py3                  services/health                 tools/keep-block-check
    lib/cloud                       lib/dispatchcloud/worker        sdk/go/dispatch                 services/arv-git-httpd          services/keep-balance           tools/keep-exercise
    lib/cloud/azure                 lib/service                     sdk/go/health                   services/crunch-dispatch-local  services/keepproxy              tools/keep-rsync
    lib/cloud/ec2                   sdk/cwl                         sdk/go/httpserver               services/crunch-dispatch-slurm  services/keepstore              tools/sync-groups
    lib/cmd                         sdk/cwl:py3                     sdk/go/keepclient               services/crunch-run             services/keep-web
    lib/controller                  sdk/go/arvados                  sdk/go/manifest                 services/crunchstat             services/nodemanager
    lib/crunchstat                  sdk/go/arvadosclient            sdk/go/stats                    services/dockercleaner          services/nodemanager:py3
    lib/dispatchcloud               sdk/go/asyncbuf                 sdk/pam                         services/dockercleaner:py3      services/ws
    What next? 

Example: testing lib/dispatchcloud/container, showing verbose/debug logs:

    What next? test lib/dispatchcloud/container/ -check.vv
    ======= test lib/dispatchcloud/container
    START: queue_test.go:99: IntegrationSuite.TestCancelIfNoInstanceType
    WARN[0000] cancel container with no suitable instance type  ContainerUUID=zzzzz-dz642-queuedcontainer error="no suitable instance type"
    WARN[0000] cancel container with no suitable instance type  ContainerUUID=zzzzz-dz642-queuedcontainer error="no suitable instance type"
    START: queue_test.go:37: IntegrationSuite.TearDownTest
    PASS: queue_test.go:37: IntegrationSuite.TearDownTest   0.846s

    PASS: queue_test.go:99: IntegrationSuite.TestCancelIfNoInstanceType     0.223s

    START: queue_test.go:42: IntegrationSuite.TestGetLockUnlockCancel
    INFO[0001] adding container to queue                     ContainerUUID=zzzzz-dz642-queuedcontainer InstanceType=testType Priority=1 State=Queued
    START: queue_test.go:37: IntegrationSuite.TearDownTest
    PASS: queue_test.go:37: IntegrationSuite.TearDownTest   0.901s

    PASS: queue_test.go:42: IntegrationSuite.TestGetLockUnlockCancel        0.177s

    OK: 2 passed
    PASS
    ok      git.arvados.org/arvados.git/lib/dispatchcloud/container       2.150s
    ======= test lib/dispatchcloud/container -- 3s
    Pass: lib/dispatchcloud/container tests (3s)
    All test suites passed.

### Running individual test cases

#### Golang

Most Go packages use gocheck. Use gocheck command line args like -check.f.

    What next? test lib/dispatchcloud/container -check.vv -check.f=LockUnlock
    ======= test lib/dispatchcloud/container
    START: queue_test.go:42: IntegrationSuite.TestGetLockUnlockCancel
    INFO[0000] adding container to queue                     ContainerUUID=zzzzz-dz642-queuedcontainer InstanceType=testType Priority=1 State=Queued
    START: queue_test.go:37: IntegrationSuite.TearDownTest
    PASS: queue_test.go:37: IntegrationSuite.TearDownTest   0.812s

    PASS: queue_test.go:42: IntegrationSuite.TestGetLockUnlockCancel        0.184s

    OK: 1 passed
    PASS
    ok      git.arvados.org/arvados.git/lib/dispatchcloud/container       1.000s
    ======= test lib/dispatchcloud/container -- 2s

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

    What next? test sdk/python:py3 --disable-warnings --tb=no --no-showlocals tests/test_keep_client.py::KeepDiskCacheTestCase
    ======= test sdk/python
    […pip output…]
    ========================================================== test session starts ==========================================================
    platform linux -- Python 3.8.19, pytest-8.2.0, pluggy-1.5.0
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
    What next? test sdk/python:py3 --disable-warnings --tb=no --no-showlocals --lf
    ======= test sdk/python
    […pip output…]
    ========================================================== test session starts ==========================================================
    platform linux -- Python 3.8.19, pytest-8.2.0, pluggy-1.5.0
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

    What next? test services/api TESTOPTS=--name=/.*signed.locators.*/
    [...]
    # Running:

    ....

    Finished in 1.080084s, 3.7034 runs/s, 461.0751 assertions/s.

### Restarting services for integration tests

If you have changed services/api code, and you want to check whether it breaks the lib/dispatchcloud/container integration tests:

    What next? reset                                # teardown the integration-testing environment
    What next? install services/api                 # (only needed if you've updated dependencies)
    What next? test lib/dispatchcloud/container     # bring up the integration-testing environment and run tests
    What next? test lib/dispatchcloud/container     # leave the integration-testing environment up and run tests

### Updating cache after pulling main

Always quit interactive mode and restart after modifying run-tests.sh (via git-pull, git-checkout, editing, etc).

When you start, run “install all” to get the latest gem/python dependencies, install updated versions of Arvados services used by integration tests, etc.

Then you can resume your cycle of “test lib/controller”, etc.

### Controlling test order (Rails)

Rails tests start off with a line like this

    Run options: -v -d --seed 57089

The seed value determines the order tests are run. To reproduce reproduce an order-dependent test failure, specify the same seed as a previous failed run:

    What next? test services/api TESTOPTS="-v -d --seed 57089"

### Other options

For more usage info, try:

    ~/arvados/build/run-tests.sh --help

## Running workbench diagnostics tests

You can run workbench diagnostics tests against any production server.

Update your workbench application.yml to add a “diagnostics” section with the login token and pipeline details. The below example configuration is to run the “qr1hi-p5p6p-ftcb0o61u4yd2zr” pipeline in “qr1hi” environment.

    diagnostics:
      secret_token: useanicelongrandomlygeneratedsecrettokenstring
      arvados_workbench_url: https://workbench.qr1hi.arvadosapi.com
      user_tokens:
        active: yourcurrenttokenintheenvironmenttowhichyouarepointing
      pipelines_to_test:
        pipeline_1:
          template_uuid: qr1hi-p5p6p-ftcb0o61u4yd2zr
          input_paths: []
          max_wait_seconds: 300

You can now run the “qr1hi” diagnostics tests using the following command:

      cd $ARVADOS_HOME
      RAILS_ENV=diagnostics bundle exec rake TEST=test/diagnostics/pipeline_test.rb

## Running workbench2 tests

React uses a lot of filesystem watchers (via inotify). The default number of watched files is relatively low at 8192. Increase that with:

    echo fs.inotify.max_user_watches=524288 \| sudo tee -a /etc/sysctl.conf && sudo sysctl -p

### Docker

The integration tests can be run on non-debian based systems using docker. The workbench2 subfolder includes a Makefile target that preinstalls the necessary dependencies in a docker container using Ansible.

With Docker and Ansible installed, run this command from within the `arvados/services/workbench2` directory:

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

#### Docker Troubleshooting

##### Missing X server or \$DISPLAY

Run:

    xhost +local:root

##### No version of Cypress is installed / other error starting Cypress

Recreate the home volume which re-installs Cypress and other persisted dependencies by running:

    make clean-docker-volume
    make workbench-docker-volume

### Debian Host System

These instructions assume a Debian 10 (buster) host system.

Install the Arvados test dependencies:

    echo "deb http://deb.debian.org/debian buster-backports main" > /etc/apt/sources.list.d/backports.list
    apt-get update
    apt-get install -y --no-install-recommends golang -t buster-backports
    apt-get install -y --no-install-recommends build-essential ca-certificates git libpam0g-dev

Install a few more dependencies for workbench2:

    apt-get update
    apt-get install -y --no-install-recommends gnupg2 sudo curl
    curl -sS https://dl.yarnpkg.com/debian/pubkey.gpg | sudo apt-key add -
    echo "deb https://dl.yarnpkg.com/debian/ stable main" | sudo tee /etc/apt/sources.list.d/yarn.list
    apt-get update
    apt-get install -y --no-install-recommends yarn libgbm-dev
    # we need the correct version of node to install cypress
    # use arvados-server install to get it (and all other dependencies)
    # All this will then not need to be repeated by ./tools/run-integration-tests.sh
    # so we are not doing double work.
    cd /usr/src/arvados
    go mod download
    cd cmd/arvados-server
    go install
    ~/go/bin/arvados-server install -type test
    cd <your-wb2-directory>
    yarn run cypress install

Make sure you have both the arvados and arvados-workbench2 source trees available, and then use the following commands (adjust path for the arvados source tree, if necessary) from your workbench2 source tree.

### Running Tests

Run the unit tests with:

    make unit-tests
    # or
    yarn test

Run the cypress integration tests with:

    ARVADOS_DIRECTORY=/path/to/arvados
    # (optional, defaults to WB path under ARVADOS_DIRECTORY)
    WORKSPACE=/path/to/arvados/services/workbench2
    ./tools/run-integration-tests.sh -i
