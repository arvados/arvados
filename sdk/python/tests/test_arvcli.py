# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from unittest import mock
import pytest
from arvados.commands import arvcli


def patch_argv(*args):
    return mock.patch("sys.argv", ["arvcli.py"] + list(args))


@patch_argv("-h")
def test_global_option_help():
    parser = arvcli.ArvCLIArgumentParser()
    with pytest.raises(SystemExit) as exit_status:
        parser.parse_known_args()
    assert exit_status.value.code == 0


@patch_argv("-h", "keep", "ls")
def test_global_option_help_followed_by_subcommand():
    """When called as arvcli.py -h [subcommand], the subcommand is ignored,
    the -h option is consumed by the parser, and the help message is printed,
    followed by normal exit.
    """
    parser = arvcli.ArvCLIArgumentParser()
    with pytest.raises(SystemExit) as exit_status:
        parser.parse_known_args()
    assert exit_status.value.code == 0


@patch_argv("-s", "--format=yaml")
def test_global_conflict_options():
    parser = arvcli.ArvCLIArgumentParser()
    with pytest.raises(SystemExit) as exit_status:
        parser.parse_known_args()
    assert exit_status.value.code != 0


# Pass-through (sub)commands and their corresponding 'entry point' functions.
PASSTHROUGH_CMD_FUNCS = [("keep ls", "arvados.commands.ls.main"),
                         ("keep get", "arvados.commands.get.main"),
                         ("keep put", "arvados.commands.put.main"),
                         ("keep docker", "arvados.commands.keepdocker.main"),
                         ("ws", "arvados.commands.ws.main"),
                         ("copy", "arvados.commands.arv_copy.main")]


@pytest.mark.parametrize("subcommand,main_fcn_name", PASSTHROUGH_CMD_FUNCS)
def test_passthrough_commands_args(subcommand, main_fcn_name):
    """Test that arbitrary argv ('[...] arvcli.py subcommand --foo bar') to
    arvcli.py gets passed to the underlying subcommand; i.e. the passed-through
    subcommand's entry function gets called with ["--foo", "bar"].
    """
    with patch_argv(*(subcommand.split() + ["--foo", "bar"])):
        with mock.patch(main_fcn_name) as s:
            with pytest.raises(SystemExit):
                arvcli.dispatch()
            s.assert_called_with(["--foo", "bar"])


@pytest.mark.parametrize("subcommand,main_fcn_name", PASSTHROUGH_CMD_FUNCS)
def test_passthrough_commands_help(subcommand, main_fcn_name):
    """Test that the -h flag to a subcommand (as opposed to the main command)
    gets passed to the underlying script rather than consumed by the main arg
    parser.
    """
    with patch_argv(*(subcommand.split() + ["-h"])):
        with mock.patch(main_fcn_name) as s:
            with pytest.raises(SystemExit):
                arvcli.dispatch()
            s.assert_called_with(["-h"])
