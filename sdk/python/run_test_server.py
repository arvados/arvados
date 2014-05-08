import subprocess
import time
import os
import signal
import yaml
import sys
import argparse
import arvados.config
import shutil
import tempfile

ARV_API_SERVER_DIR = '../../services/api'
KEEP_SERVER_DIR = '../../services/keep'
SERVER_PID_PATH = 'tmp/pids/server.pid'
WEBSOCKETS_SERVER_PID_PATH = 'tmp/pids/passenger.3001.pid'

def find_server_pid(PID_PATH, wait=10):
    now = time.time()
    timeout = now + wait
    good_pid = False
    while (not good_pid) and (now <= timeout):
        time.sleep(0.2)
        try:
            with open(PID_PATH, 'r') as f:
                server_pid = int(f.read())
            good_pid = (os.kill(server_pid, 0) == None)
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
        while now <= timeout:
            with open(PID_PATH, 'r') as f:
                server_pid = int(f.read())
            os.kill(server_pid, signal.SIGTERM) == None
            now = time.time()
    except IOError:
        good_pid = False
    except OSError:
        good_pid = False

def run(websockets=False, reuse_server=False):
    cwd = os.getcwd()
    os.chdir(os.path.join(os.path.dirname(__file__), ARV_API_SERVER_DIR))

    if websockets:
        pid_file = WEBSOCKETS_SERVER_PID_PATH
    else:
        pid_file = SERVER_PID_PATH

    test_pid = find_server_pid(pid_file, 0)

    if test_pid == None or not reuse_server:
        if test_pid != None:
            stop()

        # delete cached discovery document
        shutil.rmtree(os.path.join("~", ".cache", "arvados", "discovery"), True)

        # Setup database
        os.environ["RAILS_ENV"] = "test"
        subprocess.call(['bundle', 'exec', 'rake', 'db:test:load'])
        subprocess.call(['bundle', 'exec', 'rake', 'db:fixtures:load'])

        if websockets:
            os.environ["ARVADOS_WEBSOCKETS"] = "true"
            subprocess.call(['openssl', 'req', '-new', '-x509', '-nodes',
                             '-out', './self-signed.pem',
                             '-keyout', './self-signed.key',
                             '-days', '3650',
                             '-subj', '/CN=localhost'])
            subprocess.call(['passenger', 'start', '-d', '-p3001', '--ssl',
                             '--ssl-certificate', 'self-signed.pem',
                             '--ssl-certificate-key', 'self-signed.key'])
        else:
            subprocess.call(['bundle', 'exec', 'rails', 'server', '-d', '-p3001'])

        pid = find_server_pid(SERVER_PID_PATH)

    #os.environ["ARVADOS_API_HOST"] = "localhost:3001"
    os.environ["ARVADOS_API_HOST"] = "127.0.0.1:3001"
    os.environ["ARVADOS_API_HOST_INSECURE"] = "true"
    os.environ["ARVADOS_API_TOKEN"] = ""
    os.chdir(cwd)

def stop():
    cwd = os.getcwd()
    os.chdir(os.path.join(os.path.dirname(__file__), ARV_API_SERVER_DIR))

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

def _start_keep(n):
    keep0 = tempfile.mkdtemp()
    kp0 = subprocess.Popen(["bin/keep", "-volumes={}".format(keep0), "-listen=:{}".format(25107+n)])
    with open("tmp/keep{}.pid".format(n), 'w') as f:
        f.write(str(kp0.pid))
    with open("tmp/keep{}.volume".format(n), 'w') as f:
        f.write(keep0)

def run_keep():
    stop_keep()

    cwd = os.getcwd()
    os.chdir(os.path.join(os.path.dirname(__file__), KEEP_SERVER_DIR))
    os.environ["GOPATH"] = os.getcwd()
    subprocess.call(["go", "install", "keep"])

    if not os.path.exists("tmp"):
        os.mkdir("tmp")

    _start_keep(0)
    _start_keep(1)

    authorize_with("admin")
    api = arvados.api('v1')
    a = api.keep_disks().list().execute()
    for d in api.keep_disks().list().execute()['items']:
        api.keep_disks().delete(uuid=d['uuid']).execute()

    api.keep_disks().create(body={"keep_disk": {"service_host": "localhost",  "service_port": 25107} }).execute()
    api.keep_disks().create(body={"keep_disk": {"service_host": "localhost",  "service_port": 25108} }).execute()

    os.chdir(cwd)

def _stop_keep(n):
    kill_server_pid("tmp/keep{}.pid".format(n), 0)
    if os.path.exists("tmp/keep{}.volume".format(n)):
        with open("tmp/keep{}.volume".format(n), 'r') as r:
            shutil.rmtree(r.read(), True)

def stop_keep():
    cwd = os.getcwd()
    os.chdir(os.path.join(os.path.dirname(__file__), KEEP_SERVER_DIR))

    _stop_keep(0)
    _stop_keep(1)

    shutil.rmtree("tmp", True)

    os.chdir(cwd)

def fixture(fix):
    '''load a fixture yaml file'''
    with open(os.path.join(os.path.dirname(__file__), ARV_API_SERVER_DIR, "test", "fixtures",
                           fix + ".yml")) as f:
        return yaml.load(f.read())

def authorize_with(token):
    '''token is the symbolic name of the token from the api_client_authorizations fixture'''
    arvados.config.settings()["ARVADOS_API_TOKEN"] = fixture("api_client_authorizations")[token]["api_token"]

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument('action', type=str, help='''one of "start", "stop", "start_keep", "stop_keep"''')
    parser.add_argument('--websockets', action='store_true', default=False)
    parser.add_argument('--reuse', action='store_true', default=False)
    parser.add_argument('--auth', type=str, help='Print authorization info for given api_client_authorizations fixture')
    args = parser.parse_args()

    if args.action == 'start':
        run(args.websockets, args.reuse)
        if args.auth != None:
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
