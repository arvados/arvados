#! /usr/bin/env python

# traffic_test.py
#
# Launch a test Keep and API server and PUT and GET a bunch of blocks.
# Can be used to simulate client traffic in Keep to evaluate memory usage,
# error logging, performance, etc.
#
# This script is warty and is relatively environment-specific, but the
# example run described below should execute cleanly.
#
# Usage:
#   traffic_test.py start
#       Starts the test servers.
#   traffic_test.py put file1 file2 file3 ....
#       Runs arv-put on each file.
#   traffic_test.py get hash1 hash2 hash3 ....
#       Loops forever issuing GET requests for specified blocks.
#   traffic_test.py stop
#       Stops the test servers.
#
# Example:
#
#   $ ./traffic_test.py start
#   $ ./traffic_test.py put GS00253-DNA_A02_200_37.tsv.bz2 \
#         GS00253-DNA_B01_200_37.tsv.bz2 \
#         GS00253-DNA_B02_200_37.tsv.bz2
#   $ ./traffic_test.py get $(find /tmp/tmp* -type f -printf "%f ")
#     [loops forever]
#     ^C
#   $ ./traffic_test.py stop
#
# Multiple "get" runs may be run concurrently to evaluate Keep's handling
# of additional concurrent clients.

PYSDK_DIR    = "../../../sdk/python"
PYTEST_DIR   = PYSDK_DIR + "/tests"
ARV_PUT_PATH = PYSDK_DIR + "/bin/arv-put"
ARV_GET_PATH = PYSDK_DIR + "/bin/arv-get"
SECONDS_BETWEEN_GETS = 1

import argparse
import httplib2
import os
import random
import subprocess
import sys
import time

# for run_test_server.py
sys.path.insert(0, PYSDK_DIR)
sys.path.insert(0, PYTEST_DIR)
import arvados
import run_test_server

def arv_cmd(*args):
    p = subprocess.Popen([sys.executable] + list(args),
                         stdout=subprocess.PIPE)
    (arvout, arverr) = p.communicate()
    if p.returncode != 0:
        print "error {} from {} {}: {}".format(
            p.returncode, sys.executable, args, arverr)
        sys.exit(p.returncode)
    return arvout

def start():
    run_test_server.run()
    run_test_server.run_keep()

def put(files):
    os.environ["ARVADOS_API_HOST"] = "127.0.0.1:3000"
    run_test_server.authorize_with('active')
    for v in ["ARVADOS_API_HOST",
              "ARVADOS_API_HOST_INSECURE",
              "ARVADOS_API_TOKEN"]:
        os.environ[v] = arvados.config.settings()[v]

    if not os.environ.has_key('PYTHONPATH'):
        os.environ['PYTHONPATH'] = ''
    os.environ['PYTHONPATH'] = "{}:{}:{}".format(
        PYSDK_DIR, PYTEST_DIR, os.environ['PYTHONPATH'])

    for c in files:
        manifest_uuid = arv_cmd(ARV_PUT_PATH, c)

def get(blocks):
    os.environ["ARVADOS_API_HOST"] = "127.0.0.1:3000"

    run_test_server.authorize_with('active')
    for v in ["ARVADOS_API_HOST",
              "ARVADOS_API_HOST_INSECURE",
              "ARVADOS_API_TOKEN"]:
        os.environ[v] = arvados.config.settings()[v]

    nqueries = 0
    while True:
        b = random.choice(blocks)
        print "GET /" + b
        body = arv_cmd(ARV_GET_PATH, b)
        print "got {} bytes".format(len(body))
        time.sleep(SECONDS_BETWEEN_GETS)
        nqueries = nqueries + 1

def stop():
    run_test_server.stop_keep()
    run_test_server.stop()

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument('action',
                        type=str,
                        nargs='+',
                        help='''"start", "put", "get", "stop"''')
    args = parser.parse_args()

    if args.action[0] == 'start':
        start()
    elif args.action[0] == 'put':
        put(args.action[1:])
    elif args.action[0] == 'get':
        get(args.action[1:])
    elif args.action[0] == 'stop':
        stop()
    else:
        print('Unrecognized action "{}"'.format(args.action))
        print('actions are "start", "put", "get", "stop"')
