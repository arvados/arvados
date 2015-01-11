#!/usr/bin/env python

import argparse
import os
import shutil
import signal
import subprocess
import sys
import tempfile
import time
import unittest
import yaml

MY_DIRNAME = os.path.dirname(os.path.realpath(__file__))
if __name__ == '__main__' and os.path.exists(
      os.path.join(MY_DIRNAME, '..', 'arvados', '__init__.py')):
    # We're being launched to support another test suite.
    # Add the Python SDK source to the library path.
    sys.path.insert(1, os.path.dirname(MY_DIRNAME))

import arvados.api
import arvados.config

SERVICES_SRC_DIR = os.path.join(MY_DIRNAME, '../../../services')
SERVER_PID_PATH = 'tmp/pids/webrick-test.pid'
WEBSOCKETS_SERVER_PID_PATH = 'tmp/pids/passenger-test.pid'
if 'GOPATH' in os.environ:
    gopaths = os.environ['GOPATH'].split(':')
    gobins = [os.path.join(path, 'bin') for path in gopaths]
    os.environ['PATH'] = ':'.join(gobins) + ':' + os.environ['PATH']

if os.path.isdir('tests'):
    TEST_TMPDIR = 'tests/tmp'
else:
    TEST_TMPDIR = 'tmp'

def find_server_pid(PID_PATH, wait=10):
    now = time.time()
    timeout = now + wait
    good_pid = False
    while (not good_pid) and (now <= timeout):
        time.sleep(0.2)
        try:
            with open(PID_PATH, 'r') as f:
                server_pid = int(f.read())
            good_pid = (os.kill(server_pid, 0) is None)
        except IOError:
            good_pid = False
        except OSError:
            good_pid = False
        now = time.time()

    if not good_pid:
        return None

    return server_pid

def kill_server_pid(PID_PATH, wait=10):
    try:
        now = time.time()
        timeout = now + wait
        with open(PID_PATH, 'r') as f:
            server_pid = int(f.read())
        while now <= timeout:
            os.kill(server_pid, signal.SIGTERM)
            os.getpgid(server_pid) # throw OSError if no such pid
            now = time.time()
            time.sleep(0.1)
    except IOError:
        good_pid = False
    except OSError:
        good_pid = False

def run(websockets=False, reuse_server=False):
    cwd = os.getcwd()
    os.chdir(os.path.join(SERVICES_SRC_DIR, 'api'))

    if websockets:
        pid_file = WEBSOCKETS_SERVER_PID_PATH
    else:
        pid_file = SERVER_PID_PATH

    test_pid = find_server_pid(pid_file, 0)

    if test_pid is None or not reuse_server:
        # do not try to run both server variants at once
        stop()

        # delete cached discovery document
        shutil.rmtree(arvados.http_cache('discovery'))

        # Setup database
        os.environ["RAILS_ENV"] = "test"
        subprocess.call(['bundle', 'exec', 'rake', 'tmp:cache:clear'])
        subprocess.call(['bundle', 'exec', 'rake', 'db:test:load'])
        subprocess.call(['bundle', 'exec', 'rake', 'db:fixtures:load'])

        subprocess.call(['bundle', 'exec', 'rails', 'server', '-d',
                         '--pid',
                         os.path.join(os.getcwd(), SERVER_PID_PATH),
                         '-p3000'])
        os.environ["ARVADOS_API_HOST"] = "127.0.0.1:3000"

        if websockets:
            os.environ["ARVADOS_WEBSOCKETS"] = "ws-only"
            subprocess.call(['bundle', 'exec',
                             'passenger', 'start', '-d', '-p3333',
                             '--pid-file',
                             os.path.join(os.getcwd(), WEBSOCKETS_SERVER_PID_PATH)
                         ])

        pid = find_server_pid(SERVER_PID_PATH)

    os.environ["ARVADOS_API_HOST_INSECURE"] = "true"
    os.environ["ARVADOS_API_TOKEN"] = ""
    os.chdir(cwd)

def stop():
    cwd = os.getcwd()
    os.chdir(os.path.join(SERVICES_SRC_DIR, 'api'))

    kill_server_pid(WEBSOCKETS_SERVER_PID_PATH, 0)
    kill_server_pid(SERVER_PID_PATH, 0)

    try:
        os.unlink('self-signed.pem')
    except:
        pass

    try:
        os.unlink('self-signed.key')
    except:
        pass

    os.chdir(cwd)

def _start_keep(n, keep_args):
    keep0 = tempfile.mkdtemp()
    keep_cmd = ["keepstore",
                "-volumes={}".format(keep0),
                "-listen=:{}".format(25107+n),
                "-pid={}".format("{}/keep{}.pid".format(TEST_TMPDIR, n))]

    for arg, val in keep_args.iteritems():
        keep_cmd.append("{}={}".format(arg, val))

    kp0 = subprocess.Popen(keep_cmd)
    with open("{}/keep{}.pid".format(TEST_TMPDIR, n), 'w') as f:
        f.write(str(kp0.pid))

    with open("{}/keep{}.volume".format(TEST_TMPDIR, n), 'w') as f:
        f.write(keep0)

def run_keep(blob_signing_key=None, enforce_permissions=False):
    stop_keep()

    if not os.path.exists(TEST_TMPDIR):
        os.mkdir(TEST_TMPDIR)

    keep_args = {}
    if blob_signing_key:
        with open(os.path.join(TEST_TMPDIR, "keep.blob_signing_key"), "w") as f:
            keep_args['--permission-key-file'] = f.name
            f.write(blob_signing_key)
    if enforce_permissions:
        keep_args['--enforce-permissions'] = 'true'

    _start_keep(0, keep_args)
    _start_keep(1, keep_args)

    os.environ["ARVADOS_API_HOST"] = "127.0.0.1:3000"
    os.environ["ARVADOS_API_HOST_INSECURE"] = "true"

    authorize_with("admin")
    api = arvados.api('v1', cache=False)
    for d in api.keep_services().list().execute()['items']:
        api.keep_services().delete(uuid=d['uuid']).execute()
    for d in api.keep_disks().list().execute()['items']:
        api.keep_disks().delete(uuid=d['uuid']).execute()

    s1 = api.keep_services().create(body={"keep_service": {
                "uuid": "zzzzz-bi6l4-5bo5n1iekkjyz6b",
                "service_host": "localhost",
                "service_port": 25107,
                "service_type": "disk"
                }}).execute()
    s2 = api.keep_services().create(body={"keep_service": {
                "uuid": "zzzzz-bi6l4-2nz60e0ksj7vr3s",
                "service_host": "localhost",
                "service_port": 25108,
                "service_type": "disk"
                }}).execute()
    api.keep_disks().create(body={"keep_disk": {"keep_service_uuid": s1["uuid"] } }).execute()
    api.keep_disks().create(body={"keep_disk": {"keep_service_uuid": s2["uuid"] } }).execute()

def _stop_keep(n):
    kill_server_pid("{}/keep{}.pid".format(TEST_TMPDIR, n), 0)
    if os.path.exists("{}/keep{}.volume".format(TEST_TMPDIR, n)):
        with open("{}/keep{}.volume".format(TEST_TMPDIR, n), 'r') as r:
            shutil.rmtree(r.read(), True)
        os.unlink("{}/keep{}.volume".format(TEST_TMPDIR, n))
    if os.path.exists(os.path.join(TEST_TMPDIR, "keep.blob_signing_key")):
        os.remove(os.path.join(TEST_TMPDIR, "keep.blob_signing_key"))

def stop_keep():
    _stop_keep(0)
    _stop_keep(1)

def run_keep_proxy(auth):
    stop_keep_proxy()

    if not os.path.exists(TEST_TMPDIR):
        os.mkdir(TEST_TMPDIR)

    os.environ["ARVADOS_API_HOST"] = "127.0.0.1:3000"
    os.environ["ARVADOS_API_HOST_INSECURE"] = "true"
    os.environ["ARVADOS_API_TOKEN"] = fixture("api_client_authorizations")[auth]["api_token"]

    kp0 = subprocess.Popen(["keepproxy",
                            "-pid={}/keepproxy.pid".format(TEST_TMPDIR),
                            "-listen=:{}".format(25101)])

    authorize_with("admin")
    api = arvados.api('v1', cache=False)
    api.keep_services().create(body={"keep_service": {"service_host": "localhost",  "service_port": 25101, "service_type": "proxy"} }).execute()

    os.environ["ARVADOS_KEEP_PROXY"] = "http://localhost:25101"

def stop_keep_proxy():
    kill_server_pid(os.path.join(TEST_TMPDIR, "keepproxy.pid"), 0)

def fixture(fix):
    '''load a fixture yaml file'''
    with open(os.path.join(SERVICES_SRC_DIR, 'api', "test", "fixtures",
                           fix + ".yml")) as f:
        yaml_file = f.read()
        try:
          trim_index = yaml_file.index("# Test Helper trims the rest of the file")
          yaml_file = yaml_file[0:trim_index]
        except ValueError:
          pass
        return yaml.load(yaml_file)

def authorize_with(token):
    '''token is the symbolic name of the token from the api_client_authorizations fixture'''
    arvados.config.settings()["ARVADOS_API_TOKEN"] = fixture("api_client_authorizations")[token]["api_token"]
    arvados.config.settings()["ARVADOS_API_HOST"] = os.environ.get("ARVADOS_API_HOST")
    arvados.config.settings()["ARVADOS_API_HOST_INSECURE"] = "true"

class TestCaseWithServers(unittest.TestCase):
    """TestCase to start and stop supporting Arvados servers.

    Define any of MAIN_SERVER, KEEP_SERVER, and/or KEEP_PROXY_SERVER
    class variables as a dictionary of keyword arguments.  If you do,
    setUpClass will start the corresponding servers by passing these
    keyword arguments to the run, run_keep, and/or run_keep_server
    functions, respectively.  It will also set Arvados environment
    variables to point to these servers appropriately.  If you don't
    run a Keep or Keep proxy server, setUpClass will set up a
    temporary directory for Keep local storage, and set it as
    KEEP_LOCAL_STORE.

    tearDownClass will stop any servers started, and restore the
    original environment.
    """
    MAIN_SERVER = None
    KEEP_SERVER = None
    KEEP_PROXY_SERVER = None

    @staticmethod
    def _restore_dict(src, dest):
        for key in dest.keys():
            if key not in src:
                del dest[key]
        dest.update(src)

    @classmethod
    def setUpClass(cls):
        cls._orig_environ = os.environ.copy()
        cls._orig_config = arvados.config.settings().copy()
        cls._cleanup_funcs = []
        for server_kwargs, start_func, stop_func in (
              (cls.MAIN_SERVER, run, stop),
              (cls.KEEP_SERVER, run_keep, stop_keep),
              (cls.KEEP_PROXY_SERVER, run_keep_proxy, stop_keep_proxy)):
            if server_kwargs is not None:
                start_func(**server_kwargs)
                cls._cleanup_funcs.append(stop_func)
        os.environ.pop('ARVADOS_EXTERNAL_CLIENT', None)
        if cls.KEEP_PROXY_SERVER is None:
            os.environ.pop('ARVADOS_KEEP_PROXY', None)
        if (cls.KEEP_SERVER is None) and (cls.KEEP_PROXY_SERVER is None):
            cls.local_store = tempfile.mkdtemp()
            os.environ['KEEP_LOCAL_STORE'] = cls.local_store
            cls._cleanup_funcs.append(
                lambda: shutil.rmtree(cls.local_store, ignore_errors=True))
        else:
            os.environ.pop('KEEP_LOCAL_STORE', None)
        arvados.config.initialize()

    @classmethod
    def tearDownClass(cls):
        for clean_func in cls._cleanup_funcs:
            clean_func()
        cls._restore_dict(cls._orig_environ, os.environ)
        cls._restore_dict(cls._orig_config, arvados.config.settings())


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument('action', type=str, help='''one of "start", "stop", "start_keep", "stop_keep"''')
    parser.add_argument('--websockets', action='store_true', default=False)
    parser.add_argument('--reuse', action='store_true', default=False)
    parser.add_argument('--auth', type=str, help='Print authorization info for given api_client_authorizations fixture')
    args = parser.parse_args()

    if args.action == 'start':
        run(websockets=args.websockets, reuse_server=args.reuse)
        if args.auth is not None:
            authorize_with(args.auth)
            print("export ARVADOS_API_HOST={}".format(arvados.config.settings()["ARVADOS_API_HOST"]))
            print("export ARVADOS_API_TOKEN={}".format(arvados.config.settings()["ARVADOS_API_TOKEN"]))
            print("export ARVADOS_API_HOST_INSECURE={}".format(arvados.config.settings()["ARVADOS_API_HOST_INSECURE"]))
    elif args.action == 'stop':
        stop()
    elif args.action == 'start_keep':
        run_keep()
    elif args.action == 'stop_keep':
        stop_keep()
    elif args.action == 'start_keep_proxy':
        run_keep_proxy("admin")
    elif args.action == 'stop_keep_proxy':
        stop_keep_proxy()
    else:
        print('Unrecognized action "{}", actions are "start", "stop", "start_keep", "stop_keep"'.format(args.action))
