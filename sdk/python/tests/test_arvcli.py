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


PARAMETER_TRANSFORM_TESTS = [
    ({
        "ensure_unique_name": {
            "type": "boolean",
            "description": "If the given name is already used by this owner, adjust the name to ensure uniqueness instead of returning an error.",
            "location": "query",
            "required": False,
            "default": "false"
        }
    }, {
        "--ensure-unique-name": {
            "dest": "ensure_unique_name",
            "action": "store_true",
            "help": "If the given name is already used by this owner, adjust the name to ensure uniqueness instead of returning an error.",
            "required": False,
            "default": False
        },
        "--no-ensure-unique-name": {
            "dest": "ensure_unique_name",
            "action": "store_false",
            "default": False,
            "required": False
        }
    }),
    ({
        "create_system_auth": {
            "type": "array",
            "required": False,
            "default": '["all"]',
            "description": "An array of strings defining the scope of resources this token will be allowed to access. Refer to the [scopes reference][] for details.\n\n[scopes reference]: https://doc.arvados.org/api/tokens.html#scopes\n",
            "location": "query"
        }
    }, {
        "--create-system-auth": {
            "type": str,
            "metavar": "STR",
            "required": False,
            "default": '["all"]',
            "help": "An array of strings defining the scope of resources this token will be allowed to access. Refer to the [scopes reference][] for details.\n\n[scopes reference]: https://doc.arvados.org/api/tokens.html#scopes\n",
        }
    }),
    ({
        "offset": {
            "type": "integer",
            "required": False,
            "default": "0",
            "description": "Return matching objects starting from this index.\nNote that result indexes may change if objects are modified in between a series\nof list calls.\n",
            "location": "query"
        }
    }, {
        "--offset": {
            "type": int,
            "metavar": "N",
            "required": False,
            "default": 0,
            "help": "Return matching objects starting from this index.\nNote that result indexes may change if objects are modified in between a series\nof list calls.\n"
        }
    }),
    ({
        "replace_files": {
            "type": "object",
            "description": "Add, delete, and replace files and directories with new content\nand/or content from other collections. Refer to the\n[replace_files reference][] for details.\n\n[replace_files reference]: https://doc.arvados.org/api/methods/collections.html#replace_files\n\n",
            "required": False,
            "location": "query",
            "properties": {},
            "additionalProperties": {"type": "string"}
        }
    }, {
        "--replace-files": {
            "type": str,
            "metavar": "STR",
            "help": "Add, delete, and replace files and directories with new content\nand/or content from other collections. Refer to the\n[replace_files reference][] for details.\n\n[replace_files reference]: https://doc.arvados.org/api/methods/collections.html#replace_files\n\n",
            "required": False,
        }
    })
]


@pytest.mark.parametrize(
    "input_parameters_schema,output_dict",
    PARAMETER_TRANSFORM_TESTS
)
def test_parameter_schema_to_argument(input_parameters_schema, output_dict):
    assert arvcli.parameters_schema_to_arguments(input_parameters_schema) == output_dict


def test_resource_subcommand_stub_help(capsys):
    with pytest.raises(SystemExit) as e:
        arvcli.dispatch("user list -h".split())
    assert e.value.code == 0
    captured_text = capsys.readouterr()
    assert "Retrieve a UserList." in captured_text.out
