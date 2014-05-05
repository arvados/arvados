import subprocess
import time
import os
import signal
import yaml
import sys
import argparse

ARV_API_SERVER_DIR = '../../services/api'
SERVER_PID_PATH = 'tmp/pids/server.pid'
WEBSOCKETS_SERVER_PID_PATH = 'tmp/pids/passenger.3001.pid'

def find_server_pid(PID_PATH):
    timeout = time.time() + 10
    good_pid = False
    while (not good_pid) and (time.time() < timeout):
        time.sleep(0.2)
        try:
            with open(PID_PATH, 'r') as f:
                server_pid = int(f.read())
            good_pid = (server_pid > 0) and (os.kill(server_pid, 0) == None)
        except:
            good_pid = False

    if not good_pid:
        raise Exception("could not find API server Rails pid")

    return server_pid

def run(websockets=False):
    cwd = os.getcwd()
    os.chdir(ARV_API_SERVER_DIR)
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
        find_server_pid(WEBSOCKETS_SERVER_PID_PATH)
    else:
        subprocess.call(['bundle', 'exec', 'rails', 'server', '-d', '-p3001'])
        find_server_pid(SERVER_PID_PATH)
    #os.environ["ARVADOS_API_HOST"] = "localhost:3001"
    os.environ["ARVADOS_API_HOST"] = "127.0.0.1:3001"
    os.environ["ARVADOS_API_HOST_INSECURE"] = "true"
    os.chdir(cwd)

def stop(websockets=False):
    cwd = os.getcwd()
    os.chdir(ARV_API_SERVER_DIR)
    if websockets:
        os.kill(find_server_pid(WEBSOCKETS_SERVER_PID_PATH), signal.SIGTERM)
        os.unlink('self-signed.pem')
        os.unlink('self-signed.key')
    else:
        os.kill(find_server_pid(SERVER_PID_PATH), signal.SIGTERM)
    os.chdir(cwd)

def fixture(fix):
    '''load a fixture yaml file'''
    with open(os.path.join(ARV_API_SERVER_DIR, "test", "fixtures",
                           fix + ".yml")) as f:
        return yaml.load(f.read())

def authorize_with(token):
    '''token is the symbolic name of the token from the api_client_authorizations fixture'''
    os.environ["ARVADOS_API_TOKEN"] = fixture("api_client_authorizations")[token]["api_token"]

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument('action', type=str, help='''one of "start" or "stop"''')
    parser.add_argument('--websockets', action='store_true', default=False)
    args = parser.parse_args()

    if args.action == 'start':
        run(args.websockets)
    elif args.action == 'stop':
        stop(args.websockets)
