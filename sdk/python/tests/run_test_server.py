# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from __future__ import print_function
from __future__ import division
from builtins import str
from builtins import range
import argparse
import atexit
import errno
import glob
import httplib2
import os
import pipes
import random
import re
import shutil
import signal
import socket
import string
import subprocess
import sys
import tempfile
import time
import unittest
import yaml

try:
    from urllib.parse import urlparse
except ImportError:
    from urlparse import urlparse

MY_DIRNAME = os.path.dirname(os.path.realpath(__file__))
if __name__ == '__main__' and os.path.exists(
      os.path.join(MY_DIRNAME, '..', 'arvados', '__init__.py')):
    # We're being launched to support another test suite.
    # Add the Python SDK source to the library path.
    sys.path.insert(1, os.path.dirname(MY_DIRNAME))

import arvados
import arvados.config

ARVADOS_DIR = os.path.realpath(os.path.join(MY_DIRNAME, '../../..'))
SERVICES_SRC_DIR = os.path.join(ARVADOS_DIR, 'services')

# Work around https://bugs.python.org/issue27805, should be no longer
# necessary from sometime in Python 3.8.x
if not os.environ.get('ARVADOS_DEBUG', ''):
    WRITE_MODE = 'a'
else:
    WRITE_MODE = 'w'

if 'GOPATH' in os.environ:
    # Add all GOPATH bin dirs to PATH -- but insert them after the
    # ruby gems bin dir, to ensure "bundle" runs the Ruby bundler
    # command, not the golang.org/x/tools/cmd/bundle command.
    gopaths = os.environ['GOPATH'].split(':')
    addbins = [os.path.join(path, 'bin') for path in gopaths]
    newbins = []
    for path in os.environ['PATH'].split(':'):
        newbins.append(path)
        if os.path.exists(os.path.join(path, 'bundle')):
            newbins += addbins
            addbins = []
    newbins += addbins
    os.environ['PATH'] = ':'.join(newbins)

TEST_TMPDIR = os.path.join(ARVADOS_DIR, 'tmp')
if not os.path.exists(TEST_TMPDIR):
    os.mkdir(TEST_TMPDIR)

my_api_host = None
_cached_config = {}
_cached_db_config = {}
_already_used_port = {}

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
        except EnvironmentError:
            good_pid = False
        now = time.time()

    if not good_pid:
        return None

    return server_pid

def kill_server_pid(pidfile, wait=10, passenger_root=False):
    # Must re-import modules in order to work during atexit
    import os
    import signal
    import subprocess
    import time

    now = time.time()
    startTERM = now
    deadline = now + wait

    if passenger_root:
        # First try to shut down nicely
        restore_cwd = os.getcwd()
        os.chdir(passenger_root)
        subprocess.call([
            'bundle', 'exec', 'passenger', 'stop', '--pid-file', pidfile])
        os.chdir(restore_cwd)
        # Use up to half of the +wait+ period waiting for "passenger
        # stop" to work. If the process hasn't exited by then, start
        # sending TERM signals.
        startTERM += wait//2

    server_pid = None
    while now <= deadline and server_pid is None:
        try:
            with open(pidfile, 'r') as f:
                server_pid = int(f.read())
        except IOError:
            # No pidfile = nothing to kill.
            return
        except ValueError as error:
            # Pidfile exists, but we can't parse it. Perhaps the
            # server has created the file but hasn't written its PID
            # yet?
            print("Parse error reading pidfile {}: {}".format(pidfile, error),
                  file=sys.stderr)
            time.sleep(0.1)
            now = time.time()

    while now <= deadline:
        try:
            exited, _ = os.waitpid(server_pid, os.WNOHANG)
            if exited > 0:
                _remove_pidfile(pidfile)
                return
        except OSError:
            # already exited, or isn't our child process
            pass
        try:
            if now >= startTERM:
                os.kill(server_pid, signal.SIGTERM)
                print("Sent SIGTERM to {} ({})".format(server_pid, pidfile),
                      file=sys.stderr)
        except OSError as error:
            if error.errno == errno.ESRCH:
                # Thrown by os.getpgid() or os.kill() if the process
                # does not exist, i.e., our work here is done.
                _remove_pidfile(pidfile)
                return
            raise
        time.sleep(0.1)
        now = time.time()

    print("Server PID {} ({}) did not exit, giving up after {}s".
          format(server_pid, pidfile, wait),
          file=sys.stderr)

def _remove_pidfile(pidfile):
    try:
        os.unlink(pidfile)
    except:
        if os.path.lexists(pidfile):
            raise

def find_available_port():
    """Return an IPv4 port number that is not in use right now.

    We assume whoever needs to use the returned port is able to reuse
    a recently used port without waiting for TIME_WAIT (see
    SO_REUSEADDR / SO_REUSEPORT).

    Some opportunity for races here, but it's better than choosing
    something at random and not checking at all. If all of our servers
    (hey Passenger) knew that listening on port 0 was a thing, the OS
    would take care of the races, and this wouldn't be needed at all.
    """

    global _already_used_port
    while True:
        sock = socket.socket()
        sock.bind(('0.0.0.0', 0))
        port = sock.getsockname()[1]
        sock.close()
        if port not in _already_used_port:
            _already_used_port[port] = True
            return port

def _wait_until_port_listens(port, timeout=10, warn=True):
    """Wait for a process to start listening on the given port.

    If nothing listens on the port within the specified timeout (given
    in seconds), print a warning on stderr before returning.
    """
    try:
        subprocess.check_output(['which', 'netstat'])
    except subprocess.CalledProcessError:
        print("WARNING: No `netstat` -- cannot wait for port to listen. "+
              "Sleeping 0.5 and hoping for the best.",
              file=sys.stderr)
        time.sleep(0.5)
        return
    deadline = time.time() + timeout
    while time.time() < deadline:
        if re.search(r'\ntcp.*:'+str(port)+' .* LISTEN *\n', subprocess.check_output(['netstat', '-Wln']).decode()):
            return True
        time.sleep(0.1)
    if warn:
        print(
            "WARNING: Nothing is listening on port {} (waited {} seconds).".
            format(port, timeout),
            file=sys.stderr)
    return False

def _logfilename(label):
    """Set up a labelled log file, and return a path to write logs to.

    Normally, the returned path is {tmpdir}/{label}.log.

    In debug mode, logs are also written to stderr, with [label]
    prepended to each line. The returned path is a FIFO.

    +label+ should contain only alphanumerics: it is also used as part
    of the FIFO filename.

    """
    logfilename = os.path.join(TEST_TMPDIR, label+'.log')
    if not os.environ.get('ARVADOS_DEBUG', ''):
        return logfilename
    fifo = os.path.join(TEST_TMPDIR, label+'.fifo')
    try:
        os.remove(fifo)
    except OSError as error:
        if error.errno != errno.ENOENT:
            raise
    os.mkfifo(fifo, 0o700)
    stdbuf = ['stdbuf', '-i0', '-oL', '-eL']
    # open(fifo, 'r') would block waiting for someone to open the fifo
    # for writing, so we need a separate cat process to open it for
    # us.
    cat = subprocess.Popen(
        stdbuf+['cat', fifo],
        stdin=open('/dev/null'),
        stdout=subprocess.PIPE)
    tee = subprocess.Popen(
        stdbuf+['tee', '-a', logfilename],
        stdin=cat.stdout,
        stdout=subprocess.PIPE)
    subprocess.Popen(
        stdbuf+['sed', '-e', 's/^/['+label+'] /'],
        stdin=tee.stdout,
        stdout=sys.stderr)
    return fifo

def run(leave_running_atexit=False):
    """Ensure an API server is running, and ARVADOS_API_* env vars have
    admin credentials for it.

    If ARVADOS_TEST_API_HOST is set, a parent process has started a
    test server for us to use: we just need to reset() it using the
    admin token fixture.

    If a previous call to run() started a new server process, and it
    is still running, we just need to reset() it to fixture state and
    return.

    If neither of those options work out, we'll really start a new
    server.
    """
    global my_api_host

    # Delete cached discovery documents.
    #
    # This will clear cached docs that belong to other processes (like
    # concurrent test suites) even if they're still running. They should
    # be able to tolerate that.
    for fn in glob.glob(os.path.join(
            str(arvados.http_cache('discovery')),
            '*,arvados,v1,rest,*')):
        os.unlink(fn)

    pid_file = _pidfile('api')
    pid_file_ok = find_server_pid(pid_file, 0)

    existing_api_host = os.environ.get('ARVADOS_TEST_API_HOST', my_api_host)
    if existing_api_host and pid_file_ok:
        if existing_api_host == my_api_host:
            try:
                return reset()
            except:
                # Fall through to shutdown-and-start case.
                pass
        else:
            # Server was provided by parent. Can't recover if it's
            # unresettable.
            return reset()

    # Before trying to start up our own server, call stop() to avoid
    # "Phusion Passenger Standalone is already running on PID 12345".
    # (If we've gotten this far, ARVADOS_TEST_API_HOST isn't set, so
    # we know the server is ours to kill.)
    stop(force=True)

    restore_cwd = os.getcwd()
    api_src_dir = os.path.join(SERVICES_SRC_DIR, 'api')
    os.chdir(api_src_dir)

    # Either we haven't started a server of our own yet, or it has
    # died, or we have lost our credentials, or something else is
    # preventing us from calling reset(). Start a new one.

    if not os.path.exists('tmp'):
        os.makedirs('tmp')

    if not os.path.exists('tmp/api'):
        os.makedirs('tmp/api')

    if not os.path.exists('tmp/logs'):
        os.makedirs('tmp/logs')

    # Install the git repository fixtures.
    gitdir = os.path.join(SERVICES_SRC_DIR, 'api', 'tmp', 'git')
    gittarball = os.path.join(SERVICES_SRC_DIR, 'api', 'test', 'test.git.tar')
    if not os.path.isdir(gitdir):
        os.makedirs(gitdir)
    subprocess.check_output(['tar', '-xC', gitdir, '-f', gittarball])

    # Customizing the passenger config template is the only documented
    # way to override the default passenger_stat_throttle_rate (10 s).
    # In the testing environment, we want restart.txt to take effect
    # immediately.
    resdir = subprocess.check_output(['bundle', 'exec', 'passenger-config', 'about', 'resourcesdir']).decode().rstrip()
    with open(resdir + '/templates/standalone/config.erb') as f:
        template = f.read()
    newtemplate = re.sub('http {', 'http {\n        passenger_stat_throttle_rate 0;', template)
    if newtemplate == template:
        raise "template edit failed"
    with open('tmp/passenger-nginx.conf.erb', 'w') as f:
        f.write(newtemplate)

    port = internal_port_from_config("RailsAPI")
    env = os.environ.copy()
    env['RAILS_ENV'] = 'test'
    env['ARVADOS_RAILS_LOG_TO_STDOUT'] = '1'
    env.pop('ARVADOS_WEBSOCKETS', None)
    env.pop('ARVADOS_TEST_API_HOST', None)
    env.pop('ARVADOS_API_HOST', None)
    env.pop('ARVADOS_API_HOST_INSECURE', None)
    env.pop('ARVADOS_API_TOKEN', None)
    logf = open(_logfilename('railsapi'), WRITE_MODE)
    railsapi = subprocess.Popen(
        ['bundle', 'exec',
         'passenger', 'start', '-p{}'.format(port),
         '--nginx-config-template', 'tmp/passenger-nginx.conf.erb',
	 '--no-friendly-error-pages',
	 '--disable-anonymous-telemetry',
	 '--disable-security-update-check',
         '--pid-file', pid_file,
         '--log-file', '/dev/stdout',
         '--ssl',
         '--ssl-certificate', 'tmp/self-signed.pem',
         '--ssl-certificate-key', 'tmp/self-signed.key'],
        env=env, stdin=open('/dev/null'), stdout=logf, stderr=logf)

    if not leave_running_atexit:
        atexit.register(kill_server_pid, pid_file, passenger_root=api_src_dir)

    my_api_host = "127.0.0.1:"+str(port)
    os.environ['ARVADOS_API_HOST'] = my_api_host

    # Make sure the server has written its pid file and started
    # listening on its TCP port
    _wait_until_port_listens(port)
    find_server_pid(pid_file)

    reset()
    os.chdir(restore_cwd)

def reset():
    """Reset the test server to fixture state.

    This resets the ARVADOS_TEST_API_HOST provided by a parent process
    if any, otherwise the server started by run().

    It also resets ARVADOS_* environment vars to point to the test
    server with admin credentials.
    """
    existing_api_host = os.environ.get('ARVADOS_TEST_API_HOST', my_api_host)
    token = auth_token('admin')
    httpclient = httplib2.Http(ca_certs=os.path.join(
        SERVICES_SRC_DIR, 'api', 'tmp', 'self-signed.pem'))
    httpclient.request(
        'https://{}/database/reset'.format(existing_api_host),
        'POST',
        headers={'Authorization': 'OAuth2 {}'.format(token), 'Connection':'close'})

    os.environ['ARVADOS_API_HOST_INSECURE'] = 'true'
    os.environ['ARVADOS_API_TOKEN'] = token
    os.environ['ARVADOS_API_HOST'] = existing_api_host

def stop(force=False):
    """Stop the API server, if one is running.

    If force==False, kill it only if we started it ourselves. (This
    supports the use case where a Python test suite calls run(), but
    run() just uses the ARVADOS_TEST_API_HOST provided by the parent
    process, and the test suite cleans up after itself by calling
    stop(). In this case the test server provided by the parent
    process should be left alone.)

    If force==True, kill it even if we didn't start it
    ourselves. (This supports the use case in __main__, where "run"
    and "stop" happen in different processes.)
    """
    global my_api_host
    if force or my_api_host is not None:
        kill_server_pid(_pidfile('api'))
        my_api_host = None

def get_config():
    with open(os.environ["ARVADOS_CONFIG"]) as f:
        return yaml.safe_load(f)

def internal_port_from_config(service, idx=0):
    return int(urlparse(
        sorted(list(get_config()["Clusters"]["zzzzz"]["Services"][service]["InternalURLs"].keys()))[idx]).
               netloc.split(":")[1])

def external_port_from_config(service):
    return int(urlparse(get_config()["Clusters"]["zzzzz"]["Services"][service]["ExternalURL"]).netloc.split(":")[1])

def run_controller():
    if 'ARVADOS_TEST_PROXY_SERVICES' in os.environ:
        return
    stop_controller()
    logf = open(_logfilename('controller'), WRITE_MODE)
    port = internal_port_from_config("Controller")
    controller = subprocess.Popen(
        ["arvados-server", "controller"],
        stdin=open('/dev/null'), stdout=logf, stderr=logf, close_fds=True)
    with open(_pidfile('controller'), 'w') as f:
        f.write(str(controller.pid))
    _wait_until_port_listens(port)
    return port

def stop_controller():
    if 'ARVADOS_TEST_PROXY_SERVICES' in os.environ:
        return
    kill_server_pid(_pidfile('controller'))

def run_ws():
    if 'ARVADOS_TEST_PROXY_SERVICES' in os.environ:
        return
    stop_ws()
    port = internal_port_from_config("Websocket")
    logf = open(_logfilename('ws'), WRITE_MODE)
    ws = subprocess.Popen(
        ["arvados-server", "ws"],
        stdin=open('/dev/null'), stdout=logf, stderr=logf, close_fds=True)
    with open(_pidfile('ws'), 'w') as f:
        f.write(str(ws.pid))
    _wait_until_port_listens(port)
    return port

def stop_ws():
    if 'ARVADOS_TEST_PROXY_SERVICES' in os.environ:
        return
    kill_server_pid(_pidfile('ws'))

def _start_keep(n, blob_signing=False):
    datadir = os.path.join(TEST_TMPDIR, "keep%d.data"%n)
    if os.path.exists(datadir):
        shutil.rmtree(datadir)
    os.mkdir(datadir)
    port = internal_port_from_config("Keepstore", idx=n)

    # Currently, if there are multiple InternalURLs for a single host,
    # the only way to tell a keepstore process which one it's supposed
    # to listen on is to supply a redacted version of the config, with
    # the other InternalURLs removed.
    conf = os.path.join(TEST_TMPDIR, "keep%d.yaml"%n)
    confdata = get_config()
    confdata['Clusters']['zzzzz']['Services']['Keepstore']['InternalURLs'] = {"http://127.0.0.1:%d"%port: {}}
    confdata['Clusters']['zzzzz']['Collections']['BlobSigning'] = blob_signing
    with open(conf, 'w') as f:
        yaml.safe_dump(confdata, f)
    keep_cmd = ["arvados-server", "keepstore", "-config", conf]

    with open(_logfilename('keep{}'.format(n)), WRITE_MODE) as logf:
        with open('/dev/null') as _stdin:
            child = subprocess.Popen(
                keep_cmd, stdin=_stdin, stdout=logf, stderr=logf, close_fds=True)

    print('child.pid is %d'%child.pid, file=sys.stderr)
    with open(_pidfile('keep{}'.format(n)), 'w') as f:
        f.write(str(child.pid))

    _wait_until_port_listens(port)

    return port

def run_keep(num_servers=2, **kwargs):
    stop_keep(num_servers)

    api = arvados.api(
        version='v1',
        host=os.environ['ARVADOS_API_HOST'],
        token=os.environ['ARVADOS_API_TOKEN'],
        insecure=True)

    for d in api.keep_services().list(filters=[['service_type','=','disk']]).execute()['items']:
        api.keep_services().delete(uuid=d['uuid']).execute()
    for d in api.keep_disks().list().execute()['items']:
        api.keep_disks().delete(uuid=d['uuid']).execute()

    for d in range(0, num_servers):
        port = _start_keep(d, **kwargs)
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

    # If keepproxy and/or keep-web is running, send SIGHUP to make
    # them discover the new keepstore services.
    for svc in ('keepproxy', 'keep-web'):
        pidfile = _pidfile(svc)
        if os.path.exists(pidfile):
            try:
                with open(pidfile) as pid:
                    os.kill(int(pid.read()), signal.SIGHUP)
            except OSError:
                os.remove(pidfile)

def _stop_keep(n):
    kill_server_pid(_pidfile('keep{}'.format(n)))

def stop_keep(num_servers=2):
    for n in range(0, num_servers):
        _stop_keep(n)

def run_keep_proxy():
    if 'ARVADOS_TEST_PROXY_SERVICES' in os.environ:
        os.environ["ARVADOS_KEEP_SERVICES"] = "http://localhost:{}".format(internal_port_from_config('Keepproxy'))
        return
    stop_keep_proxy()

    port = internal_port_from_config("Keepproxy")
    env = os.environ.copy()
    env['ARVADOS_API_TOKEN'] = auth_token('anonymous')
    logf = open(_logfilename('keepproxy'), WRITE_MODE)
    kp = subprocess.Popen(
        ['arvados-server', 'keepproxy'], env=env, stdin=open('/dev/null'), stdout=logf, stderr=logf, close_fds=True)

    with open(_pidfile('keepproxy'), 'w') as f:
        f.write(str(kp.pid))
    _wait_until_port_listens(port)

    print("Using API %s token %s" % (os.environ['ARVADOS_API_HOST'], auth_token('admin')), file=sys.stdout)
    api = arvados.api(
        version='v1',
        host=os.environ['ARVADOS_API_HOST'],
        token=auth_token('admin'),
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
    os.environ["ARVADOS_KEEP_SERVICES"] = "http://localhost:{}".format(port)
    _wait_until_port_listens(port)

def stop_keep_proxy():
    if 'ARVADOS_TEST_PROXY_SERVICES' in os.environ:
        return
    kill_server_pid(_pidfile('keepproxy'))

def run_arv_git_httpd():
    if 'ARVADOS_TEST_PROXY_SERVICES' in os.environ:
        return
    stop_arv_git_httpd()

    gitport = internal_port_from_config("GitHTTP")
    env = os.environ.copy()
    env.pop('ARVADOS_API_TOKEN', None)
    logf = open(_logfilename('githttpd'), WRITE_MODE)
    agh = subprocess.Popen(['arvados-server', 'git-httpd'],
        env=env, stdin=open('/dev/null'), stdout=logf, stderr=logf)
    with open(_pidfile('githttpd'), 'w') as f:
        f.write(str(agh.pid))
    _wait_until_port_listens(gitport)

def stop_arv_git_httpd():
    if 'ARVADOS_TEST_PROXY_SERVICES' in os.environ:
        return
    kill_server_pid(_pidfile('githttpd'))

def run_keep_web():
    if 'ARVADOS_TEST_PROXY_SERVICES' in os.environ:
        return
    stop_keep_web()

    keepwebport = internal_port_from_config("WebDAV")
    env = os.environ.copy()
    logf = open(_logfilename('keep-web'), WRITE_MODE)
    keepweb = subprocess.Popen(
        ['arvados-server', 'keep-web'],
        env=env, stdin=open('/dev/null'), stdout=logf, stderr=logf)
    with open(_pidfile('keep-web'), 'w') as f:
        f.write(str(keepweb.pid))
    _wait_until_port_listens(keepwebport)

def stop_keep_web():
    if 'ARVADOS_TEST_PROXY_SERVICES' in os.environ:
        return
    kill_server_pid(_pidfile('keep-web'))

def run_nginx():
    if 'ARVADOS_TEST_PROXY_SERVICES' in os.environ:
        return
    stop_nginx()
    nginxconf = {}
    nginxconf['UPSTREAMHOST'] = '127.0.0.1'
    nginxconf['LISTENHOST'] = '127.0.0.1'
    nginxconf['CONTROLLERPORT'] = internal_port_from_config("Controller")
    nginxconf['ARVADOS_API_HOST'] = "0.0.0.0:" + str(external_port_from_config("Controller"))
    nginxconf['CONTROLLERSSLPORT'] = external_port_from_config("Controller")
    nginxconf['KEEPWEBPORT'] = internal_port_from_config("WebDAV")
    nginxconf['KEEPWEBDLSSLPORT'] = external_port_from_config("WebDAVDownload")
    nginxconf['KEEPWEBSSLPORT'] = external_port_from_config("WebDAV")
    nginxconf['KEEPPROXYPORT'] = internal_port_from_config("Keepproxy")
    nginxconf['KEEPPROXYSSLPORT'] = external_port_from_config("Keepproxy")
    nginxconf['GITPORT'] = internal_port_from_config("GitHTTP")
    nginxconf['GITSSLPORT'] = external_port_from_config("GitHTTP")
    nginxconf['HEALTHPORT'] = internal_port_from_config("Health")
    nginxconf['HEALTHSSLPORT'] = external_port_from_config("Health")
    nginxconf['WSPORT'] = internal_port_from_config("Websocket")
    nginxconf['WSSSLPORT'] = external_port_from_config("Websocket")
    nginxconf['WORKBENCH1PORT'] = internal_port_from_config("Workbench1")
    nginxconf['WORKBENCH1SSLPORT'] = external_port_from_config("Workbench1")
    nginxconf['WORKBENCH2PORT'] = internal_port_from_config("Workbench2")
    nginxconf['WORKBENCH2SSLPORT'] = external_port_from_config("Workbench2")
    nginxconf['SSLCERT'] = os.path.join(SERVICES_SRC_DIR, 'api', 'tmp', 'self-signed.pem')
    nginxconf['SSLKEY'] = os.path.join(SERVICES_SRC_DIR, 'api', 'tmp', 'self-signed.key')
    nginxconf['ACCESSLOG'] = _logfilename('nginx_access')
    nginxconf['ERRORLOG'] = _logfilename('nginx_error')
    nginxconf['TMPDIR'] = TEST_TMPDIR + '/nginx'
    nginxconf['INTERNALSUBNETS'] = '169.254.0.0/16 0;'

    conftemplatefile = os.path.join(MY_DIRNAME, 'nginx.conf')
    conffile = os.path.join(TEST_TMPDIR, 'nginx.conf')
    with open(conffile, 'w') as f:
        f.write(re.sub(
            r'{{([A-Z]+[A-Z0-9]+)}}',
            lambda match: str(nginxconf.get(match.group(1))),
            open(conftemplatefile).read()))

    env = os.environ.copy()
    env['PATH'] = env['PATH']+':/sbin:/usr/sbin:/usr/local/sbin'

    nginx = subprocess.Popen(
        ['nginx',
         '-g', 'error_log stderr info; pid '+_pidfile('nginx')+';',
         '-c', conffile],
        env=env, stdin=open('/dev/null'), stdout=sys.stderr)
    _wait_until_port_listens(nginxconf['CONTROLLERSSLPORT'])

def setup_config():
    rails_api_port = find_available_port()
    controller_port = find_available_port()
    controller_external_port = find_available_port()
    websocket_port = find_available_port()
    websocket_external_port = find_available_port()
    workbench1_port = find_available_port()
    workbench1_external_port = find_available_port()
    workbench2_port = find_available_port()
    workbench2_external_port = find_available_port()
    git_httpd_port = find_available_port()
    git_httpd_external_port = find_available_port()
    health_httpd_port = find_available_port()
    health_httpd_external_port = find_available_port()
    keepproxy_port = find_available_port()
    keepproxy_external_port = find_available_port()
    keepstore_ports = sorted([str(find_available_port()) for _ in range(0,4)])
    keep_web_port = find_available_port()
    keep_web_external_port = find_available_port()
    keep_web_dl_external_port = find_available_port()

    configsrc = os.environ.get("CONFIGSRC", None)
    if configsrc:
        clusterconf = os.path.join(configsrc, "config.yml")
        print("Getting config from %s" % clusterconf, file=sys.stderr)
        pgconnection = yaml.safe_load(open(clusterconf))["Clusters"]["zzzzz"]["PostgreSQL"]["Connection"]
    else:
        # assume "arvados-server install -type test" has set up the
        # conventional db credentials
        pgconnection = {
	    "client_encoding": "utf8",
	    "host": "localhost",
	    "dbname": "arvados_test",
	    "user": "arvados",
	    "password": "insecure_arvados_test",
        }

    localhost = "127.0.0.1"
    services = {
        "RailsAPI": {
            "InternalURLs": {
                "https://%s:%s"%(localhost, rails_api_port): {},
            },
        },
        "Controller": {
            "ExternalURL": "https://%s:%s" % (localhost, controller_external_port),
            "InternalURLs": {
                "http://%s:%s"%(localhost, controller_port): {},
            },
        },
        "Websocket": {
            "ExternalURL": "wss://%s:%s/websocket" % (localhost, websocket_external_port),
            "InternalURLs": {
                "http://%s:%s"%(localhost, websocket_port): {},
            },
        },
        "Workbench1": {
            "ExternalURL": "https://%s:%s/" % (localhost, workbench1_external_port),
            "InternalURLs": {
                "http://%s:%s"%(localhost, workbench1_port): {},
            },
        },
        "Workbench2": {
            "ExternalURL": "https://%s:%s/" % (localhost, workbench2_external_port),
            "InternalURLs": {
                "http://%s:%s"%(localhost, workbench2_port): {},
            },
        },
        "GitHTTP": {
            "ExternalURL": "https://%s:%s" % (localhost, git_httpd_external_port),
            "InternalURLs": {
                "http://%s:%s"%(localhost, git_httpd_port): {}
            },
        },
        "Health": {
            "ExternalURL": "https://%s:%s" % (localhost, health_httpd_external_port),
            "InternalURLs": {
                "http://%s:%s"%(localhost, health_httpd_port): {}
            },
        },
        "Keepstore": {
            "InternalURLs": {
                "http://%s:%s"%(localhost, port): {} for port in keepstore_ports
            },
        },
        "Keepproxy": {
            "ExternalURL": "https://%s:%s" % (localhost, keepproxy_external_port),
            "InternalURLs": {
                "http://%s:%s"%(localhost, keepproxy_port): {},
            },
        },
        "WebDAV": {
            "ExternalURL": "https://%s:%s" % (localhost, keep_web_external_port),
            "InternalURLs": {
                "http://%s:%s"%(localhost, keep_web_port): {},
            },
        },
        "WebDAVDownload": {
            "ExternalURL": "https://%s:%s" % (localhost, keep_web_dl_external_port),
            "InternalURLs": {
                "http://%s:%s"%(localhost, keep_web_port): {},
            },
        },
    }

    config = {
        "Clusters": {
            "zzzzz": {
                "ManagementToken": "e687950a23c3a9bceec28c6223a06c79",
                "SystemRootToken": auth_token('system_user'),
                "API": {
                    "RequestTimeout": "30s",
                },
                "Login": {
                    "Test": {
                        "Enable": True,
                        "Users": {
                            "alice": {
                                "Email": "alice@example.com",
                                "Password": "xyzzy"
                            }
                        }
                    },
                },
                "SystemLogs": {
                    "LogLevel": ('info' if os.environ.get('ARVADOS_DEBUG', '') in ['','0'] else 'debug'),
                },
                "PostgreSQL": {
                    "Connection": pgconnection,
                },
                "TLS": {
                    "Insecure": True,
                },
                "Services": services,
                "Users": {
                    "AnonymousUserToken": auth_token('anonymous'),
                    "UserProfileNotificationAddress": "arvados@example.com",
                },
                "Collections": {
                    "CollectionVersioning": True,
                    "BlobSigningKey": "zfhgfenhffzltr9dixws36j1yhksjoll2grmku38mi7yxd66h5j4q9w4jzanezacp8s6q0ro3hxakfye02152hncy6zml2ed0uc",
                    "TrustAllContent": False,
                    "ForwardSlashNameSubstitution": "/",
                    "TrashSweepInterval": "-1s",
                },
                "Git": {
                    "Repositories": os.path.join(SERVICES_SRC_DIR, 'api', 'tmp', 'git', 'test'),
                },
                "Containers": {
                    "JobsAPI": {
                        "GitInternalDir": os.path.join(SERVICES_SRC_DIR, 'api', 'tmp', 'internal.git'),
                    },
                    "LocalKeepBlobBuffersPerVCPU": 0,
                    "Logging": {
                        "SweepInterval": 0, # disable, otherwise test cases can't acquire dblock
                    },
                    "SupportedDockerImageFormats": {"v1": {}},
                    "ShellAccess": {
                        "Admin": True,
                        "User": True,
                    },
                },
                "Volumes": {
                    "zzzzz-nyw5e-%015d"%n: {
                        "AccessViaHosts": {
                            "http://%s:%s" % (localhost, keepstore_ports[n]): {},
                        },
                        "Driver": "Directory",
                        "DriverParameters": {
                            "Root": os.path.join(TEST_TMPDIR, "keep%d.data"%n),
                        },
                    } for n in range(len(keepstore_ports))
                },
            },
        },
    }

    conf = os.path.join(TEST_TMPDIR, 'arvados.yml')
    with open(conf, 'w') as f:
        yaml.safe_dump(config, f)

    ex = "export ARVADOS_CONFIG="+conf
    print(ex)


def stop_nginx():
    if 'ARVADOS_TEST_PROXY_SERVICES' in os.environ:
        return
    kill_server_pid(_pidfile('nginx'))

def _pidfile(program):
    return os.path.join(TEST_TMPDIR, program + '.pid')

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
        return yaml.safe_load(yaml_file)

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
    WS_SERVER = None
    KEEP_SERVER = None
    KEEP_PROXY_SERVER = None
    KEEP_WEB_SERVER = None

    @staticmethod
    def _restore_dict(src, dest):
        for key in list(dest.keys()):
            if key not in src:
                del dest[key]
        dest.update(src)

    @classmethod
    def setUpClass(cls):
        cls._orig_environ = os.environ.copy()
        cls._orig_config = arvados.config.settings().copy()
        cls._cleanup_funcs = []
        os.environ.pop('ARVADOS_KEEP_SERVICES', None)
        for server_kwargs, start_func, stop_func in (
                (cls.MAIN_SERVER, run, reset),
                (cls.WS_SERVER, run_ws, stop_ws),
                (cls.KEEP_SERVER, run_keep, stop_keep),
                (cls.KEEP_PROXY_SERVER, run_keep_proxy, stop_keep_proxy),
                (cls.KEEP_WEB_SERVER, run_keep_web, stop_keep_web)):
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
    actions = [
        'start', 'stop',
        'start_ws', 'stop_ws',
        'start_controller', 'stop_controller',
        'start_keep', 'stop_keep',
        'start_keep_proxy', 'stop_keep_proxy',
        'start_keep-web', 'stop_keep-web',
        'start_githttpd', 'stop_githttpd',
        'start_nginx', 'stop_nginx', 'setup_config',
    ]
    parser = argparse.ArgumentParser()
    parser.add_argument('action', type=str, help="one of {}".format(actions))
    parser.add_argument('--auth', type=str, metavar='FIXTURE_NAME', help='Print authorization info for given api_client_authorizations fixture')
    parser.add_argument('--num-keep-servers', metavar='int', type=int, default=2, help="Number of keep servers desired")
    parser.add_argument('--keep-blob-signing', action="store_true", help="Enable blob signing for keepstore servers")

    args = parser.parse_args()

    if args.action not in actions:
        print("Unrecognized action '{}'. Actions are: {}.".
              format(args.action, actions),
              file=sys.stderr)
        sys.exit(1)
    # Create a new process group so our child processes don't exit on
    # ^C in run-tests.sh interactive mode.
    os.setpgid(0, 0)
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
    elif args.action == 'start_ws':
        run_ws()
    elif args.action == 'stop_ws':
        stop_ws()
    elif args.action == 'start_controller':
        run_controller()
    elif args.action == 'stop_controller':
        stop_controller()
    elif args.action == 'start_keep':
        run_keep(blob_signing=args.keep_blob_signing, num_servers=args.num_keep_servers)
    elif args.action == 'stop_keep':
        stop_keep(num_servers=args.num_keep_servers)
    elif args.action == 'start_keep_proxy':
        run_keep_proxy()
    elif args.action == 'stop_keep_proxy':
        stop_keep_proxy()
    elif args.action == 'start_githttpd':
        run_arv_git_httpd()
    elif args.action == 'stop_githttpd':
        stop_arv_git_httpd()
    elif args.action == 'start_keep-web':
        run_keep_web()
    elif args.action == 'stop_keep-web':
        stop_keep_web()
    elif args.action == 'start_nginx':
        run_nginx()
        print("export ARVADOS_API_HOST=0.0.0.0:{}".format(external_port_from_config('Controller')))
    elif args.action == 'stop_nginx':
        stop_nginx()
    elif args.action == 'setup_config':
        setup_config()
    else:
        raise Exception("action recognized but not implemented!?")
