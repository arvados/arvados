# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from unittest import mock
import pytest
import argparse
import re
import io
import os
import json
from contextlib import contextmanager
import arvados
from arvados.commands import arvcli
from ruamel.yaml import YAML
yaml = YAML(typ="safe", pure=True)


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


def test_cli_parser_has_singular_plural_mapping():
    api_client = arvados.api("v1")
    cmd_parser = arvcli.ArvCLIArgumentParser(
        api_client._resourceDesc["resources"]
    )
    for resource in cmd_parser.resource_dictionary.keys():
        k = arvcli._ArgUtil.singularize_resource(resource)
        assert cmd_parser._subcommand_to_resource[k] == resource
    assert cmd_parser._subcommand_to_resource["sy"] == cmd_parser._subcommand_to_resource["sys"]


@pytest.mark.parametrize("key,argument_name", (
    ("ensure_unique_name", "--ensure-unique-name"),
    ("filters", "--filters"),
))
def test_parameter_key_to_argument_name(key, argument_name):
    assert arvcli._ArgUtil.parameter_key_to_argument_name(key) == argument_name


def test_get_method_options():
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
            },
            "filters": {
                "type": "array",
                "required": False,
                "description": "help-filters.",
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
                "type": arvcli._ArgTypes.json_array,
                "metavar": "JSON_ARRAY",
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
                "default": "100",
                "help": "help-limit. Default: 100.",
                "required": False
            }
        ),
        (
            # NOTE: IRL, --filters parameter doesn't appear for methods that
            # have the request parameter. This is purely used for testing
            # schema-to-argparser conversion.
            ("-f", "--filters"),
            {
                "type": arvcli._ArgTypes.json_filter,
                "metavar": "{JSON,FILE,-}",
                "help": "help-filters. This can be a filename from which to read JSON (use '-' to read from stdin).",
                "required": False
            }
        ),
        # Request parameter
        (
            ("-o", "--container-request"),
            {
                "dest": "body",
                "type": arvcli._ArgTypes.json_body,
                "metavar": "{JSON,FILE,-}",
                "help": "Either a string representing container_request as JSON or a filename from which to read container_request JSON (use '-' to read from stdin). This option must be specified.",
                "required": True
            }
        )
    ]
    assert list(
        arvcli._ArgUtil.get_method_options(input_method_schema)
    ) == output


@pytest.mark.usefixtures("tmp_path")
class TestArgTypes:
    """Test the private type converter-validators under the arvcli._ArgTypes
    namespace.
    """
    def test_json_array_makes_list(self):
        assert arvcli._ArgTypes.json_array("[]") == []

    def test_json_object_makes_dict(self):
        assert arvcli._ArgTypes.json_object("{}") == {}

    @pytest.mark.parametrize("invalid_input", ("{}", '""', "0", "null"))
    def test_json_array_rejects_non_array(self, invalid_input):
        with pytest.raises(argparse.ArgumentTypeError):
            arvcli._ArgTypes.json_array(invalid_input)

    @pytest.mark.parametrize("invalid_input", ("[]", '""', "0", "null"))
    def test_json_object_rejects_non_object(self, invalid_input):
        with pytest.raises(argparse.ArgumentTypeError):
            arvcli._ArgTypes.json_object(invalid_input)


@pytest.mark.parametrize(
    "invalid_value",
    ("foo", '"foo"', '{"foo": null}', '1.0', 'false', 'true', 'null')
)
def test_cli_can_intercept_invalid_json_subtype(invalid_value, capsys):
    # --scope takes JSON array
    cli = ["api_client_authorization", "create_system_auth", "--scope"]
    cli.append(invalid_value)
    with pytest.raises(SystemExit) as exit_status:
        arvcli.dispatch(cli)
    assert exit_status.value.code == 2
    captured = capsys.readouterr()
    assert "not valid JSON array" in captured.err


@pytest.mark.usefixtures("capsys", "tmp_path")
class TestRequestBodyWithCollectionCreateCMD:
    md5_empty = "d41d8cd98f00b204e9800998ecf8427e"
    collection_test_name = "empty-test"
    manifest_data = {
        "name": collection_test_name,
        "manifest_text": f". {md5_empty}+0 0:0:empty\n"
    }
    collection_uuid_pattern = re.compile(r"^[0-9a-z]{5}-4zz18-[0-9a-z]{15}$")
    cli = ["collection", "create", "--collection"]

    def teardown_method(self):
        # Remove the collection by name after each test method invocation.
        api_client = arvados.api("v1")
        collection_list_obj = api_client.collections().list(
            filters=f'[["name", "=", "{self.collection_test_name}"]]'
        ).execute()
        for item in collection_list_obj.get("items", []):
            item_uuid = item.get("uuid")
            if item_uuid is not None:
                api_client.collections().delete(uuid=item_uuid).execute()

    def test_request_body_missing(self, capsys):
        with pytest.raises(SystemExit) as exit_status:
            arvcli.dispatch(self.cli)
        assert exit_status.value.code == 2
        captured = capsys.readouterr()
        assert captured.err
        assert not captured.out

    @mock.patch("sys.stdin", new_callable=io.StringIO)
    def test_request_body_stdin_valid_json(self, mock_stdin, capsys):
        json.dump(self.manifest_data, mock_stdin)
        mock_stdin.seek(0)
        with pytest.raises(SystemExit) as exit_status:
            arvcli.dispatch(self.cli + ["-"])
        assert exit_status.value.code == 0
        captured = capsys.readouterr()
        assert not captured.err
        actual = json.loads(captured.out)
        assert actual["kind"] == "arvados#collection"
        assert actual["name"] == self.manifest_data["name"]
        assert self.collection_uuid_pattern.match(actual["uuid"])

    def test_request_body_file_valid_json_out_yaml(self, tmp_path, capsys):
        f = tmp_path / "body.json"
        f.write_text(json.dumps(self.manifest_data))
        with pytest.raises(SystemExit) as exit_status:
            arvcli.dispatch(["--format", "yaml"] + self.cli + [f"{f!s}"])
        assert exit_status.value.code == 0
        captured = capsys.readouterr()
        assert not captured.err
        actual = yaml.load(captured.out)
        assert actual["kind"] == "arvados#collection"
        assert actual["name"] == self.manifest_data["name"]
        assert self.collection_uuid_pattern.match(actual["uuid"])

    def test_request_body_file_valid_json_out_short(self, tmp_path, capsys):
        f = tmp_path / "body.json"
        f.write_text(json.dumps(self.manifest_data))
        with pytest.raises(SystemExit) as exit_status:
            arvcli.dispatch(["-s"] + self.cli + [f"{f!s}"])
        assert exit_status.value.code == 0
        captured = capsys.readouterr()
        assert not captured.err
        assert self.collection_uuid_pattern.match(captured.out.rstrip())

    @mock.patch("sys.stdin", new_callable=io.StringIO)
    def test_replace_files(self, mock_stdin, capsys):
        json.dump(self.manifest_data, mock_stdin)
        mock_stdin.seek(0)
        replace_files = json.dumps({
            "/foo/bar.txt": "manifest_text/empty"
        })
        with pytest.raises(SystemExit) as exit_status:
            arvcli.dispatch(self.cli + ["-", "--replace-files", replace_files])
        assert exit_status.value.code == 0
        captured = capsys.readouterr()
        assert not captured.err
        actual = json.loads(captured.out)
        assert re.match(
            fr"^\./foo {self.md5_empty}\+0\+A[0-9a-f]{{40}}@[0-9a-f]{{8}} 0:0:bar\.txt\n$",
            actual["manifest_text"]
        )
