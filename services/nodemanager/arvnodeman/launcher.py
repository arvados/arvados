#!/usr/bin/env python

from __future__ import absolute_import, print_function

import argparse
import logging
import signal
import sys
import time

import daemon
import pykka

from . import config as nmconfig
from .daemon import NodeManagerDaemonActor
from .jobqueue import JobQueueMonitorActor, ServerCalculator
from .nodelist import ArvadosNodeListMonitorActor, CloudNodeListMonitorActor
from .timedcallback import TimedCallBackActor

node_daemon = None

def abort(msg, code=1):
    print("arvados-node-manager: " + msg)
    sys.exit(code)

def parse_cli(args):
    parser = argparse.ArgumentParser(
        prog='arvados-node-manager',
        description="Dynamically allocate Arvados cloud compute nodes")
    parser.add_argument(
        '--foreground', action='store_true', default=False,
        help="Run in the foreground.  Don't daemonize.")
    parser.add_argument(
        '--config', help="Path to configuration file")
    return parser.parse_args(args)

def load_config(path):
    if not path:
        abort("No --config file specified", 2)
    config = nmconfig.NodeManagerConfig()
    try:
        with open(path) as config_file:
            config.readfp(config_file)
    except (IOError, OSError) as error:
        abort("Error reading configuration file {}: {}".format(path, error))
    return config

def setup_logging(path, level, **sublevels):
    handler = logging.FileHandler(path)
    handler.setFormatter(logging.Formatter(
            '%(asctime)s %(name)s[%(process)d] %(levelname)s: %(message)s',
            '%Y-%m-%d %H:%M:%S'))
    root_logger = logging.getLogger()
    root_logger.addHandler(handler)
    root_logger.setLevel(level)
    for logger_name, sublevel in sublevels.iteritems():
        sublogger = logging.getLogger(logger_name)
        sublogger.setLevel(sublevel)

def launch_pollers(config):
    cloud_client = config.new_cloud_client()
    arvados_client = config.new_arvados_client()
    cloud_size_list = config.node_sizes(cloud_client.list_sizes())
    if not cloud_size_list:
        abort("No valid node sizes configured")

    server_calculator = ServerCalculator(
        cloud_size_list,
        config.getint('Daemon', 'min_nodes'),
        config.getint('Daemon', 'max_nodes'))
    poll_time = config.getint('Daemon', 'poll_time')
    max_poll_time = config.getint('Daemon', 'max_poll_time')

    timer = TimedCallBackActor.start(poll_time / 10.0).proxy()
    cloud_node_poller = CloudNodeListMonitorActor.start(
        cloud_client, timer, poll_time, max_poll_time).proxy()
    arvados_node_poller = ArvadosNodeListMonitorActor.start(
        arvados_client, timer, poll_time, max_poll_time).proxy()
    job_queue_poller = JobQueueMonitorActor.start(
        config.new_arvados_client(), timer, server_calculator,
        poll_time, max_poll_time).proxy()
    return timer, cloud_node_poller, arvados_node_poller, job_queue_poller

_caught_signals = {}
def shutdown_signal(signal_code, frame):
    current_count = _caught_signals.get(signal_code, 0)
    _caught_signals[signal_code] = current_count + 1
    if node_daemon is None:
        pykka.ActorRegistry.stop_all()
        sys.exit(-signal_code)
    elif current_count == 0:
        node_daemon.shutdown()
    elif current_count == 1:
        pykka.ActorRegistry.stop_all()
    else:
        sys.exit(-signal_code)

def main(args=None):
    global node_daemon
    args = parse_cli(args)
    config = load_config(args.config)

    if not args.foreground:
        daemon.DaemonContext().open()
    for sigcode in [signal.SIGINT, signal.SIGQUIT, signal.SIGTERM]:
        signal.signal(sigcode, shutdown_signal)

    setup_logging(config.get('Logging', 'file'), **config.log_levels())
    node_setup, node_shutdown, node_update, node_monitor = \
        config.dispatch_classes()
    timer, cloud_node_poller, arvados_node_poller, job_queue_poller = \
        launch_pollers(config)
    cloud_node_updater = node_update.start(config.new_cloud_client).proxy()
    node_daemon = NodeManagerDaemonActor.start(
        job_queue_poller, arvados_node_poller, cloud_node_poller,
        cloud_node_updater, timer,
        config.new_arvados_client, config.new_cloud_client,
        config.shutdown_windows(),
        config.getint('Daemon', 'min_nodes'),
        config.getint('Daemon', 'max_nodes'),
        config.getint('Daemon', 'poll_stale_after'),
        config.getint('Daemon', 'node_stale_after'),
        node_setup, node_shutdown, node_monitor).proxy()

    signal.pause()
    daemon_stopped = node_daemon.actor_ref.actor_stopped.is_set
    while not daemon_stopped():
        time.sleep(1)
    pykka.ActorRegistry.stop_all()


if __name__ == '__main__':
    main()
