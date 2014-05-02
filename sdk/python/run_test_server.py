import subprocess
import time
import os
import signal
import yaml

ARV_API_SERVER_DIR = '../../services/api'
SERVER_PID_PATH = 'tmp/pids/server.pid'

def find_server_pid():
    timeout = time.time() + 10
    good_pid = False
    while (not good_pid) and (time.time() < timeout):
        time.sleep(0.2)
        try:
            with open(SERVER_PID_PATH, 'r') as f:
                server_pid = int(f.read())
            good_pid = (server_pid > 0) and (os.kill(server_pid, 0) == None)
        except:
            good_pid = False

    if not good_pid:
        raise Exception("could not find API server Rails pid")

    os.environ["ARVADOS_API_HOST"] = "localhost:3001"
    os.environ["ARVADOS_API_HOST_INSECURE"] = "true"

    return server_pid

def run():
    cwd = os.getcwd()
    os.chdir(ARV_API_SERVER_DIR)
    os.environ["RAILS_ENV"] = "test"
    subprocess.call(['bundle', 'exec', 'rake', 'db:test:load'])
    subprocess.call(['bundle', 'exec', 'rake', 'db:fixtures:load'])
    subprocess.call(['bundle', 'exec', 'rails', 'server', '-d'])
    find_server_pid()
    os.chdir(cwd)

def stop():
    cwd = os.getcwd()
    os.chdir(ARV_API_SERVER_DIR)
    os.kill(find_server_pid(), signal.SIGTERM)
    os.chdir(cwd)

def fixture(fix):
    '''load a fixture yaml file'''
    with open(os.path.join(ARV_API_SERVER_DIR, "test", "fixtures",
                           fix + ".yml")) as f:
        return yaml.load(f.read())

def authorize_with(token):
    '''token is the symbolic name of the token from the api_client_authorizations fixture'''
    os.environ["ARVADOS_API_TOKEN"] = fixture("api_client_authorizations")[token]["api_token"]
