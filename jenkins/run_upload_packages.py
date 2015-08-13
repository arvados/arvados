#!/usr/bin/env python3

import argparse
import functools
import glob
import logging
import os
import pipes
import shutil
import subprocess
import sys
import time

class TimestampFile:
    def __init__(self, path):
        self.path = path
        self.start_time = time.time()

    def last_upload(self):
        try:
            return os.path.getmtime(self.path)
        except EnvironmentError:
            return -1

    def update(self):
        os.close(os.open(self.path, os.O_CREAT | os.O_APPEND))
        os.utime(self.path, (time.time(), self.start_time))


class PackageSuite:
    NEED_SSH = False

    def __init__(self, glob_root, rel_globs):
        logger_part = getattr(self, 'LOGGER_PART', os.path.basename(glob_root))
        self.logger = logging.getLogger('arvados-dev.upload.' + logger_part)
        self.globs = [os.path.join(glob_root, rel_glob)
                      for rel_glob in rel_globs]

    def files_to_upload(self, since_timestamp):
        for abs_glob in self.globs:
            for path in glob.glob(abs_glob):
                if os.path.getmtime(path) >= since_timestamp:
                    yield path

    def upload_file(self, path):
        raise NotImplementedError("PackageSuite.upload_file")

    def upload_files(self, paths):
        for path in paths:
            self.logger.info("Uploading %s", path)
            self.upload_file(path)

    def post_uploads(self, paths):
        pass

    def update_packages(self, since_timestamp):
        upload_paths = list(self.files_to_upload(since_timestamp))
        if upload_paths:
            self.upload_files(upload_paths)
            self.post_uploads(upload_paths)


class PythonPackageSuite(PackageSuite):
    LOGGER_PART = 'python'

    def __init__(self, glob_root, rel_globs):
        super().__init__(glob_root, rel_globs)
        self.seen_packages = set()

    def upload_file(self, path):
        src_dir = os.path.dirname(os.path.dirname(path))
        if src_dir in self.seen_packages:
            return
        self.seen_packages.add(src_dir)
        # NOTE: If we ever start uploading Python 3 packages, we'll need to
        # figure out some way to adapt cmd to match.  It might be easiest
        # to give all our setup.py files the executable bit, and run that
        # directly.
        # We also must run `sdist` before `upload`: `upload` uploads any
        # distributions previously generated in the command.  It doesn't
        # know how to upload distributions already on disk.  We write the
        # result to a dedicated directory to avoid interfering with our
        # timestamp tracking.
        cmd = ['python2.7', 'setup.py']
        if not self.logger.isEnabledFor(logging.INFO):
            cmd.append('--quiet')
        cmd.extend(['sdist', '--dist-dir', '.upload_dist', 'upload'])
        subprocess.check_call(cmd, cwd=src_dir)
        shutil.rmtree(os.path.join(src_dir, '.upload_dist'))


class GemPackageSuite(PackageSuite):
    LOGGER_PART = 'gems'

    def upload_file(self, path):
        cmd = ['gem', 'push', path]
        push_proc = subprocess.Popen(cmd, stdout=subprocess.PIPE)
        repushed = any(line == b'Repushing of gem versions is not allowed.\n'
                       for line in push_proc.stdout)
        # Read any remaining stdout before closing.
        for line in push_proc.stdout:
            pass
        push_proc.stdout.close()
        if (push_proc.wait() != 0) and not repushed:
            raise subprocess.CalledProcessError(push_proc.returncode, cmd)


class DistroPackageSuite(PackageSuite):
    NEED_SSH = True
    REMOTE_DEST_DIR = 'tmp'

    def __init__(self, glob_root, rel_globs, target, ssh_host, ssh_opts):
        super().__init__(glob_root, rel_globs)
        self.target = target
        self.ssh_host = ssh_host
        self.ssh_opts = ['-o' + opt for opt in ssh_opts]
        if not self.logger.isEnabledFor(logging.INFO):
            self.ssh_opts.append('-q')

    def _build_cmd(self, base_cmd, *args):
        cmd = [base_cmd]
        cmd.extend(self.ssh_opts)
        cmd.extend(args)
        return cmd

    def _paths_basenames(self, paths):
        return (os.path.basename(path) for path in paths)

    def _run_script(self, script, *args):
        # SSH will use a shell to run our bash command, so we have to
        # quote our arguments.
        # self.__class__.__name__ provides $0 for the script, which makes a
        # nicer message if there's an error.
        subprocess.check_call(self._build_cmd(
                'ssh', self.ssh_host, 'bash', '-ec', pipes.quote(script),
                self.__class__.__name__, *(pipes.quote(s) for s in args)))

    def upload_files(self, paths):
        cmd = self._build_cmd('scp', *paths)
        cmd.append('{self.ssh_host}:{self.REMOTE_DEST_DIR}'.format(self=self))
        subprocess.check_call(cmd)


class DebianPackageSuite(DistroPackageSuite):
    FREIGHT_SCRIPT = """
cd "$1"; shift
DISTNAME=$1; shift
freight add "$@" "apt/$DISTNAME"
freight cache
rm "$@"
"""
    TARGET_DISTNAMES = {
        'debian7': 'wheezy',
        'debian8': 'jessie',
        'ubuntu1204': 'precise',
        'ubuntu1404': 'trusty',
        }

    def post_uploads(self, paths):
        self._run_script(self.FREIGHT_SCRIPT, self.REMOTE_DEST_DIR,
                         self.TARGET_DISTNAMES[self.target],
                         *self._paths_basenames(paths))


class RedHatPackageSuite(DistroPackageSuite):
    CREATEREPO_SCRIPT = """
cd "$1"; shift
REPODIR=$1; shift
rpmsign --addsign "$@" </dev/null
mv "$@" "$REPODIR"
createrepo "$REPODIR"
"""
    REPO_ROOT = '/var/www/rpm.arvados.org/'
    TARGET_REPODIRS = {
        'centos6': 'CentOS/6/os/x86_64/'
        }

    def post_uploads(self, paths):
        repo_dir = os.path.join(self.REPO_ROOT,
                                self.TARGET_REPODIRS[self.target])
        self._run_script(self.CREATEREPO_SCRIPT, self.REMOTE_DEST_DIR,
                         repo_dir, *self._paths_basenames(paths))


def _define_suite(suite_class, *rel_globs, **kwargs):
    return functools.partial(suite_class, rel_globs=rel_globs, **kwargs)

PACKAGE_SUITES = {
    'python': _define_suite(PythonPackageSuite,
                            'sdk/pam/dist/*.tar.gz',
                            'sdk/python/dist/*.tar.gz',
                            'services/nodemanager/dist/*.tar.gz',
                            'services/fuse/dist/*.tar.gz',
                        ),
    'gems': _define_suite(GemPackageSuite, 'sdk/ruby/*.gem', 'sdk/cli/*.gem'),
    }
for target in ['debian7', 'debian8', 'ubuntu1204', 'ubuntu1404']:
    PACKAGE_SUITES[target] = _define_suite(
        DebianPackageSuite, os.path.join('packages', target, '*.deb'),
        target=target)
for target in ['centos6']:
    PACKAGE_SUITES[target] = _define_suite(
        RedHatPackageSuite, os.path.join('packages', target, '*.rpm'),
        target=target)

def parse_arguments(arguments):
    parser = argparse.ArgumentParser(
        prog="run_upload_packages.py",
        description="Upload Arvados packages to various repositories")
    parser.add_argument(
        '--workspace', '-W', default=os.environ.get('WORKSPACE'),
        help="Arvados source directory with built packages to upload")
    parser.add_argument(
        '--ssh-host', '-H',
        help="Host specification for distribution repository server")
    parser.add_argument('-o', action='append', default=[], dest='ssh_opts',
                         metavar='OPTION', help="Pass option to `ssh -o`")
    parser.add_argument('--verbose', '-v', action='count', default=0,
                        help="Log more information and subcommand output")
    parser.add_argument(
        'targets', nargs='*', default=['all'], metavar='target',
        help="Upload packages to these targets (default all)\nAvailable targets: " +
        ', '.join(sorted(PACKAGE_SUITES.keys())))
    args = parser.parse_args(arguments)
    if 'all' in args.targets:
        args.targets = list(PACKAGE_SUITES.keys())

    if args.workspace is None:
        parser.error("workspace not set from command line or environment")
    for target in args.targets:
        try:
            suite_class = PACKAGE_SUITES[target].func
        except KeyError:
            parser.error("unrecognized target {!r}".format(target))
        if suite_class.NEED_SSH and (args.ssh_host is None):
            parser.error(
                "--ssh-host must be specified to upload distribution packages")
    return args

def setup_logger(stream_dest, args):
    log_handler = logging.StreamHandler(stream_dest)
    log_handler.setFormatter(logging.Formatter(
            '%(asctime)s %(name)s[%(process)d] %(levelname)s: %(message)s',
            '%Y-%m-%d %H:%M:%S'))
    logger = logging.getLogger('arvados-dev.upload')
    logger.addHandler(log_handler)
    logger.setLevel(max(1, logging.WARNING - (10 * args.verbose)))

def build_suite_and_upload(target, since_timestamp, args):
    suite_def = PACKAGE_SUITES[target]
    kwargs = {}
    if suite_def.func.NEED_SSH:
        kwargs.update(ssh_host=args.ssh_host, ssh_opts=args.ssh_opts)
    suite = suite_def(args.workspace, **kwargs)
    suite.update_packages(since_timestamp)

def main(arguments, stdout=sys.stdout, stderr=sys.stderr):
    args = parse_arguments(arguments)
    setup_logger(stderr, args)
    ts_file = TimestampFile(os.path.join(args.workspace, 'packages',
                                         '.last_upload'))
    last_upload_ts = ts_file.last_upload()
    for target in args.targets:
        build_suite_and_upload(target, last_upload_ts, args)
    ts_file.update()

if __name__ == '__main__':
    main(sys.argv[1:])
