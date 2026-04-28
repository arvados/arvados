# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import argparse
from collections import namedtuple
import re
import io
import json
from unittest import mock
from pathlib import Path
from typing import TextIO
import ciso8601
import pytest
from ruamel.yaml import YAML
yaml = YAML(typ="safe", pure=True)
yaml.default_flow_style = False

import arvados
from arvados.commands import arvcli
from . import run_test_server


COLLECTION_UUID_PATTERN = re.compile(r"^[0-9a-z]{5}-4zz18-[0-9a-z]{15}$")


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

    def test_group_uuid_validation(self):
        assert all(
            arvcli._ArgTypes.group_uuid(fx["uuid"])
            for fx in run_test_server.fixture("groups").values()
        )
        with pytest.raises(argparse.ArgumentTypeError):
            arvcli._ArgTypes.group_uuid("zzzzz-j7d0g-123456789")


@pytest.fixture
def run_arvcli(capsys):

    def the_run(cli_args):
        with pytest.raises(SystemExit) as exc:
            arvcli.dispatch(cli_args)
        captured = capsys.readouterr()
        return exc.value.code, captured.out, captured.err

    return the_run


@pytest.mark.parametrize(
    "invalid_value",
    ("foo", '"foo"', '{"foo": null}', '1.0', 'false', 'true', 'null')
)
def test_cli_can_intercept_invalid_json_subtype(invalid_value, run_arvcli):
    # --scopes takes JSON array
    cli = ["api_client_authorization", "create_system_auth", "--scopes"]
    cli.append(invalid_value)
    exit_code, out, err = run_arvcli(cli)
    assert exit_code == 2
    assert "not valid JSON array" in err


class TestSameFlagInTwoPlaces:
    def test_s_flag(self, run_arvcli):
        # As "global" parameter, "-s" is for "--short" (display UUID[s] in
        # output only).  As parameter to the resource method, the second "-s"
        # is for "--select", which limits the output attributes.
        # For a counterexample, see the function
        # test_uuid_output_with_list_items_having_no_uuid()
        exit_code, out, err = run_arvcli(
            ["-s", "collection", "list", "-s", '["uuid"]']
        )
        assert exit_code == 0
        lines = out.splitlines()
        assert any(lines)
        assert all(COLLECTION_UUID_PATTERN.match(line) for line in lines)

    def test_f_flag(self, run_arvcli):
        # As "global" parameter, "-f" is for "--format", which takes one arg
        # value. As parameter to the resource method, "-f" is for "--filters"
        active_user = run_test_server.fixture("users")["active"]["uuid"]
        exit_code, out, err = run_arvcli([
            "-f", "uuid",
            "collection", "list",
            "-f", json.dumps([["owner_uuid", "=", active_user]])
        ])
        assert exit_code == 0
        assert not err
        lines = out.splitlines()
        assert any(lines)
        assert all(COLLECTION_UUID_PATTERN.match(line) for line in lines)


class TestCommonMethods:
    """Basic tests that sample the common methods -- get, list, create, update,
    delete -- with different resources and global CLI options. Basic sanity
    checks are performed from the results of these calls.
    """

    def test_container_request_get_yaml(self, run_arvcli):
        fix = run_test_server.fixture("container_requests")["queued"]
        exit_code, out, err = run_arvcli([
            "--format", "yaml",
            "container_request", "get",
            "--uuid", fix["uuid"]
        ])
        assert exit_code == 0
        result = yaml.load(out)
        attrs = (
            "name", "container_image", "owner_uuid", "command", "output_path"
        )
        for attr in attrs:
            assert result[attr] == fix[attr]

    def test_group_list_format_json_common_args(self, run_arvcli):
        exit_code, out, err = run_arvcli([
            "--format", "json",
            "group", "list",
            "--offset", "1",
            "--limit", "10",
            "--filters", json.dumps([["group_class", "=", "project"]]),
            "--count=none",
            "--order", '["modified_at desc"]',
            "--select", '["uuid", "name", "modified_at"]'
        ])
        assert exit_code == 0
        result = json.loads(out)
        assert result["kind"] == "arvados#groupList"

    @pytest.mark.usefixtures("reset_test_server_db")
    def test_link_create_format_uuid(self, run_arvcli):
        me = run_test_server.fixture("users")["active"]
        project = run_test_server.fixture("groups")["private"]
        exit_code, out, err = run_arvcli([
            "--format", "uuid",
            "link", "create",
            "--link", json.dumps({
                "link_class": "star",
                "owner_uuid": me["uuid"],
                "tail_uuid": me["uuid"],
                "head_uuid": project["uuid"]
            })
        ])
        assert exit_code == 0
        assert re.match(r"^[0-9a-z]{5}-o0j2j-[0-9a-z]{15}$", out)

    @pytest.mark.usefixtures("reset_test_server_db")
    def test_user_update(self, run_arvcli):
        me = run_test_server.fixture("users")["active"]
        my_email = "no-reply@test.example"
        exit_code, out, err = run_arvcli([
            "user", "update",
            "--uuid", me["uuid"],
            "--user", json.dumps({"email": my_email})
        ])
        assert exit_code == 0
        result = json.loads(out)
        assert result["uuid"] == me["uuid"]
        assert result["email"] == my_email

    @pytest.mark.usefixtures("reset_test_server_db")
    def test_authorized_key_delete(self, run_arvcli):
        key = run_test_server.fixture("authorized_keys")["active"]
        exit_code, out, err = run_arvcli([
            "authorized_key", "delete",
            "--uuid", key["uuid"]
        ])
        assert exit_code == 0
        # Same key is gone.
        exit_code, out, err = run_arvcli([
            "authorized_key", "get",
            "--uuid", key["uuid"]
        ])
        assert exit_code == 1
        assert "404 not found" in err.lower()


def _no_extra_spaces_at_end(text: str) -> bool:
    # Text ends in newline but without extraneous whitespace characters.
    return re.search(r"(\A|\S)\n\Z", text) is not None


class TestRequestBodyWithCollectionCreateCMD:
    md5_empty = "d41d8cd98f00b204e9800998ecf8427e"
    collection_test_name = "empty-test"
    manifest_data = {
        "name": collection_test_name,
        "manifest_text": f". {md5_empty}+0 0:0:empty\n"
    }
    cli = ["collection", "create", "--collection"]

    def setup_method(self):
        run_test_server.reset()

    @classmethod
    def teardown_class(self):
        run_test_server.reset()

    def test_request_body_missing(self, run_arvcli):
        exit_code, out, err = run_arvcli(self.cli)
        assert exit_code == 2
        assert err
        assert not out

    @mock.patch("sys.stdin", new_callable=io.StringIO)
    def test_request_body_stdin_valid_json(self, mock_stdin, run_arvcli):
        json.dump(self.manifest_data, mock_stdin)
        mock_stdin.seek(0)
        exit_code, out, err = run_arvcli(self.cli + ["-"])
        assert exit_code == 0
        assert not err
        actual = json.loads(out)
        assert actual["kind"] == "arvados#collection"
        assert actual["name"] == self.manifest_data["name"]
        assert COLLECTION_UUID_PATTERN.match(actual["uuid"])
        assert _no_extra_spaces_at_end(out)

    def test_request_body_file_valid_json_out_yaml(self, tmp_path, run_arvcli):
        f = tmp_path / "body.json"
        f.write_text(json.dumps(self.manifest_data))
        exit_code, out, err = run_arvcli(
            ["--format", "yaml"] + self.cli + [f"{f!s}"]
        )
        assert exit_code == 0
        assert not err
        actual = yaml.load(out)
        assert actual["kind"] == "arvados#collection"
        assert actual["name"] == self.manifest_data["name"]
        assert COLLECTION_UUID_PATTERN.match(actual["uuid"])
        assert _no_extra_spaces_at_end(out)

    def test_request_body_file_valid_json_out_short(self, tmp_path, run_arvcli):
        f = tmp_path / "body.json"
        f.write_text(json.dumps(self.manifest_data))
        exit_code, out, err = run_arvcli(["-s"] + self.cli + [f"{f!s}"])
        assert exit_code == 0
        assert not err
        assert _no_extra_spaces_at_end(out)
        assert COLLECTION_UUID_PATTERN.match(out.rstrip())

    @mock.patch("sys.stdin", new_callable=io.StringIO)
    def test_replace_files(self, mock_stdin, run_arvcli):
        json.dump(self.manifest_data, mock_stdin)
        mock_stdin.seek(0)
        replace_files = json.dumps({
            "/foo/bar.txt": "manifest_text/empty"
        })
        exit_code, out, err = run_arvcli(
            self.cli + ["-", "--replace-files", replace_files]
        )
        assert exit_code == 0
        assert not err
        actual = json.loads(out)
        assert re.match(
            fr"^\./foo {self.md5_empty}\+0\+A[0-9a-f]{{40}}@[0-9a-f]{{8}} 0:0:bar\.txt\n$",
            actual["manifest_text"]
        )

    def test_invalid_request(self, tmp_path, run_arvcli):
        f = tmp_path / "body.json"
        f.write_text(json.dumps(self.manifest_data))
        # request will be invalid because replace_files does not reference
        # manifest data in body.
        replace_files = json.dumps({"/foo": "current/bar"})
        exit_code, out, err = run_arvcli([
            "collection",
            "create",
            "--collection",
            f"{f!s}",
            "--replace-files",
            replace_files
        ])
        assert exit_code == 1
        assert not out
        assert re.search(r"\breq-[0-9a-z]{20}\b", err)
        assert _no_extra_spaces_at_end(err)


def _parse_simple_stream(manifest: str) -> dict[str, str]:
    """Extract the digest, size, and path from a simple, one-file stream from
    a manifest text, following the format used in the API service test fixture
    data (see services/api/test/fixtures/collections.yml; e.g.
    `manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n"`).
    """
    stream_pattern = re.compile(
        r"(?a)\A(\.(/[^/\s]+)*)"  # stream-name
        r" (?P<digest>[0-9a-f]{32})\+(?P<size>[0-9]+)"  # locator as digest+size
        r" [0-9]+:(?P=size):(?P<filename>[^\s]+)\n\Z"
    )
    m = stream_pattern.match(manifest)
    return m.groupdict() if m is not None else {}


@pytest.mark.usefixtures("reset_test_server_db")
def test_collection_update_with_replace_files(run_arvcli):
    foo_uuid = run_test_server.fixture("collections")["foo_file"]["uuid"]
    bar_pdh = run_test_server.fixture("collections")["bar_file"]["portable_data_hash"]
    bar_manifest = run_test_server.fixture("collections")["bar_file"]["manifest_text"]
    replace = json.dumps({"/bar": f"{bar_pdh}/bar"})

    exit_code, out, err = run_arvcli([
        "collection", "update",
        "--uuid", foo_uuid,
        "--collection", "{}",
        "--replace-files", replace
    ])
    assert exit_code == 0
    result = json.loads(out)
    # Quick and dirty check that the file "bar" is now in the manifest.
    bar_elements = _parse_simple_stream(bar_manifest)
    bar_locator_part = f"{bar_elements['digest']}+{bar_elements['size']}"
    assert bar_locator_part in result["manifest_text"]
    bar_file_part = f":{bar_elements['size']}:{bar_elements['filename']}"
    assert bar_file_part in result["manifest_text"]


def test_uuid_output_with_list_items_having_no_uuid(run_arvcli):
    exit_code, out, err = run_arvcli([
        "--format", "uuid", "collection", "list", "--select", '["name"]',
    ])
    assert exit_code == 1
    assert not out
    assert "did not include a uuid" in err


class TestDefaultValuesForAPICalls:
    resources = arvados.api("v1")._resourceDesc["resources"]

    @classmethod
    def get_default(cls, resource, method, parameter):
        default = cls.resources[resource]["methods"][method]["parameters"][parameter].get("default", "null")
        return json.loads(default)

    def test_no_override_default_parameter_value(self, run_arvcli):
        exit_code, out, err = run_arvcli(["user", "list"])
        assert exit_code == 0
        assert not err
        result = json.loads(out)
        assert result["limit"] == self.get_default("users", "list", "limit")

    def test_override_default_parameter_value(self, run_arvcli):
        limit = 1
        exit_code, out, err = run_arvcli(
            ["user", "list", "--limit", str(limit)]
        )
        assert exit_code == 0
        assert not err
        result = json.loads(out)
        assert result["limit"] == limit


# The "config get" command doesn't take any parameter.
class TestConfigGet:
    def test_config_get(self, run_arvcli):
        exit_code, out, err = run_arvcli(["config", "get"])
        assert exit_code == 0

    def test_config_get_uuid(self, run_arvcli):
        exit_code, out, err = run_arvcli(["--format", "uuid", "config", "get"])
        assert exit_code == 1
        assert not out
        err = io.StringIO(err)
        assert err.readline().rstrip() == "Error: response did not include a uuid:"
        assert json.load(err)


class TestApiClientAuthorizationsResource:
    users = run_test_server.fixture("users")
    auths = run_test_server.fixture("api_client_authorizations")

    @classmethod
    def teardown_class(self):
        run_test_server.reset()

    def assert_same_api_auth(self, fix: dict, result: dict):
        """Compare an API auth fixture as loaded by run_test_server.fixture()
        to a result returned by the API.
        """
        assert fix["uuid"] == result["uuid"]
        assert fix["api_token"] == result["api_token"]
        # Resolve user name in fixture to owner_uuid. "Cheat" by looking up the
        # users fixtures directly.
        owner_uuid = self.users[fix["user"]]["uuid"]
        assert owner_uuid == result["owner_uuid"]
        # Resolve date. The date field in the fixture is timezone-naïve, so we
        # have to coerce away the timezone information for comparability.
        result_expires_at = ciso8601.parse_datetime_as_naive(
            result["expires_at"]
        )
        assert fix["expires_at"] == result_expires_at
        assert fix.get("scopes", ["all"]) == result["scopes"]

    def test_current(self, run_arvcli):
        me = "active"
        run_test_server.authorize_with(me)
        fix = self.auths[me]

        exit_code, out, err = run_arvcli(
            ["api_client_authorization", "current"]
        )

        assert exit_code == 0
        result = json.loads(out)
        self.assert_same_api_auth(fix, result)

    # TODO: investigate possible authorization issue with testing
    # the create_system_auth method.


GEC = arvcli.ObjectEditingProcessBase.get_editor_cmdline


class TestGetEditorCmdline:

    @pytest.fixture
    def installed_nano(self, tmp_path, monkeypatch):
        """Ensure that `nano` is installed by placing an executable named
        "nano" in a temp directory and then set $PATH to that directory. When
        requested, yields the full path of the `nano` executable.
        """
        nano = tmp_path / "nano"
        nano.write_text("#!/bin/sh\nexit 0\n")
        nano.chmod(0o500)
        monkeypatch.setenv("PATH", str(tmp_path))
        yield str(nano)

    @pytest.fixture
    def uninstalled_nano(self, tmp_path, monkeypatch):
        """Ensure that `nano` is not in the $PATH, by setting $PATH to an empty
        temp directory.
        """
        monkeypatch.setenv("PATH", str(tmp_path))
        yield

    def test_env_var(self, monkeypatch):
        monkeypatch.setenv("VISUAL", "foo --bar")
        monkeypatch.setenv("EDITOR", "bar")
        assert GEC() == ["foo", "--bar"]
        monkeypatch.delenv("VISUAL")
        assert GEC() == ["bar"]

    def test_fallback_nano(self, monkeypatch, installed_nano):
        monkeypatch.delenv("VISUAL", raising=False)
        monkeypatch.delenv("EDITOR", raising=False)
        assert GEC() == [installed_nano]

    def test_fallback_no_nano(self, monkeypatch, uninstalled_nano):
        monkeypatch.delenv("VISUAL", raising=False)
        monkeypatch.delenv("EDITOR", raising=False)
        assert GEC() == ["vi"]


@pytest.fixture
def setup_editor_simulator(tmp_path, monkeypatch):
    editor_dir = Path(__file__).parent
    editor_path = editor_dir / "editor_simulator.py"
    monkeypatch.setenv("PATH", str(editor_dir), prepend=":")

    base_dir = tmp_path / "editor_input"
    base_dir.mkdir()
    # Temporary file for editor_simulator.py that gets written each time the
    # editor is "installed"
    edit_source = base_dir / "source"
    # Persistent file that keeps a record of editor input, for each request to
    # this fixture (typically means function scope).
    log = base_dir / "log"
    logf = open(log, "a")

    def editor_fcn(content: str = "", *extra_args):
        with open(edit_source, "w") as s:
            s.write(content)
        logf.write(content)
        logf.write("-----\n")
        editor_cmd = f"{editor_path!s} -i {edit_source!s}"
        if extra_args:
            editor_cmd += f" {' '.join(extra_args)}"
        monkeypatch.setenv("VISUAL", editor_cmd)

    try:
        yield editor_fcn
    finally:
        logf.close()


class PlainStringEditing(arvcli.ObjectEditingProcessBase):
    """'Plain' editing process for which 'serialization'/'deserialization'
    are simply string-writing and reading respectively.
    """
    def serialize(self, obj: str, file: TextIO) -> None:
        file.write(obj)

    def deserialize(self, file: TextIO) -> str:
        return file.read()


class TestObjectEditingProcessBase:
    """Test a minimal concrete derived-class of ObjectEditingProcessBase."""
    def test_basic(self, setup_editor_simulator):
        content = "Hello, world!\n"
        setup_editor_simulator(content)
        with PlainStringEditing() as ed:
            assert ed.tmp_file is not None
            assert Path(ed.tmp_file.name).exists()
            ed.edit()
            assert ed.run_result is not None
            assert ed.load() == content
            # Inside the same context, the ed.edit() method can be called more
            # than once with the desired result.
            content = "foo bar"
            setup_editor_simulator(content)
            ed.edit()
            assert ed.load() == content
        assert not Path(ed.tmp_file.name).exists()

    def test_initial_object(self):
        initial_obj = "init"
        with PlainStringEditing(initial_obj) as ed:
            # Snoop the temp file.
            with open(ed.tmp_file.name, "r") as t:
                filled_content = t.read()
        assert filled_content == initial_obj

    def test_tempfile_name_prefix(self):
        fake_uuid = "foo-bar"
        with PlainStringEditing(uuid=fake_uuid) as ed:
            assert Path(ed.tmp_file.name).stem.startswith(f"{fake_uuid}-")

    def test_tempfile_name_no_empty_prefix(self):
        # It's risky to do negative tests on the tempfile's actual name because
        # it's random in a platform-dependent way. We don't want a widowed
        # hyphen/dash character *created by us* when uuid="" or when
        # initial_object["uuid"] is empty, but a hyphen may as well happen to
        # be the first character generated randomly.
        ed = PlainStringEditing(uuid="")
        assert ed.prefix is None

    def test_tempfile_name_prefix_from_obj_uuid(self):
        uuid = "foo-bar"
        initial_obj = {"uuid": uuid}
        with arvcli.JSONEditingProcess(initial_obj) as ed:
            assert Path(ed.tmp_file.name).stem.startswith(f"{uuid}-")

        uuid_override = "foo-bar-baz"
        with arvcli.JSONEditingProcess(initial_obj, uuid=uuid_override) as ed:
            assert Path(ed.tmp_file.name).stem.startswith(f"{uuid_override}-")

        initial_obj = {}
        ed = arvcli.JSONEditingProcess(initial_obj)
        assert ed.prefix is None

    def test_tempfile_name_suffix(self):
        ext = "dat"
        with PlainStringEditing(file_extension=ext) as ed:
            assert Path(ed.tmp_file.name).suffix == f".{ext}"

    def test_tempfile_name_suffix_no_empty_extension(self):
        # See also the comment for test_tempfile_name_no_empty_prefix().
        ed = PlainStringEditing(file_extension="")
        assert ed.suffix is None

    def test_editor_did_not_edit(self, setup_editor_simulator):
        setup_editor_simulator("", "-a")
        initial_obj = "init"
        with PlainStringEditing(initial_obj) as ed:
            ed.edit()
            assert ed.load() == initial_obj


def test_create_subcommad_s_option(setup_editor_simulator, run_arvcli):
    with mock.patch("subprocess.run") as sr:
        exit_code, out, err = run_arvcli(["-s", "create", "collection"])
    assert exit_code == 2
    assert not sr.called


def yaml_dumps(obj) -> str:
    buf = io.StringIO()
    yaml.dump(obj, stream=buf)
    return buf.getvalue()


EditFormatCase = namedtuple("EditFormatCase", ("format", "dumps", "loads"))


class TestEditingSubcommands:
    def setup_method(self):
        run_test_server.reset()

    @classmethod
    def teardown_class(self):
        run_test_server.reset()

    @pytest.mark.parametrize("format_case", (
        EditFormatCase("json", json.dumps, json.loads),
        EditFormatCase("yaml", yaml_dumps, yaml.load)
    ))
    def test_basic_create(
        self, format_case, setup_editor_simulator, run_arvcli
    ):
        obj = {"name": "a new project", "group_class": "project"}
        setup_editor_simulator(format_case.dumps(obj))

        # Force arvcli to believe that we have a tty.
        with mock.patch("os.isatty", new=lambda _: True):
            exit_code, out, err = run_arvcli(
                ["--format", format_case.format, "create", "group"]
            )

        assert exit_code == 0
        result = format_case.loads(out)
        for k in obj.keys():
            assert obj[k] == result[k]

    def test_create_in_project_yaml(self, setup_editor_simulator, run_arvcli):
        # Simulate editing the temp file with owner_uuid field pre-filled due
        # to the --project-uuid CLI argument. YAML is much easier to setup
        # with our fake editor because simple appending will suffice.
        parent_proj = run_test_server.fixture("groups")["aproject"]
        # The object to be appended to the pre-filled stub in the temp file.
        obj = {"name": "a new sub-project", "group_class": "project"}
        setup_editor_simulator(yaml_dumps(obj), "-a")

        with mock.patch("os.isatty", new=lambda _: True):
            exit_code, out, err = run_arvcli([
                "--format", "yaml",
                "create", "group",
                "--project-uuid", parent_proj["uuid"]
            ])

        assert exit_code == 0
        result = yaml.load(out)
        assert result["owner_uuid"] == parent_proj["uuid"]
        for k in obj.keys():
            assert obj[k] == result[k]
