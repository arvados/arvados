# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

from future.utils import native_str
from builtins import range
from builtins import object
import argparse
import arvados
import daemon
import llfuse
import logging
import os
import resource
import signal
import subprocess
import sys
import time
import resource

import arvados.commands._util as arv_cmd
from arvados_fuse import crunchstat
from arvados_fuse import *
from arvados_fuse.unmount import unmount
from arvados_fuse._version import __version__

class ArgumentParser(argparse.ArgumentParser):
    def __init__(self):
        super(ArgumentParser, self).__init__(
            parents=[arv_cmd.retry_opt],
            description='''Mount Keep data under the local filesystem.  Default mode is --home''',
            epilog="""
    Note: When using the --exec feature, you must either specify the
    mountpoint before --exec, or mark the end of your --exec arguments
    with "--".
            """)
        self.add_argument('--version', action='version',
                          version=u"%s %s" % (sys.argv[0], __version__),
                          help='Print version and exit.')
        self.add_argument('mountpoint', type=str, help="""Mount point.""")
        self.add_argument('--allow-other', action='store_true',
                            help="""Let other users read the mount""")
        self.add_argument('--subtype', type=str, metavar='STRING',
                            help="""Report mounted filesystem type as "fuse.STRING", instead of just "fuse".""")

        mode = self.add_mutually_exclusive_group()

        mode.add_argument('--all', action='store_const', const='all', dest='mode',
                                help="""Mount a subdirectory for each mode: home, shared, by_tag, by_id (default if no --mount-* arguments are given).""")
        mode.add_argument('--custom', action='store_const', const=None, dest='mode',
                                help="""Mount a top level meta-directory with subdirectories as specified by additional --mount-* arguments (default if any --mount-* arguments are given).""")
        mode.add_argument('--home', action='store_const', const='home', dest='mode',
                                help="""Mount only the user's home project.""")
        mode.add_argument('--shared', action='store_const', const='shared', dest='mode',
                                help="""Mount only list of projects shared with the user.""")
        mode.add_argument('--by-tag', action='store_const', const='by_tag', dest='mode',
                                help="""Mount subdirectories listed by tag.""")
        mode.add_argument('--by-id', action='store_const', const='by_id', dest='mode',
                                help="""Mount subdirectories listed by portable data hash or uuid.""")
        mode.add_argument('--by-pdh', action='store_const', const='by_pdh', dest='mode',
                                help="""Mount subdirectories listed by portable data hash.""")
        mode.add_argument('--project', type=str, metavar='UUID',
                                help="""Mount the specified project.""")
        mode.add_argument('--collection', type=str, metavar='UUID_or_PDH',
                                help="""Mount only the specified collection.""")

        mounts = self.add_argument_group('Custom mount options')
        mounts.add_argument('--mount-by-pdh',
                            type=str, metavar='PATH', action='append', default=[],
                            help="Mount each readable collection at mountpoint/PATH/P where P is the collection's portable data hash.")
        mounts.add_argument('--mount-by-id',
                            type=str, metavar='PATH', action='append', default=[],
                            help="Mount each readable collection at mountpoint/PATH/UUID and mountpoint/PATH/PDH where PDH is the collection's portable data hash and UUID is its UUID.")
        mounts.add_argument('--mount-by-tag',
                            type=str, metavar='PATH', action='append', default=[],
                            help="Mount all collections with tag TAG at mountpoint/PATH/TAG/UUID.")
        mounts.add_argument('--mount-home',
                            type=str, metavar='PATH', action='append', default=[],
                            help="Mount the current user's home project at mountpoint/PATH.")
        mounts.add_argument('--mount-shared',
                            type=str, metavar='PATH', action='append', default=[],
                            help="Mount projects shared with the current user at mountpoint/PATH.")
        mounts.add_argument('--mount-tmp',
                            type=str, metavar='PATH', action='append', default=[],
                            help="Create a new collection, mount it in read/write mode at mountpoint/PATH, and delete it when unmounting.")


        self.add_argument('--debug', action='store_true', help="""Debug mode""")
        self.add_argument('--logfile', help="""Write debug logs and errors to the specified file (default stderr).""")
        self.add_argument('--foreground', action='store_true', help="""Run in foreground (default is to daemonize unless --exec specified)""", default=False)
        self.add_argument('--encoding', type=str, help="Character encoding to use for filesystem, default is utf-8 (see Python codec registry for list of available encodings)", default="utf-8")

        self.add_argument('--file-cache', type=int, help="File data cache size, in bytes (default 8 GiB for disk-based cache or 256 MiB with RAM-only cache)", default=0)
        self.add_argument('--directory-cache', type=int, help="Directory data cache size, in bytes (default 128 MiB)", default=128*1024*1024)

        cachetype = self.add_mutually_exclusive_group()
        cachetype.add_argument('--ram-cache', action='store_false', dest='disk_cache', help="Use in-memory caching only", default=True)
        cachetype.add_argument('--disk-cache', action='store_true', dest='disk_cache', help="Use disk based caching (default)", default=True)

        self.add_argument('--disk-cache-dir', type=str, help="Disk cache location (default ~/.cache/arvados/keep)", default=None)

        self.add_argument('--disable-event-listening', action='store_true', help="Don't subscribe to events on the API server", dest="disable_event_listening", default=False)

        self.add_argument('--read-only', action='store_false', help="Mount will be read only (default)", dest="enable_write", default=False)
        self.add_argument('--read-write', action='store_true', help="Mount will be read-write", dest="enable_write", default=False)
        self.add_argument('--storage-classes', type=str, metavar='CLASSES', help="Specify comma separated list of storage classes to be used when saving data of new collections", default=None)

        self.add_argument('--crunchstat-interval', type=float, help="Write stats to stderr every N seconds (default disabled)", default=0)

        unmount = self.add_mutually_exclusive_group()
        unmount.add_argument('--unmount', action='store_true', default=False,
                             help="Forcefully unmount the specified mountpoint (if it's a fuse mount) and exit. If --subtype is given, unmount only if the mount has the specified subtype. WARNING: This command can affect any kind of fuse mount, not just arv-mount.")
        unmount.add_argument('--unmount-all', action='store_true', default=False,
                             help="Forcefully unmount every fuse mount at or below the specified path and exit. If --subtype is given, unmount only mounts that have the specified subtype. Exit non-zero if any other types of mounts are found at or below the given path. WARNING: This command can affect any kind of fuse mount, not just arv-mount.")
        unmount.add_argument('--replace', action='store_true', default=False,
                             help="If a fuse mount is already present at mountpoint, forcefully unmount it before mounting")
        self.add_argument('--unmount-timeout',
                          type=float, default=2.0,
                          help="Time to wait for graceful shutdown after --exec program exits and filesystem is unmounted")

        self.add_argument('--exec', type=str, nargs=argparse.REMAINDER,
                            dest="exec_args", metavar=('command', 'args', '...', '--'),
                            help="""Mount, run a command, then unmount and exit""")


class Mount(object):
    def __init__(self, args, logger=logging.getLogger('arvados.arv-mount')):
        self.daemon = False
        self.logger = logger
        self.args = args
        self.listen_for_events = False

        self.args.mountpoint = os.path.realpath(self.args.mountpoint)
        if self.args.logfile:
            self.args.logfile = os.path.realpath(self.args.logfile)

        try:
            nofile_limit = resource.getrlimit(resource.RLIMIT_NOFILE)
            if nofile_limit[0] < 10240:
                resource.setrlimit(resource.RLIMIT_NOFILE, (min(10240, nofile_limit[1]), nofile_limit[1]))
        except Exception as e:
            self.logger.warning("arv-mount: unable to adjust file handle limit: %s", e)

        self.logger.debug("arv-mount: file handle limit is %s", resource.getrlimit(resource.RLIMIT_NOFILE))

        try:
            self._setup_logging()
            self._setup_api()
            self._setup_mount()
        except Exception as e:
            self.logger.exception("arv-mount: exception during setup: %s", e)
            exit(1)

    def __enter__(self):
        if self.args.replace:
            unmount(path=self.args.mountpoint,
                    timeout=self.args.unmount_timeout)
        llfuse.init(self.operations, native_str(self.args.mountpoint), self._fuse_options())
        if self.daemon:
            daemon.DaemonContext(
                working_directory=os.path.dirname(self.args.mountpoint),
                files_preserve=list(range(
                    3, resource.getrlimit(resource.RLIMIT_NOFILE)[1]))
            ).open()
        if self.listen_for_events and not self.args.disable_event_listening:
            self.operations.listen_for_events()
        self.llfuse_thread = threading.Thread(None, lambda: self._llfuse_main())
        self.llfuse_thread.daemon = True
        self.llfuse_thread.start()
        self.operations.initlock.wait()
        return self

    def __exit__(self, exc_type, exc_value, traceback):
        if self.operations.events:
            self.operations.events.close(timeout=self.args.unmount_timeout)
        subprocess.call(["fusermount", "-u", "-z", self.args.mountpoint])
        self.llfuse_thread.join(timeout=self.args.unmount_timeout)
        if self.llfuse_thread.is_alive():
            self.logger.warning("Mount.__exit__:"
                                " llfuse thread still alive %fs after umount"
                                " -- abandoning and exiting anyway",
                                self.args.unmount_timeout)

    def run(self):
        if self.args.unmount or self.args.unmount_all:
            unmount(path=self.args.mountpoint,
                    subtype=self.args.subtype,
                    timeout=self.args.unmount_timeout,
                    recursive=self.args.unmount_all)
        elif self.args.exec_args:
            self._run_exec()
        else:
            self._run_standalone()

    def _fuse_options(self):
        """FUSE mount options; see mount.fuse(8)"""
        opts = [optname for optname in ['allow_other', 'debug']
                if getattr(self.args, optname)]
        # Increase default read/write size from 4KiB to 128KiB
        opts += ["big_writes", "max_read=131072"]
        if self.args.subtype:
            opts += ["subtype="+self.args.subtype]
        return opts

    def _setup_logging(self):
        # Configure a log handler based on command-line switches.
        if self.args.logfile:
            log_handler = logging.FileHandler(self.args.logfile)
            log_handler.setFormatter(logging.Formatter(
                '%(asctime)s %(name)s[%(process)d] %(levelname)s: %(message)s',
                '%Y-%m-%d %H:%M:%S'))
        else:
            log_handler = None

        if log_handler is not None:
            arvados.logger.removeHandler(arvados.log_handler)
            arvados.logger.addHandler(log_handler)

        if self.args.debug:
            arvados.logger.setLevel(logging.DEBUG)
            logging.getLogger('arvados.keep').setLevel(logging.DEBUG)
            logging.getLogger('arvados.api').setLevel(logging.DEBUG)
            logging.getLogger('arvados.collection').setLevel(logging.DEBUG)
            self.logger.debug("arv-mount debugging enabled")

        self.logger.info("%s %s started", sys.argv[0], __version__)
        self.logger.info("enable write is %s", self.args.enable_write)

    def _setup_api(self):
        try:
            self.api = arvados.safeapi.ThreadSafeApiCache(
                apiconfig=arvados.config.settings(),
                api_params={
                    'num_retries': self.args.retries,
                },
                # default value of file_cache is 0, this tells KeepBlockCache to
                # choose a default based on whether disk_cache is enabled or not.
                keep_params={
                    'block_cache': arvados.keep.KeepBlockCache(cache_max=self.args.file_cache,
                                                               disk_cache=self.args.disk_cache,
                                                               disk_cache_dir=self.args.disk_cache_dir),
                    'num_retries': self.args.retries,
                },
                version='v1',
            )
        except KeyError as e:
            self.logger.error("Missing environment: %s", e)
            exit(1)
        # Do a sanity check that we have a working arvados host + token.
        self.api.users().current().execute()

    def _setup_mount(self):
        self.operations = Operations(
            os.getuid(),
            os.getgid(),
            api_client=self.api,
            encoding=self.args.encoding,
            inode_cache=InodeCache(cap=self.args.directory_cache),
            enable_write=self.args.enable_write)

        if self.args.crunchstat_interval:
            statsthread = threading.Thread(
                target=crunchstat.statlogger,
                args=(self.args.crunchstat_interval,
                      self.api.keep,
                      self.operations))
            statsthread.daemon = True
            statsthread.start()

        usr = self.api.users().current().execute(num_retries=self.args.retries)
        now = time.time()
        dir_class = None
        dir_args = [llfuse.ROOT_INODE, self.operations.inodes, self.api, self.args.retries, self.args.enable_write]
        mount_readme = False

        storage_classes = None
        if self.args.storage_classes is not None:
            storage_classes = self.args.storage_classes.replace(' ', '').split(',')
            self.logger.info("Storage classes requested for new collections: {}".format(', '.join(storage_classes)))

        if self.args.collection is not None:
            # Set up the request handler with the collection at the root
            # First check that the collection is readable
            self.api.collections().get(uuid=self.args.collection).execute()
            self.args.mode = 'collection'
            dir_class = CollectionDirectory
            dir_args.append(self.args.collection)
        elif self.args.project is not None:
            self.args.mode = 'project'
            dir_class = ProjectDirectory
            dir_args.append(
                self.api.groups().get(uuid=self.args.project).execute(
                    num_retries=self.args.retries))

        if (self.args.mount_by_id or
            self.args.mount_by_pdh or
            self.args.mount_by_tag or
            self.args.mount_home or
            self.args.mount_shared or
            self.args.mount_tmp):
            if self.args.mode is not None:
                sys.exit(
                    "Cannot combine '{}' mode with custom --mount-* options.".
                    format(self.args.mode))
        elif self.args.mode is None:
            # If no --mount-custom or custom mount args, --all is the default
            self.args.mode = 'all'

        if self.args.mode in ['by_id', 'by_pdh']:
            # Set up the request handler with the 'magic directory' at the root
            dir_class = MagicDirectory
            dir_args.append(self.args.mode == 'by_pdh')
        elif self.args.mode == 'by_tag':
            dir_class = TagsDirectory
        elif self.args.mode == 'shared':
            dir_class = SharedDirectory
            dir_args.append(usr)
        elif self.args.mode == 'home':
            dir_class = ProjectDirectory
            dir_args.append(usr)
            dir_args.append(True)
        elif self.args.mode == 'all':
            self.args.mount_by_id = ['by_id']
            self.args.mount_by_tag = ['by_tag']
            self.args.mount_home = ['home']
            self.args.mount_shared = ['shared']
            mount_readme = True

        if dir_class is not None:
            if dir_class in [TagsDirectory, CollectionDirectory]:
                ent = dir_class(*dir_args)
            else:
                ent = dir_class(*dir_args, storage_classes=storage_classes)
            self.operations.inodes.add_entry(ent)
            self.listen_for_events = ent.want_event_subscribe()
            return

        e = self.operations.inodes.add_entry(Directory(
            llfuse.ROOT_INODE, self.operations.inodes, self.api.config, self.args.enable_write))
        dir_args[0] = e.inode

        for name in self.args.mount_by_id:
            self._add_mount(e, name, MagicDirectory(*dir_args, pdh_only=False, storage_classes=storage_classes))
        for name in self.args.mount_by_pdh:
            self._add_mount(e, name, MagicDirectory(*dir_args, pdh_only=True))
        for name in self.args.mount_by_tag:
            self._add_mount(e, name, TagsDirectory(*dir_args))
        for name in self.args.mount_home:
            self._add_mount(e, name, ProjectDirectory(*dir_args, project_object=usr, poll=True, storage_classes=storage_classes))
        for name in self.args.mount_shared:
            self._add_mount(e, name, SharedDirectory(*dir_args, exclude=usr, poll=True, storage_classes=storage_classes))
        for name in self.args.mount_tmp:
            self._add_mount(e, name, TmpCollectionDirectory(*dir_args, storage_classes=storage_classes))

        if mount_readme:
            text = self._readme_text(
                arvados.config.get('ARVADOS_API_HOST'),
                usr['email'])
            self._add_mount(e, 'README', StringFile(e.inode, text, now))

    def _add_mount(self, tld, name, ent):
        if name in ['', '.', '..'] or '/' in name:
            sys.exit("Mount point '{}' is not supported.".format(name))
        tld._entries[name] = self.operations.inodes.add_entry(ent)
        self.listen_for_events = (self.listen_for_events or ent.want_event_subscribe())

    def _readme_text(self, api_host, user_email):
        return '''
Welcome to Arvados!  This directory provides file system access to
files and objects available on the Arvados installation located at
'{}' using credentials for user '{}'.

From here, the following directories are available:

  by_id/     Access to Keep collections by uuid or portable data hash (see by_id/README for details).
  by_tag/    Access to Keep collections organized by tag.
  home/      The contents of your home project.
  shared/    Projects shared with you.

'''.format(api_host, user_email)

    def _run_exec(self):
        rc = 255
        with self:
            try:
                sp = subprocess.Popen(self.args.exec_args, shell=False)

                # forward signals to the process.
                signal.signal(signal.SIGINT, lambda signum, frame: sp.send_signal(signum))
                signal.signal(signal.SIGTERM, lambda signum, frame: sp.send_signal(signum))
                signal.signal(signal.SIGQUIT, lambda signum, frame: sp.send_signal(signum))

                # wait for process to complete.
                rc = sp.wait()

                # restore default signal handlers.
                signal.signal(signal.SIGINT, signal.SIG_DFL)
                signal.signal(signal.SIGTERM, signal.SIG_DFL)
                signal.signal(signal.SIGQUIT, signal.SIG_DFL)
            except Exception as e:
                self.logger.exception(
                    'arv-mount: exception during exec %s', self.args.exec_args)
                try:
                    rc = e.errno
                except AttributeError:
                    pass
        exit(rc)

    def _run_standalone(self):
        try:
            self.daemon = not self.args.foreground
            with self:
                self.llfuse_thread.join(timeout=None)
        except Exception as e:
            self.logger.exception('arv-mount: exception during mount: %s', e)
            exit(getattr(e, 'errno', 1))
        exit(0)

    def _llfuse_main(self):
        try:
            llfuse.main()
        except:
            llfuse.close(unmount=False)
            raise
        llfuse.close()
