#!/usr/bin/env python

from __future__ import print_function
import argparse
import atexit
import errno
import httplib2
import os
import pipes
import random
import re
import shutil
import signal
import socket
import subprocess
import string
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

import arvados
import arvados.config

ARVADOS_DIR = os.path.realpath(os.path.join(MY_DIRNAME, '../../..'))
SERVICES_SRC_DIR = os.path.join(ARVADOS_DIR, 'services')
if 'GOPATH' in os.environ:
    gopaths = os.environ['GOPATH'].split(':')
    gobins = [os.path.join(path, 'bin') for path in gopaths]
    os.environ['PATH'] = ':'.join(gobins) + ':' + os.environ['PATH']

TEST_TMPDIR = os.path.join(ARVADOS_DIR, 'tmp')
if not os.path.exists(TEST_TMPDIR):
    os.mkdir(TEST_TMPDIR)

my_api_host = None
_cached_config = {}

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
        startTERM += wait/2

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
                return
            raise
        time.sleep(0.1)
        now = time.time()

    print("Server PID {} ({}) did not exit, giving up after {}s".
          format(server_pid, pidfile, wait),
          file=sys.stderr)

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

    sock = socket.socket()
    sock.bind(('0.0.0.0', 0))
    port = sock.getsockname()[1]
    sock.close()
    return port

def _wait_until_port_listens(port, timeout=10):
    """Wait for a process to start listening on the given port.

    If nothing listens on the port within the specified timeout (given
    in seconds), print a warning on stderr before returning.
    """
    try:
        subprocess.check_output(['which', 'lsof'])
    except subprocess.CalledProcessError:
        print("WARNING: No `lsof` -- cannot wait for port to listen. "+
              "Sleeping 0.5 and hoping for the best.",
              file=sys.stderr)
        time.sleep(0.5)
        return
    deadline = time.time() + timeout
    while time.time() < deadline:
        try:
            subprocess.check_output(
                ['lsof', '-t', '-i', 'tcp:'+str(port)])
        except subprocess.CalledProcessError:
            time.sleep(0.1)
            continue
        return
    print(
        "WARNING: Nothing is listening on port {} (waited {} seconds).".
        format(port, timeout),
        file=sys.stderr)

def _fifo2stderr(label):
    """Create a fifo, and copy it to stderr, prepending label to each line.

    Return value is the path to the new FIFO.

    +label+ should contain only alphanumerics: it is also used as part
    of the FIFO filename.
    """
    fifo = os.path.join(TEST_TMPDIR, label+'.fifo')
    try:
        os.remove(fifo)
    except OSError as error:
        if error.errno != errno.ENOENT:
            raise
    os.mkfifo(fifo, 0700)
    subprocess.Popen(
        ['sed', '-e', 's/^/['+label+'] /', fifo],
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

    # Delete cached discovery document.
    shutil.rmtree(arvados.http_cache('discovery'))

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
            '-subj', '/CN=0.0.0.0'],
        stdout=sys.stderr)

    # Install the git repository fixtures.
    gitdir = os.path.join(SERVICES_SRC_DIR, 'api', 'tmp', 'git')
    gittarball = os.path.join(SERVICES_SRC_DIR, 'api', 'test', 'test.git.tar')
    if not os.path.isdir(gitdir):
        os.makedirs(gitdir)
    subprocess.check_output(['tar', '-xC', gitdir, '-f', gittarball])

    port = find_available_port()
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
         '--pid-file', pid_file,
         '--log-file', os.path.join(os.getcwd(), 'log/test.log'),
         '--ssl',
         '--ssl-certificate', 'tmp/self-signed.pem',
         '--ssl-certificate-key', 'tmp/self-signed.key'],
        env=env)

    if not leave_running_atexit:
        atexit.register(kill_server_pid, pid_file, passenger_root=api_src_dir)

    match = re.search(r'Accessible via: https://(.*?)/', start_msg)
    if not match:
        raise Exception(
            "Passenger did not report endpoint: {}".format(start_msg))
    my_api_host = match.group(1)
    os.environ['ARVADOS_API_HOST'] = my_api_host

    # Make sure the server has written its pid file and started
    # listening on its TCP port
    find_server_pid(pid_file)
    _wait_until_port_listens(port)

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
        headers={'Authorization': 'OAuth2 {}'.format(token)})
    os.environ['ARVADOS_API_HOST_INSECURE'] = 'true'
    os.environ['ARVADOS_API_HOST'] = existing_api_host
    os.environ['ARVADOS_API_TOKEN'] = token

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

def _start_keep(n, keep_args):
    keep0 = tempfile.mkdtemp()
    port = find_available_port()
    keep_cmd = ["keepstore",
                "-volume={}".format(keep0),
                "-listen=:{}".format(port),
                "-pid="+_pidfile('keep{}'.format(n))]

    for arg, val in keep_args.iteritems():
        keep_cmd.append("{}={}".format(arg, val))

    logf = open(_fifo2stderr('keep{}'.format(n)), 'w')
    kp0 = subprocess.Popen(
        keep_cmd, stdin=open('/dev/null'), stdout=logf, stderr=logf, close_fds=True)

    with open(_pidfile('keep{}'.format(n)), 'w') as f:
        f.write(str(kp0.pid))

    with open("{}/keep{}.volume".format(TEST_TMPDIR, n), 'w') as f:
        f.write(keep0)

    _wait_until_port_listens(port)

    return port

def run_keep(blob_signing_key=None, enforce_permissions=False, num_servers=2):
    stop_keep(num_servers)

    keep_args = {}
    if not blob_signing_key:
        blob_signing_key = 'zfhgfenhffzltr9dixws36j1yhksjoll2grmku38mi7yxd66h5j4q9w4jzanezacp8s6q0ro3hxakfye02152hncy6zml2ed0uc'
    with open(os.path.join(TEST_TMPDIR, "keep.blob_signing_key"), "w") as f:
        keep_args['-blob-signing-key-file'] = f.name
        f.write(blob_signing_key)
    if enforce_permissions:
        keep_args['-enforce-permissions'] = 'true'
    with open(os.path.join(TEST_TMPDIR, "keep.data-manager-token-file"), "w") as f:
        keep_args['-data-manager-token-file'] = f.name
        f.write(auth_token('data_manager'))
    keep_args['-never-delete'] = 'false'

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

    # If keepproxy is running, send SIGHUP to make it discover the new
    # keepstore services.
    proxypidfile = _pidfile('keepproxy')
    if os.path.exists(proxypidfile):
        try:
            os.kill(int(open(proxypidfile).read()), signal.SIGHUP)
        except OSError:
            os.remove(proxypidfile)

def _stop_keep(n):
    kill_server_pid(_pidfile('keep{}'.format(n)))
    if os.path.exists("{}/keep{}.volume".format(TEST_TMPDIR, n)):
        with open("{}/keep{}.volume".format(TEST_TMPDIR, n), 'r') as r:
            shutil.rmtree(r.read(), True)
        os.unlink("{}/keep{}.volume".format(TEST_TMPDIR, n))
    if os.path.exists(os.path.join(TEST_TMPDIR, "keep.blob_signing_key")):
        os.remove(os.path.join(TEST_TMPDIR, "keep.blob_signing_key"))

def stop_keep(num_servers=2):
    for n in range(0, num_servers):
        _stop_keep(n)

def run_keep_proxy():
    if 'ARVADOS_TEST_PROXY_SERVICES' in os.environ:
        return
    stop_keep_proxy()

    port = find_available_port()
    env = os.environ.copy()
    env['ARVADOS_API_TOKEN'] = auth_token('anonymous')
    logf = open(_fifo2stderr('keepproxy'), 'w')
    kp = subprocess.Popen(
        ['keepproxy',
         '-pid='+_pidfile('keepproxy'),
         '-listen=:{}'.format(port)],
        env=env, stdin=open('/dev/null'), stdout=logf, stderr=logf, close_fds=True)

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
    os.environ["ARVADOS_KEEP_PROXY"] = "http://localhost:{}".format(port)
    _setport('keepproxy', port)
    _wait_until_port_listens(port)

def stop_keep_proxy():
    if 'ARVADOS_TEST_PROXY_SERVICES' in os.environ:
        return
    kill_server_pid(_pidfile('keepproxy'))

def run_arv_git_httpd():
    if 'ARVADOS_TEST_PROXY_SERVICES' in os.environ:
        return
    stop_arv_git_httpd()

    gitdir = os.path.join(SERVICES_SRC_DIR, 'api', 'tmp', 'git')
    gitport = find_available_port()
    env = os.environ.copy()
    env.pop('ARVADOS_API_TOKEN', None)
    logf = open(_fifo2stderr('arv-git-httpd'), 'w')
    agh = subprocess.Popen(
        ['arv-git-httpd',
         '-repo-root='+gitdir+'/test',
         '-address=:'+str(gitport)],
        env=env, stdin=open('/dev/null'), stdout=logf, stderr=logf)
    with open(_pidfile('arv-git-httpd'), 'w') as f:
        f.write(str(agh.pid))
    _setport('arv-git-httpd', gitport)
    _wait_until_port_listens(gitport)

def stop_arv_git_httpd():
    if 'ARVADOS_TEST_PROXY_SERVICES' in os.environ:
        return
    kill_server_pid(_pidfile('arv-git-httpd'))

def run_keep_web():
    if 'ARVADOS_TEST_PROXY_SERVICES' in os.environ:
        return
    stop_keep_web()

    keepwebport = find_available_port()
    env = os.environ.copy()
    env['ARVADOS_API_TOKEN'] = auth_token('anonymous')
    logf = open(_fifo2stderr('keep-web'), 'w')
    keepweb = subprocess.Popen(
        ['keep-web',
         '-allow-anonymous',
         '-attachment-only-host=download:'+str(keepwebport),
         '-listen=:'+str(keepwebport)],
        env=env, stdin=open('/dev/null'), stdout=logf, stderr=logf)
    with open(_pidfile('keep-web'), 'w') as f:
        f.write(str(keepweb.pid))
    _setport('keep-web', keepwebport)
    _wait_until_port_listens(keepwebport)

def stop_keep_web():
    if 'ARVADOS_TEST_PROXY_SERVICES' in os.environ:
        return
    kill_server_pid(_pidfile('keep-web'))

def run_nginx():
    if 'ARVADOS_TEST_PROXY_SERVICES' in os.environ:
        return
    nginxconf = {}
    nginxconf['KEEPWEBPORT'] = _getport('keep-web')
    nginxconf['KEEPWEBDLSSLPORT'] = find_available_port()
    nginxconf['KEEPWEBSSLPORT'] = find_available_port()
    nginxconf['KEEPPROXYPORT'] = _getport('keepproxy')
    nginxconf['KEEPPROXYSSLPORT'] = find_available_port()
    nginxconf['GITPORT'] = _getport('arv-git-httpd')
    nginxconf['GITSSLPORT'] = find_available_port()
    nginxconf['SSLCERT'] = os.path.join(SERVICES_SRC_DIR, 'api', 'tmp', 'self-signed.pem')
    nginxconf['SSLKEY'] = os.path.join(SERVICES_SRC_DIR, 'api', 'tmp', 'self-signed.key')
    nginxconf['ACCESSLOG'] = _fifo2stderr('nginx_access_log')

    conftemplatefile = os.path.join(MY_DIRNAME, 'nginx.conf')
    conffile = os.path.join(TEST_TMPDIR, 'nginx.conf')
    with open(conffile, 'w') as f:
        f.write(re.sub(
            r'{{([A-Z]+)}}',
            lambda match: str(nginxconf.get(match.group(1))),
            open(conftemplatefile).read()))

    env = os.environ.copy()
    env['PATH'] = env['PATH']+':/sbin:/usr/sbin:/usr/local/sbin'

    nginx = subprocess.Popen(
        ['nginx',
         '-g', 'error_log stderr info;',
         '-g', 'pid '+_pidfile('nginx')+';',
         '-c', conffile],
        env=env, stdin=open('/dev/null'), stdout=sys.stderr)
    _setport('keep-web-dl-ssl', nginxconf['KEEPWEBDLSSLPORT'])
    _setport('keep-web-ssl', nginxconf['KEEPWEBSSLPORT'])
    _setport('keepproxy-ssl', nginxconf['KEEPPROXYSSLPORT'])
    _setport('arv-git-httpd-ssl', nginxconf['GITSSLPORT'])

def stop_nginx():
    if 'ARVADOS_TEST_PROXY_SERVICES' in os.environ:
        return
    kill_server_pid(_pidfile('nginx'))

def _pidfile(program):
    return os.path.join(TEST_TMPDIR, program + '.pid')

def _portfile(program):
    return os.path.join(TEST_TMPDIR, program + '.port')

def _setport(program, port):
    with open(_portfile(program), 'w') as f:
        f.write(str(port))

# Returns 9 if program is not up.
def _getport(program):
    try:
        return int(open(_portfile(program)).read())
    except IOError:
        return 9

def _apiconfig(key):
    if _cached_config:
        return _cached_config[key]
    def _load(f, required=True):
        fullpath = os.path.join(SERVICES_SRC_DIR, 'api', 'config', f)
        if not required and not os.path.exists(fullpath):
            return {}
        return yaml.load(fullpath)
    cdefault = _load('application.default.yml')
    csite = _load('application.yml', required=False)
    _cached_config = {}
    for section in [cdefault.get('common',{}), cdefault.get('test',{}),
                    csite.get('common',{}), csite.get('test',{})]:
        _cached_config.update(section)
    return _cached_config[key]

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
    KEEP_WEB_SERVER = None

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
        'start_keep', 'stop_keep',
        'start_keep_proxy', 'stop_keep_proxy',
        'start_keep-web', 'stop_keep-web',
        'start_arv-git-httpd', 'stop_arv-git-httpd',
        'start_nginx', 'stop_nginx',
    ]
    parser = argparse.ArgumentParser()
    parser.add_argument('action', type=str, help="one of {}".format(actions))
    parser.add_argument('--auth', type=str, metavar='FIXTURE_NAME', help='Print authorization info for given api_client_authorizations fixture')
    parser.add_argument('--num-keep-servers', metavar='int', type=int, default=2, help="Number of keep servers desired")
    parser.add_argument('--keep-enforce-permissions', action="store_true", help="Enforce keep permissions")

    args = parser.parse_args()

    if args.action not in actions:
        print("Unrecognized action '{}'. Actions are: {}.".
              format(args.action, actions),
              file=sys.stderr)
        sys.exit(1)
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
        run_keep(enforce_permissions=args.keep_enforce_permissions, num_servers=args.num_keep_servers)
    elif args.action == 'stop_keep':
        stop_keep(num_servers=args.num_keep_servers)
    elif args.action == 'start_keep_proxy':
        run_keep_proxy()
    elif args.action == 'stop_keep_proxy':
        stop_keep_proxy()
    elif args.action == 'start_arv-git-httpd':
        run_arv_git_httpd()
    elif args.action == 'stop_arv-git-httpd':
        stop_arv_git_httpd()
    elif args.action == 'start_keep-web':
        run_keep_web()
    elif args.action == 'stop_keep-web':
        stop_keep_web()
    elif args.action == 'start_nginx':
        run_nginx()
    elif args.action == 'stop_nginx':
        stop_nginx()
    else:
        raise Exception("action recognized but not implemented!?")
