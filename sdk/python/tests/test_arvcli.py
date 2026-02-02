# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from unittest import mock
import pytest
from arvados.commands import arvcli


def test_global_option_help_followed_by_subcommand():
    """When called as arvcli.py -h [subcommand], the subcommand is ignored,
    the -h option is consumed by the parser, and the help message is printed,
    followed by normal exit.
    """
    parser = arvcli.ArvCLIArgumentParser({})
    with pytest.raises(SystemExit) as exit_status:
        parser.parse_known_args(["-h", "foo"])
    assert exit_status.value.code == 0


def test_no_subcommand():
    parser = arvcli.ArvCLIArgumentParser({})
    with pytest.raises(SystemExit) as exit_status:
        parser.parse_known_args(["-s"])
    assert exit_status.value.code == 2


def test_invalid_subcommand():
    parser = arvcli.ArvCLIArgumentParser({})
    with pytest.raises(SystemExit) as exit_status:
        parser.parse_known_args(["foo"])
    assert exit_status.value.code == 2


# Pass-through (sub)commands and their corresponding 'entry point' functions.
PASSTHROUGH_CMD_FUNCS = [
    ("keep ls", "arvados.commands.ls.main"),
    ("keep get", "arvados.commands.get.main"),
    ("keep put", "arvados.commands.put.main"),
    ("keep docker", "arvados.commands.keepdocker.main"),
    ("ws", "arvados.commands.ws.main"),
    ("copy", "arvados.commands.arv_copy.main")
]


@pytest.mark.parametrize("subcommand,main_fcn_name", PASSTHROUGH_CMD_FUNCS)
def test_passthrough_commands_args(subcommand, main_fcn_name):
    """Test that arbitrary argv ('[...] arvcli.py subcommand --foo bar') to
    arvcli.py gets passed to the underlying subcommand; i.e. the passed-through
    subcommand's entry function gets called with ["--foo", "bar"].
    """
    with mock.patch(main_fcn_name) as s:
        with pytest.raises(SystemExit):
            arvcli.dispatch([*subcommand.split(), "--foo", "bar"])
        s.assert_called_with(["--foo", "bar"])


@pytest.mark.parametrize("subcommand,main_fcn_name", PASSTHROUGH_CMD_FUNCS)
def test_passthrough_commands_help(subcommand, main_fcn_name):
    """Test that the -h flag to a subcommand (as opposed to the main command)
    gets passed to the underlying script rather than consumed by the main arg
    parser.
    """
    with mock.patch(main_fcn_name) as s:
        with pytest.raises(SystemExit):
            arvcli.dispatch([*subcommand.split(), "-h"])
        s.assert_called_with(["-h"])


@pytest.mark.parametrize("plural,singular", (
    ("container_requests", "container_request"),
    ("vocabularies", "vocabulary"),
    ("sys", "sys"),
    ("Foos", "Foo"),  # generic nonce word that ends in "-s"
    ("foo", "foo")  # already singular in form
))
def test_singularizer(plural, singular):
    assert arvcli.singularize_resource(plural) == singular


@pytest.mark.parametrize("key,argument_name", (
    ("ensure_unique_name", "--ensure-unique-name"),
    ("filters", "--filters"),
))
def test_parameter_key_to_argument_name(key, argument_name):
    assert arvcli.parameter_key_to_argument_name(key) == argument_name


def test_parameter_schema_to_argument():
    input_parameters_schema = {
        "ensure_unique_name": {
            "type": "boolean",
            "description": "foo.",
            "location": "query",
            "required": False,
            "default": "false"
        },
        "create_system_auth": {
            "type": "array",
            "required": False,
            "default": '["all"]',
            "description": "bar.",
            "location": "query"
        },
        "offset": {
            "type": "integer",
            "required": False,
            "default": "0",
            "description": "baz.",
            "location": "query"
        },
        "replace_files": {
            "type": "object",
            "description": "quux",
            "required": False,
            "location": "query",
            "properties": {},
            "additionalProperties": {"type": "string"}
        }
    }
    output = [
        (
            "--no-ensure-unique-name",
            {
                "dest": "ensure_unique_name",
                "action": "store_false",
                "default": False,
                "required": False
            }
        ),
        (
            "--ensure-unique-name",
            {
                "dest": "ensure_unique_name",
                "action": "store_true",
                "help": "foo.",
                "required": False,
                "default": False
            }
        ),
        (
            "--create-system-auth",
            {
                "type": str,
                "metavar": "STR",
                "required": False,
                "default": '["all"]',
                "help": "bar."
            }
        ),
        (
            "--offset",
            {
                "type": int,
                "metavar": "N",
                "required": False,
                "default": 0,
                "help": "baz."
            }
        ),
        (
            "--replace-files",
            {
                "type": str,
                "metavar": "STR",
                "help": "quux",
                "required": False,
            }
        )
    ]
    assert list(
        arvcli.parameters_schema_to_arguments(input_parameters_schema)
    ) == output
