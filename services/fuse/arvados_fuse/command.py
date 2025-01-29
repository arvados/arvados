# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

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
            description="Interact with Arvados data through a local filesystem",
        )
        self.add_argument(
            '--version',
            action='version',
            version=u"%s %s" % (sys.argv[0], __version__),
            help="Print version and exit",
        )
        self.add_argument(
            'mountpoint',
            metavar='MOUNT_DIR',
            help="Directory path to mount data",
        )

        mode_group = self.add_argument_group("Mount contents")
        mode = mode_group.add_mutually_exclusive_group()
        mode.add_argument(
            '--all',
            action='store_const',
            const='all',
            dest='mode',
            help="""
Mount a subdirectory for each mode: `home`, `shared`, `by_id`, and `by_tag`
(default if no `--mount-*` options are given)
""",
        )
        mode.add_argument(
            '--custom',
            action='store_const',
            const=None,
            dest='mode',
            help="""
Mount a subdirectory for each mode specified by a `--mount-*` option
(default if any `--mount-*` options are given;
see "Mount custom layout and filtering" section)
""",
        )
        mode.add_argument(
            '--collection',
            metavar='UUID_OR_PDH',
            help="Mount the specified collection",
        )
        mode.add_argument(
            '--home',
            action='store_const',
            const='home',
            dest='mode',
            help="Mount your home project",
        )
        mode.add_argument(
            '--project',
            metavar='UUID',
            help="Mount the specified project",
        )
        mode.add_argument(
            '--shared',
            action='store_const',
            const='shared',
            dest='mode',
            help="Mount a subdirectory for each project shared with you",
        )
        mode.add_argument(
            '--by-id',
            action='store_const',
            const='by_id',
            dest='mode',
            help="""
Mount a magic directory where collections and projects are accessible through
subdirectories named after their UUID or portable data hash
""",
        )
        mode.add_argument(
            '--by-pdh',
            action='store_const',
            const='by_pdh',
            dest='mode',
            help="""
Mount a magic directory where collections are accessible through
subdirectories named after their portable data hash
""",
        )
        mode.add_argument(
            '--by-tag',
            action='store_const',
            const='by_tag',
            dest='mode',
            help="Mount a subdirectory for each tag attached to a collection or project",
        )

        mounts = self.add_argument_group("Mount custom layout and filtering")
        mounts.add_argument(
            '--filters',
            type=arv_cmd.JSONArgument(arv_cmd.validate_filters),
            help="""
Filters to apply to all project, shared, and tag directory contents.
Pass filters as either a JSON string or a path to a JSON file.
The JSON object should be a list of filters in Arvados API list filter syntax.
""",
        )
        mounts.add_argument(
            '--mount-home',
            metavar='PATH',
            action='append',
            default=[],
            help="Make your home project available under the mount at `PATH`",
        )
        mounts.add_argument(
            '--mount-shared',
            metavar='PATH',
            action='append',
            default=[],
            help="Make projects shared with you available under the mount at `PATH`",
        )
        mounts.add_argument(
            '--mount-tmp',
            metavar='PATH',
            action='append',
            default=[],
            help="""
Make a new temporary writable collection available under the mount at `PATH`.
This collection is deleted when the mount is unmounted.
""",
        )
        mounts.add_argument(
            '--mount-by-id',
            metavar='PATH',
            action='append',
            default=[],
            help="""
Make a magic directory available under the mount at `PATH` where collections and
projects are accessible through subdirectories named after their UUID or
portable data hash
""",
        )
        mounts.add_argument(
            '--mount-by-pdh',
            metavar='PATH',
            action='append',
            default=[],
            help="""
Make a magic directory available under the mount at `PATH` where collections
are accessible through subdirectories named after portable data hash
""",
        )
        mounts.add_argument(
            '--mount-by-tag',
            metavar='PATH',
            action='append',
            default=[],
            help="""
Make a subdirectory for each tag attached to a collection or project available
under the mount at `PATH`
""" ,
        )

        perms = self.add_argument_group("Mount access and permissions")
        perms.add_argument(
            '--allow-other',
            action='store_true',
            help="Let other users on this system read mounted data (default false)",
        )
        perms.add_argument(
            '--read-only',
            action='store_false',
            default=False,
            dest='enable_write',
            help="Mounted data cannot be modified from the mount (default)",
        )
        perms.add_argument(
            '--read-write',
            action='store_true',
            default=False,
            dest='enable_write',
            help="Mounted data can be modified from the mount",
        )

        lifecycle = self.add_argument_group("Mount lifecycle management")
        lifecycle.add_argument(
            '--exec',
            nargs=argparse.REMAINDER,
            dest="exec_args",
            help="""
Mount data, run the specified command, then unmount and exit.
`--exec` reads all remaining options as the command to run,
so it must be the last option you specify.
Either end your command arguments (and other options) with a `--` argument,
or specify `--exec` after your mount point.
""",
        )
        lifecycle.add_argument(
            '--foreground',
            action='store_true',
            default=False,
            help="Run mount process in the foreground instead of daemonizing (default false)",
        )
        lifecycle.add_argument(
            '--subtype',
            help="Set mounted filesystem type to `fuse.SUBTYPE` (default is just `fuse`)",
        )
        unmount = lifecycle.add_mutually_exclusive_group()
        unmount.add_argument(
            '--replace',
            action='store_true',
            default=False,
            help="""
If a FUSE mount is already mounted at the given directory,
unmount it before mounting the requested data.
If `--subtype` is specified, unmount only if the mount has that subtype.
WARNING: This command can affect any kind of FUSE mount, not just arv-mount.
""",
        )
        unmount.add_argument(
            '--unmount',
            action='store_true',
            default=False,
            help="""
If a FUSE mount is already mounted at the given directory, unmount it and exit.
If `--subtype` is specified, unmount only if the mount has that subtype.
WARNING: This command can affect any kind of FUSE mount, not just arv-mount.
""",
        )
        unmount.add_argument(
            '--unmount-all',
            action='store_true',
            default=False,
            help="""
Unmount all FUSE mounts at or below the given directory, then exit.
If `--subtype` is specified, unmount only if the mount has that subtype.
WARNING: This command can affect any kind of FUSE mount, not just arv-mount.
""",
        )
        lifecycle.add_argument(
            '--unmount-timeout',
            type=float,
            default=2.0,
            metavar='SECONDS',
            help="""
The number of seconds to wait for a clean unmount after an `--exec` command has
exited (default %(default).01f).
After this time, the mount will be forcefully unmounted.
""",
        )

        reporting = self.add_argument_group("Mount logging and statistics")
        reporting.add_argument(
            '--crunchstat-interval',
            type=float,
            default=0.0,
            metavar='SECONDS',
            help="Write stats to stderr every N seconds (default disabled)",
        )
        reporting.add_argument(
            '--debug',
            action='store_true',
            help="Log debug information",
        )
        reporting.add_argument(
            '--logfile',
            help="Write debug logs and errors to the specified file (default stderr)",
        )

        cache = self.add_argument_group("Mount local cache setup")
        cachetype = cache.add_mutually_exclusive_group()
        cachetype.add_argument(
            '--disk-cache',
            action='store_true',
            default=True,
            dest='disk_cache',
            help="Cache data on the local filesystem (default)",
        )
        cachetype.add_argument(
            '--ram-cache',
            action='store_false',
            default=True,
            dest='disk_cache',
            help="Cache data in memory",
        )
        cache.add_argument(
            '--disk-cache-dir',
            metavar="DIRECTORY",
            help="Set custom filesystem cache location",
        )
        cache.add_argument(
            '--directory-cache',
            type=int,
            default=128*1024*1024,
            metavar='BYTES',
            help="Size of directory data cache in bytes (default 128 MiB)",
        )
        cache.add_argument(
            '--file-cache',
            type=int,
            default=0,
            metavar='BYTES',
            help="""
Size of file data cache in bytes
(default 8 GiB for filesystem cache, 256 MiB for memory cache)
""",
        )

        plumbing = self.add_argument_group("Mount interactions with Arvados and Linux")
        plumbing.add_argument(
            '--disable-event-listening',
            action='store_true',
            dest='disable_event_listening',
            default=False,
            help="Don't subscribe to events on the API server to update mount contents",
        )
        plumbing.add_argument(
            '--encoding',
            default="utf-8",
            help="""
Filesystem character encoding
(default %(default)r; specify a name from the Python codec registry)
""",
        )
        plumbing.add_argument(
            '--storage-classes',
            metavar='CLASSES',
            help="Comma-separated list of storage classes to request for new collections",
        )
        plumbing.add_argument(
            '--refresh-time',
            metavar='SECONDS',
            default=15,
            type=int,
            help="Upper limit on how long mount contents may be out of date with upstream Arvados before being refreshed on next access (default 15 seconds)",
        )
        # This is a hidden argument used by tests.  Normally this
        # value will be extracted from the cluster config, but mocking
        # the cluster config under the presence of multiple threads
        # and processes turned out to be too complicated and brittle.
        plumbing.add_argument(
            '--fsns',
            type=str,
            default=None,
            help=argparse.SUPPRESS)

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
            self._setup_logging()
        except Exception as e:
            self.logger.exception("exception during setup: %s", e)
            exit(1)

        try:
            nofile_limit = resource.getrlimit(resource.RLIMIT_NOFILE)

            minlimit = 10240
            if self.args.file_cache:
                # Adjust the file handle limit so it can meet
                # the desired cache size. Multiply by 8 because the
                # number of 64 MiB cache slots that keepclient
                # allocates is RLIMIT_NOFILE / 8
                minlimit = int((self.args.file_cache/(64*1024*1024)) * 8)

            if nofile_limit[0] < minlimit:
                resource.setrlimit(resource.RLIMIT_NOFILE, (min(minlimit, nofile_limit[1]), nofile_limit[1]))

            if minlimit > nofile_limit[1]:
                self.logger.warning("file handles required to meet --file-cache (%s) exceeds hard file handle limit (%s), cache size will be smaller than requested", minlimit, nofile_limit[1])

        except Exception as e:
            self.logger.warning("unable to adjust file handle limit: %s", e)

        nofile_limit = resource.getrlimit(resource.RLIMIT_NOFILE)
        self.logger.info("file cache capped at %s bytes or less based on available disk (RLIMIT_NOFILE is %s)", ((nofile_limit[0]//8)*64*1024*1024), nofile_limit)

        try:
            self._setup_api()
            self._setup_mount()
        except Exception as e:
            self.logger.exception("exception during setup: %s", e)
            exit(1)

    def __enter__(self):
        if self.args.replace:
            unmount(path=self.args.mountpoint,
                    timeout=self.args.unmount_timeout)
        llfuse.init(self.operations, str(self.args.mountpoint), self._fuse_options())
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
        self.api.keep.block_cache.clear()
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
            # default value of file_cache is 0, this tells KeepBlockCache to
            # choose a default based on whether disk_cache is enabled or not.

            block_cache = arvados.keep.KeepBlockCache(cache_max=self.args.file_cache,
                                                      disk_cache=self.args.disk_cache,
                                                      disk_cache_dir=self.args.disk_cache_dir)

            self.api = arvados.safeapi.ThreadSafeApiCache(
                apiconfig=arvados.config.settings(),
                api_params={
                    'num_retries': self.args.retries,
                },
                keep_params={
                    'block_cache': block_cache,
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
            enable_write=self.args.enable_write,
            fsns=self.args.fsns)

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
        dir_args = [
            llfuse.ROOT_INODE,
            self.operations.inodes,
            self.api,
            self.args.retries,
            self.args.enable_write,
            self.args.filters,
        ]
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
        elif self.args.mode == 'all':
            self.args.mount_by_id = ['by_id']
            self.args.mount_by_tag = ['by_tag']
            self.args.mount_home = ['home']
            self.args.mount_shared = ['shared']
            mount_readme = True

        if dir_class is not None:
            if dir_class in [TagsDirectory, CollectionDirectory]:
                ent = dir_class(*dir_args, poll_time=self.args.refresh_time)
            else:
                ent = dir_class(*dir_args, storage_classes=storage_classes, poll_time=self.args.refresh_time)
            self.operations.inodes.add_entry(ent)
            self.listen_for_events = ent.want_event_subscribe()
            return

        e = self.operations.inodes.add_entry(Directory(
            llfuse.ROOT_INODE,
            self.operations.inodes,
            self.args.enable_write,
            self.args.filters,
        ))
        dir_args[0] = e.inode

        for name in self.args.mount_by_id:
            self._add_mount(e, name, MagicDirectory(*dir_args, pdh_only=False,
                                                    storage_classes=storage_classes,
                                                    poll_time=self.args.refresh_time))
        for name in self.args.mount_by_pdh:
            self._add_mount(e, name, MagicDirectory(*dir_args, pdh_only=True,
                                                    poll_time=self.args.refresh_time))
        for name in self.args.mount_by_tag:
            self._add_mount(e, name, TagsDirectory(*dir_args))
        for name in self.args.mount_home:
            self._add_mount(e, name, ProjectDirectory(*dir_args, project_object=usr,
                                                      storage_classes=storage_classes,
                                                      poll_time=self.args.refresh_time))
        for name in self.args.mount_shared:
            self._add_mount(e, name, SharedDirectory(*dir_args, exclude=usr,
                                                     storage_classes=storage_classes,
                                                     poll_time=self.args.refresh_time))
        for name in self.args.mount_tmp:
            self._add_mount(e, name, TmpCollectionDirectory(*dir_args,
                                                            storage_classes=storage_classes))

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
            llfuse.main(workers=10)
        except:
            llfuse.close(unmount=False)
            raise
        self.operations.begin_shutdown()
        llfuse.close()
