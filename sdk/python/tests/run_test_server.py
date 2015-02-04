#!/usr/bin/env python

import argparse
import atexit
import httplib2
import os
import pipes
import random
import re
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

ARVADOS_DIR = os.path.realpath(os.path.join(MY_DIRNAME, '../../..'))
SERVICES_SRC_DIR = os.path.join(ARVADOS_DIR, 'services')
SERVER_PID_PATH = 'tmp/pids/test-server.pid'
if 'GOPATH' in os.environ:
    gopaths = os.environ['GOPATH'].split(':')
    gobins = [os.path.join(path, 'bin') for path in gopaths]
    os.environ['PATH'] = ':'.join(gobins) + ':' + os.environ['PATH']

TEST_TMPDIR = os.path.join(ARVADOS_DIR, 'tmp')
if not os.path.exists(TEST_TMPDIR):
    os.mkdir(TEST_TMPDIR)

my_api_host = None

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

def kill_server_pid(pidfile, wait=10, passenger=False):
    # Must re-import modules in order to work during atexit
    import os
    import signal
    import subprocess
    import time
    try:
        if passenger:
            # First try to shut down nicely
            restore_cwd = os.getcwd()
            os.chdir(os.path.join(SERVICES_SRC_DIR, 'api'))
            subprocess.call([
                'bundle', 'exec', 'passenger', 'stop', '--pid-file', pidfile])
            os.chdir(restore_cwd)
        now = time.time()
        timeout = now + wait
        with open(pidfile, 'r') as f:
            server_pid = int(f.read())
        while now <= timeout:
            if not passenger or timeout - now < wait / 2:
                # Half timeout has elapsed. Start sending SIGTERM
                os.kill(server_pid, signal.SIGTERM)
            # Raise OSError if process has disappeared
            os.getpgid(server_pid)
            time.sleep(0.1)
            now = time.time()
    except IOError:
        pass
    except OSError:
        pass

def run(leave_running_atexit=False):
    """Ensure an API server is running, and ARVADOS_API_* env vars have
    admin credentials for it.
    """
    global my_api_host

    # Delete cached discovery document.
    shutil.rmtree(arvados.http_cache('discovery'))

    os.environ['ARVADOS_API_TOKEN'] = auth_token('admin')
    os.environ['ARVADOS_API_HOST_INSECURE'] = 'true'

    pid_file = os.path.join(SERVICES_SRC_DIR, 'api', SERVER_PID_PATH)
    pid_file_ok = find_server_pid(pid_file, 0)

    if pid_file_ok:
        try:
            reset()
            return
        except:
            pass

    restore_cwd = os.getcwd()
    os.chdir(os.path.join(SERVICES_SRC_DIR, 'api'))

    # Either we haven't started a server of our own yet, or it has
    # died, or we have lost our credentials, or something else is
    # preventing us from calling reset(). Start a new one.

    if not os.path.exists('tmp/self-signed.pem'):
        # We assume here that either passenger reports its listening
        # address as https:/0.0.0.0:port/. If it reports "127.0.0.1"
        # then the certificate won't match the host and reset() will
        # fail certificate verification. If it reports "localhost",
        # clients (notably Python SDK's websocket client) might
        # resolve localhost as ::1 and then fail to connect.
        subprocess.check_call([
            'openssl', 'req', '-new', '-x509', '-nodes',
            '-out', 'tmp/self-signed.pem',
            '-keyout', 'tmp/self-signed.key',
            '-days', '3650',
            '-subj', '/CN=0.0.0.0'])

    port = random.randint(20000, 40000)
    env = os.environ.copy()
    env['RAILS_ENV'] = 'test'
    env['ARVADOS_WEBSOCKETS'] = 'yes'
    env.pop('ARVADOS_TEST_API_HOST', None)
    env.pop('ARVADOS_API_HOST', None)
    env.pop('ARVADOS_API_HOST_INSECURE', None)
    env.pop('ARVADOS_API_TOKEN', None)
    start_msg = subprocess.check_output(
        ['bundle', 'exec',
         'passenger', 'start', '-d', '-p{}'.format(port),
         '--pid-file', os.path.join(os.getcwd(), pid_file),
         '--log-file', os.path.join(os.getcwd(), 'log/test.log'),
         '--ssl',
         '--ssl-certificate', 'tmp/self-signed.pem',
         '--ssl-certificate-key', 'tmp/self-signed.key'],
        env=env)

    if not leave_running_atexit:
        atexit.register(kill_server_pid, pid_file, passenger=True)

    match = re.search(r'Accessible via: https://(.*?)/', start_msg)
    if not match:
        raise Exception(
            "Passenger did not report endpoint: {}".format(start_msg))
    my_api_host = match.group(1)
    os.environ['ARVADOS_API_HOST'] = my_api_host

    # Make sure the server has written its pid file before continuing
    find_server_pid(pid_file)

    reset()
    os.chdir(restore_cwd)

def reset():
    token = auth_token('admin')
    httpclient = httplib2.Http(ca_certs=os.path.join(
        SERVICES_SRC_DIR, 'api', 'tmp', 'self-signed.pem'))
    httpclient.request(
        'https://{}/database/reset'.format(os.environ['ARVADOS_API_HOST']),
        'POST',
        headers={'Authorization': 'OAuth2 {}'.format(token)})

def stop(force=False):
    """Stop the API server, if one is running. If force==True, kill it
    even if we didn't start it ourselves.
    """
    global my_api_host
    if force or my_api_host is not None:
        kill_server_pid(os.path.join(SERVICES_SRC_DIR, 'api', SERVER_PID_PATH))
        my_api_host = None

def _start_keep(n, keep_args):
    keep0 = tempfile.mkdtemp()
    port = random.randint(20000, 40000)
    keep_cmd = ["keepstore",
                "-volumes={}".format(keep0),
                "-listen=:{}".format(port),
                "-pid={}".format("{}/keep{}.pid".format(TEST_TMPDIR, n))]

    for arg, val in keep_args.iteritems():
        keep_cmd.append("{}={}".format(arg, val))

    kp0 = subprocess.Popen(keep_cmd)
    with open("{}/keep{}.pid".format(TEST_TMPDIR, n), 'w') as f:
        f.write(str(kp0.pid))

    with open("{}/keep{}.volume".format(TEST_TMPDIR, n), 'w') as f:
        f.write(keep0)

    return port

def run_keep(blob_signing_key=None, enforce_permissions=False):
    stop_keep()

    keep_args = {}
    if blob_signing_key:
        with open(os.path.join(TEST_TMPDIR, "keep.blob_signing_key"), "w") as f:
            keep_args['--permission-key-file'] = f.name
            f.write(blob_signing_key)
    if enforce_permissions:
        keep_args['--enforce-permissions'] = 'true'

    api = arvados.api(
        'v1', cache=False,
        host=os.environ['ARVADOS_API_HOST'],
        token=os.environ['ARVADOS_API_TOKEN'],
        insecure=True)
    for d in api.keep_services().list().execute()['items']:
        api.keep_services().delete(uuid=d['uuid']).execute()
    for d in api.keep_disks().list().execute()['items']:
        api.keep_disks().delete(uuid=d['uuid']).execute()

    for d in range(0, 2):
        port = _start_keep(d, keep_args)
        svc = api.keep_services().create(body={'keep_service': {
            'uuid': 'zzzzz-bi6l4-keepdisk{:07d}'.format(d),
            'service_host': 'localhost',
            'service_port': port,
            'service_type': 'disk',
            'service_ssl_flag': False,
        }}).execute()
        api.keep_disks().create(body={
            'keep_disk': {'keep_service_uuid': svc['uuid'] }
        }).execute()

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

def run_keep_proxy():
    stop_keep_proxy()

    admin_token = auth_token('admin')
    port = random.randint(20000,40000)
    env = os.environ.copy()
    env['ARVADOS_API_TOKEN'] = admin_token
    kp = subprocess.Popen(
        ['keepproxy',
         '-pid={}/keepproxy.pid'.format(TEST_TMPDIR),
         '-listen=:{}'.format(port)],
        env=env)

    api = arvados.api(
        'v1', cache=False,
        host=os.environ['ARVADOS_API_HOST'],
        token=admin_token,
        insecure=True)
    for d in api.keep_services().list(
            filters=[['service_type','=','proxy']]).execute()['items']:
        api.keep_services().delete(uuid=d['uuid']).execute()
    api.keep_services().create(body={'keep_service': {
        'service_host': 'localhost',
        'service_port': port,
        'service_type': 'proxy',
        'service_ssl_flag': False,
    }}).execute()
    os.environ["ARVADOS_KEEP_PROXY"] = "http://localhost:{}".format(port)

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

def auth_token(token_name):
    return fixture("api_client_authorizations")[token_name]["api_token"]

def authorize_with(token_name):
    '''token_name is the symbolic name of the token from the api_client_authorizations fixture'''
    arvados.config.settings()["ARVADOS_API_TOKEN"] = auth_token(token_name)
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
        os.environ.pop('ARVADOS_KEEP_PROXY', None)
        os.environ.pop('ARVADOS_EXTERNAL_CLIENT', None)
        for server_kwargs, start_func, stop_func in (
                (cls.MAIN_SERVER, run, reset),
                (cls.KEEP_SERVER, run_keep, stop_keep),
                (cls.KEEP_PROXY_SERVER, run_keep_proxy, stop_keep_proxy)):
            if server_kwargs is not None:
                start_func(**server_kwargs)
                cls._cleanup_funcs.append(stop_func)
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
    actions = ['start', 'stop',
               'start_keep', 'stop_keep',
               'start_keep_proxy', 'stop_keep_proxy']
    parser = argparse.ArgumentParser()
    parser.add_argument('action', type=str, help="one of {}".format(actions))
    parser.add_argument('--auth', type=str, metavar='FIXTURE_NAME', help='Print authorization info for given api_client_authorizations fixture')
    args = parser.parse_args()

    if args.action == 'start':
        stop(force=('ARVADOS_TEST_API_HOST' not in os.environ))
        run(leave_running_atexit=True)
        host = os.environ['ARVADOS_API_HOST']
        if args.auth is not None:
            token = auth_token(args.auth)
            print("export ARVADOS_API_TOKEN={}".format(pipes.quote(token)))
            print("export ARVADOS_API_HOST={}".format(pipes.quote(host)))
            print("export ARVADOS_API_HOST_INSECURE=true")
        else:
            print(host)
    elif args.action == 'stop':
        stop(force=('ARVADOS_TEST_API_HOST' not in os.environ))
    elif args.action == 'start_keep':
        run_keep()
    elif args.action == 'stop_keep':
        stop_keep()
    elif args.action == 'start_keep_proxy':
        run_keep_proxy()
    elif args.action == 'stop_keep_proxy':
        stop_keep_proxy()
    else:
        print("Unrecognized action '{}'. Actions are: {}.".format(args.action, actions))
