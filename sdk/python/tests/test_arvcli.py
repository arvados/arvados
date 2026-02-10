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
    assert arvcli._ArgUtil.singularize_resource(plural) == singular


@pytest.mark.parametrize("key,argument_name", (
    ("ensure_unique_name", "--ensure-unique-name"),
    ("filters", "--filters"),
))
def test_parameter_key_to_argument_name(key, argument_name):
    assert arvcli._ArgUtil.parameter_key_to_argument_name(key) == argument_name


def test_parameter_schema_to_argument():
    # Largely based on arvados.container_requests.create, but with a fictitious
    # parameter entry for integer type, another one for required=True, and
    # also with parameter descriptions replaced by brief strings.
    input_method_schema = {
        "parameters": {
            "select": {
                "type": "array",
                "description": "help-select.",
                "required": False,
                "location": "query"
            },
            "ensure_unique_name": {
                "type": "boolean",
                "description": "help-ensure-unique-name.",
                "location": "query",
                "required": False,
                "default": "false"
            },
            "cluster_id": {
                "type": "string",
                "description": "help-cluster-id.",
                "location": "query",
                "required": False
            },
            # Fictitious parameters
            "uuid": {
                "type": "string",
                "description": "help-uuid.",
                "required": True,
                "location": "path"

            },
            "limit": {
                "type": "integer",
                "required": False,
                "default": "100",
                "description": "help-limit.",
                "location": "query"
            }
        },
        "request": {
            "required": True,
            "properties": {
                "container_request": {
                    "$ref": "ContainerRequest"
                }
            }
        }
    }
    output = [
        (
            ("-s", "--select"),
            {
                "type": str,
                "metavar": "STR",
                "help": "help-select.",
                "required": False
            }
        ),
        (
            ("--no-ensure-unique-name",),
            {
                "dest": "ensure_unique_name",
                "action": "store_false",
                "default": False,
                "required": False
            }
        ),
        (
            ("-e", "--ensure-unique-name"),
            {
                "dest": "ensure_unique_name",
                "action": "store_true",
                "help": "help-ensure-unique-name.",
                "required": False,
                "default": False
            }
        ),
        (
            ("-c", "--cluster-id"),
            {
                "type": str,
                "metavar": "STR",
                "help": "help-cluster-id.",
                "required": False
            }
        ),
        # Fictitious parameters
        (
            ("-u", "--uuid"),
            {
                "type": str,
                "metavar": "STR",
                "help": "help-uuid. This option must be specified.",
                "required": True,
            }
        ),
        (
            ("-l", "--limit"),
            {
                "type": int,
                "metavar": "N",
                "default": 100,
                "help": "help-limit.",
                "required": False
            }
        ),
        # Request parameter
        (
            ("-o", "--container-request"),
            {
                "type": str,
                "metavar": "STR",
                "help": "Either a string representing container_request as JSON or a filename from which to read container_request JSON (use '-' to read from stdin). This option must be specified.",
                "required": True
            }
        )
    ]
    assert list(
        arvcli._ArgUtil.get_method_options(input_method_schema)
    ) == output
