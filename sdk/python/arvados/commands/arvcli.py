# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

"""Main executable for Arvados CLI SDK, the `arv` command.

This script implements the `arv` command's argument parser. The `arv` command
is meant to be invoked in the following manner:

$ arv [--flags] subcommand|resource [...options]

where `--flags` are common CLI options for the various subcommands.

The `ArvCLIArgumentParser` class, specializing the standard Python
`argparse.ArgumentParser`, provides the support for this CLI usage.
"""


import abc
import argparse
from contextlib import AbstractContextManager
import functools
import importlib
import json
import os
import re
import shlex
import shutil
import subprocess
import sys
from tempfile import NamedTemporaryFile
from typing import Any, Mapping, NoReturn, TextIO
import arvados
import arvados.commands._util as cmd_util
from ruamel.yaml import YAML, YAMLError
yaml = YAML(typ="safe", pure=True)
yaml.default_flow_style = False


class _ArgTypes:
    """Private namespace class for JSON-related CLI argument types."""

    @staticmethod
    def group_uuid(text: str) -> str:
        """Validate an Arvados group UUID as the value of a CLI argument (an
        Arvados project being a type of group).
        """
        if arvados.util.group_uuid_pattern.fullmatch(text):
            return text
        raise argparse.ArgumentTypeError(
            f"Invalid UUID for Arvados project or group: {text}"
        )

    @staticmethod
    def _validate_type(obj_type, obj):
        if isinstance(obj, obj_type):
            return obj
        # No details to raise; caller handles error messaging with pretty_name.
        raise ValueError()

    json_array = cmd_util.JSONStringArgument(
        validator=functools.partial(_validate_type, list),
        pretty_name="JSON array"
    )

    json_object = cmd_util.JSONStringArgument(
        validator=functools.partial(_validate_type, dict),
        pretty_name="JSON object"
    )

    json_filter = cmd_util.JSONArgument(
        validator=cmd_util.validate_filters,
        pretty_name="Arvados API filter"
    )

    json_body = cmd_util.JSONArgument(
        validator=json_object.post_validator,
        pretty_name="JSON request body object"
    )


class _ArgUtil:
    """Private namespace class for helpful functions (static methods) that
    processes the discovery document for the purpose of CLI parser generation.
    """
    @staticmethod
    def singularize_resource(plural: str) -> str:
        """Returns the singular form of a resource term in the original
        plural.
        """
        match plural:
            case "vocabularies":
                return "vocabulary"
            case "sys":
                return "sys"
            case _:
                return plural.removesuffix("s")

    @staticmethod
    def parameter_key_to_argument_name(parameter_key: str) -> str:
        """Convert a parameter key in the discovery document to CLI parameter
        form, for example, `--foo-bar`.

        Arguments:

        * parameter_key: str -- Parameter key in the form as they appear in the
          discovery document, typically like `foo_bar`.
        """
        return "--" + parameter_key.replace("_", "-")

    @staticmethod
    def get_method_options(method_schema):
        """Generate command-line options, in the form of "-f/--foo", from the
        parameters as defined by the API method schema in the discovery
        document.

        For each key "foo_bar" in the "parameters" field of the method schema,
        command-line options are created according to its definition as
        follows.

        If the parameter type is "boolean", a pair of options "--no-foo-bar"
        and "--foo-bar" are created, with opposite meaning.

        If the parameter type is "integer", the CLI input will be interpreted
        as a Python int.

        All other parameter types are parsed as Python str.

        The short form of each option will also be created, by taking the first
        letter of the long form, except when that letter is already used, in
        which case the second letter will be used, and so on. For example,
        "--foo-bar" will have short form "-f", unless "-f" is already used for
        another option, in which case "-o" will be used, etc.

        The "negative" form of boolean options ("--no-foo-bar") will not have
        separate short forms of their own.

        This  generator yields tuples in the form of `(names, kwargs)`, where
        `names` is a one- or two-element tuple and `kwargs` is a dict, suitable
        to be passed as
        `argparse.ArgumentParser.add_argument(*names, **kwargs)`.

        Arguments:

        * method_schema: dict --- Dict object from the parsed discover document
          that defines a method.
        """
        parameters_schema = method_schema.get("parameters", {}).copy()
        # If the method comes with the "request" field, add another parameter
        # based on the sole key in the "properties" dict of that field
        request_schema = method_schema.get("request")
        if request_schema is not None and request_schema.get("properties"):
            for parameter_key in request_schema["properties"].keys():
                parameters_schema[parameter_key] = {
                    "type": "request",  # special value for request parameter
                    "required": request_schema.get("required"),
                    "description": (
                        f"Either a string representing {parameter_key} as JSON"
                        f" or a filename from which to read {parameter_key}"
                        " JSON (use '-' to read from stdin)."
                    )
                }
        argument_key_abbrevs = set("h")  # prevent conflict with "help"
        for parameter_key, parameter_dict in parameters_schema.items():
            parameter_kwargs = {
                "required": parameter_dict.get("required", False)
            }
            parameter_kwargs["help"] = parameter_dict.get("description", "")
            if parameter_kwargs["required"]:
                parameter_kwargs["help"] += " This option must be specified."
            # The "type" member refers to one of the JSON values types, out of
            # string/integer/array/object/boolean.
            # NOTE: Currently, enum-like value choices are not implemented, as
            # the enum values cannot be directly inferred from the discover
            # doc.
            argument_key = _ArgUtil.parameter_key_to_argument_name(
                parameter_key
            )
            for argument_short_key in argument_key:
                if (
                    argument_short_key.isalpha()
                    and argument_short_key not in argument_key_abbrevs
                ):
                    argument_key_abbrevs.add(argument_short_key)
                    break
            else:
                # If the letters of the full argument name are exhausted, fall
                # back to not using a short argument, indicated by the special
                # value None:
                argument_short_key = None
            default = parameter_dict.get("default")
            if default is not None and parameter_dict.get("type") != "boolean":
                parameter_kwargs["help"] += f" Default: {default}."
            match parameter_dict.get("type"):
                case "boolean":
                    # Using the 'action="store_true" (or "store_false")'
                    # mechanism results in flag-like action rather than an
                    # option that takes a true or false value. For each bool
                    # flag "--foo", also generate an additional "negative"
                    # version "--no-foo".
                    neg_argument_key = _ArgUtil.parameter_key_to_argument_name(
                        f"no_{parameter_key}"
                    )
                    neg_parameter_kwargs = {}
                    neg_parameter_kwargs["action"] = "store_false"
                    neg_parameter_kwargs["required"] = False
                    neg_parameter_kwargs["dest"] = parameter_key
                    neg_parameter_kwargs["default"] = json.loads(
                        default if default is not None else "null"
                    )
                    yield (neg_argument_key,), neg_parameter_kwargs

                    parameter_kwargs["action"] = "store_true"
                    parameter_kwargs["dest"] = parameter_key
                    parameter_kwargs["default"] = (
                        neg_parameter_kwargs["default"]
                    )
                case "integer":
                    parameter_kwargs["type"] = int
                    parameter_kwargs["metavar"] = "N"
                case "array":
                    # The filters parameter is only used with "getter" methods
                    # that doesn't send a request body (which is exclusive to
                    # "creator"/"updater" methods). This means it's generally
                    # safe to use the "json_filter" type converter which can
                    # read from the stdin; it wouldn't conflict with the
                    # request body parameter which can also read the stdin.
                    if parameter_key == "filters":
                        parameter_kwargs["type"] = _ArgTypes.json_filter
                        parameter_kwargs["metavar"] = "{JSON,FILE,-}"
                        parameter_kwargs["help"] += (
                            " This can be a filename from which to read"
                            " JSON (use '-' to read from stdin)."
                        )
                    else:
                        parameter_kwargs["type"] = _ArgTypes.json_array
                        parameter_kwargs["metavar"] = "JSON_ARRAY"
                case "object":
                    parameter_kwargs["type"] = _ArgTypes.json_object
                    parameter_kwargs["metavar"] = "JSON_OBJECT"
                case "request":
                    parameter_kwargs["dest"] = "body"
                    parameter_kwargs["type"] = _ArgTypes.json_body
                    parameter_kwargs["metavar"] = "{JSON,FILE,-}"
                case _:
                    parameter_kwargs["type"] = str
                    parameter_kwargs["metavar"] = "STR"
            if argument_short_key is None:
                yield (argument_key,), parameter_kwargs
            else:
                yield (
                    (f"-{argument_short_key}", argument_key), parameter_kwargs
                )


class ObjectEditingProcessBase(AbstractContextManager, abc.ABC):
    """Base class represending a process (in the generic sense, rather than
    "a Unix/Linux process") of editing an Arvados object with an external
    editor on a temporary file.

    The methods `serialize(self, obj, file)` and `deserialize(self, file)` are
    abstract methods meant to be overridden. `serialize()` should write to the
    open `file` (any file-like object), and `deserialize()` should return an
    object loaded from the file.

    When initialized, no external file has been created. To do so, enter it as
    a context manager.

    Upon entering the context, the temporary file will be opened and written to
    with proper initial content if necessary. Upon leaving, the temporary file
    will be closed and cleaned-up (this normally means the file will be gone
    permanently).

    Attributes:

    * tmp_file: Optional[tempfile.NamedTemporaryFile] --- Temporary file to be
      edited.
    * prefix: Optional[str] --- Prefix of temporary filename if provided.
    * suffix: Optional[str] --- Suffix of temporary filename if provided. This
      can be a filename extension with the leading dot/period character `.`,
      useful for hinting the external editor with syntax highlighting.
    * base_command: list[str] --- Command-line argument list for invoking the
      external editor program. See `get_editor_cmdline()` for more.
    """
    _tmpfile_extension = None

    def __init__(self, initial_object=None, uuid=None, file_extension=None):
        """Arguments:

        * initial_object: Optional[Any] --- Initial object to be serialized and
          written to the temporary file before the editor process is run. If
          not provided, the file will be opened empty in the editor.
        * uuid: Optional[str] --- Arvados object UUID to be used as the prefix
          of the temporary file's basename. If not provided, the initial
          object's `uuid` field will be used if available; otherwise, a
          platform-dependent prefix will be chosen automatically. This UUID as
          part of the filename is for information only, and it may be displayed
          in the editor's UI. Its value has no bearing on the actual object
          being edited.
        * file_extension: Optional[str] --- Filename extension (without leading
          dot) of the temporary file, e.g. "json" or "yml". This information
          may be used by the editor to provide syntax highlighting, automatic
          indentation, completion, etc.
        """
        self.initial_object = initial_object

        if uuid:
            self.prefix = f"{uuid}-"
        elif (
            isinstance(initial_object, Mapping)
            and (obj_uuid := initial_object.get("uuid"))
        ):
            self.prefix = f"{obj_uuid}-"
        else:
            self.prefix = None

        ext = self._tmpfile_extension or file_extension
        self.suffix = f".{ext}" if ext else None

        self.tmp_file = None
        self.run_result = None
        self.base_command = self.get_editor_cmdline()

    @staticmethod
    def get_editor_cmdline() -> list[str]:
        """Returns a partial command-line argument list that begins with the
        external editor program. The precedence is the $VISUAL environment
        variable, followed by $EDITOR; and if both are missing, then `nano` if
        it exists in the $PATH; and finally the hard-coded value `vi` no matter
        the command exists or not.
        """
        if cmd_str := (os.environ.get("VISUAL") or os.environ.get("EDITOR")):
            cmd = shlex.split(cmd_str)
        elif cmd_str := shutil.which("nano"):
            cmd = [cmd_str]
        else:
            cmd = ["vi"]
        return cmd

    @abc.abstractmethod
    def serialize(self, obj: Any, file: TextIO) -> None:
        """Abstract method for serializing any object `obj` to the file-like
        object `file` as text.
        """

    @abc.abstractmethod
    def deserialize(self, file: TextIO) -> Any:
        """Abstract method for loading from the file-like object `file` as
        text. Returns the object deserialized from the text content.
        """

    def check_tmp_file(self):
        """Perform a basic sanity check for the temp file being usable."""
        if self.tmp_file is None or self.tmp_file.closed:
            raise RuntimeError("Temporary file is not available")

    def dump(self, obj: Any) -> None:
        """Overwrite the temporary file with the serialized object `obj`."""
        self.check_tmp_file()
        # The following should not be done while the child process is pending.
        self.tmp_file.seek(0)
        self.tmp_file.truncate()
        self.serialize(obj, self.tmp_file)
        self.tmp_file.flush()

    def load(self) -> Any:
        """Read the temporary file from the beginning. Returns the deserialized
        object, or None if the file is empty or only whitespace.
        """
        self.check_tmp_file()
        # Snoop the file to see if it consists of only whitespace characters
        # (including empty lines); if so, return the special value None.
        with open(self.tmp_file.name, "r") as fdup:
            if not fdup.read().strip():
                return None

        self.tmp_file.seek(0)
        return self.deserialize(self.tmp_file)

    def edit(self) -> None:
        """Run external editor and wait for it to finish."""
        self.check_tmp_file()
        self.run_result = subprocess.run(
            self.base_command + [self.tmp_file.name],
            check=False
        )  # Wait for child.

    def __enter__(self):
        self.tmp_file = NamedTemporaryFile(
            mode="w+", prefix=self.prefix, suffix=self.suffix
        )
        if self.initial_object is not None:
            self.dump(self.initial_object)
        return self

    def __exit__(self, exc_type, exc_value, traceback):
        self.tmp_file.close()


class EditingContentError(ValueError):
    """Exception that indicates the content provided by the user via the editor
    is invalid for the specific format.
    """
    def __init__(
        self,
        path=None, line=0, column=0,
        file_type=None,
        original_exception=None,
    ):
        self.path = path
        self.line = line
        self.column = column
        self.file_type = file_type
        self.original_exception = original_exception

    def __str__(self):
        msg = (
            f"Error: invalid input file [type {self.file_type or 'unknown'}]:"
            f" {self.path}:{self.line}:{self.column}"
        )
        if (
            self.original_exception
            and (orig_msg := str(self.original_exception))
        ):
            msg += f":\n{orig_msg}"
        return msg


class JSONEditingProcess(ObjectEditingProcessBase):
    """Subclass of editing process tuned for JSON files."""
    _tmpfile_extension = "json"
    input_error_type = functools.partial(
        EditingContentError, file_type="JSON"
    )

    def __init__(self, *args, indent: int = 1, **kwargs):
        """Arguments:

        * indent: int --- Number of spaces for each indentation level in the
          JSON file. Default: 1.
        """
        super().__init__(*args, **kwargs)
        self.indent = indent

    def serialize(self, obj: Mapping[str, Any], file: TextIO) -> None:
        return json.dump(obj, file, indent=self.indent)

    def deserialize(self, file: TextIO) -> Mapping[str, Any]:
        path = getattr(file, "name", "<unknown path>")
        try:
            obj = json.load(file)
        except json.JSONDecodeError as err:
            line = getattr(err, "lineno", 0)
            column = getattr(err, "colno", 0)
            raise self.input_error_type(
                path=path, line=line, column=column,
                original_exception=err
            )
        if not isinstance(obj, Mapping):
            raise self.input_error_type(
                path=path,
                # NOTE: the "official" term for the curly-braced entity is
                # "object" (see
                # https://datatracker.ietf.org/doc/html/rfc7159.html,
                # https://www.json.org/json-en.html)
                # but that word may be a little confusing, so we elaborate a
                # bit in the user-facing message.
                original_exception=ValueError(
                    "JSON input does not define an object"
                    " (i.e., 'mapping' or 'dict'),"
                    " hence cannot be a valid Arvados object"
                )
            )
        return obj


class YAMLEditingProcess(ObjectEditingProcessBase):
    """Subclass of editing process tuned for YAML files."""
    _tmpfile_extension = "yml"
    input_error_type = functools.partial(
        EditingContentError, file_type="YAML"
    )

    def serialize(self, obj: Mapping[str, Any], file: TextIO) -> None:
        return yaml.dump(obj, file)

    def deserialize(self, file: TextIO) -> Mapping[str, Any]:
        path = getattr(file, "name", "<unknown path>")
        try:
            obj = yaml.load(file)
        except YAMLError as err:
            if problem_mark := getattr(err, "problem_mark", None):
                line = getattr(problem_mark, "line", 0)
                column = getattr(problem_mark, "column", 0)
            else:
                line = 0
                column = 0
            raise self.input_error_type(
                path=path, line=line, column=column,
                original_exception=err
            )
        if not isinstance(obj, Mapping):
            raise self.input_error_type(
                path=path,
                # NOTE: YAML spec uses the term "mapping",
                # https://yaml.org/spec/1.2.2/#mapping
                original_exception=ValueError(
                    "YAML input does not define a mapping,"
                    " hence cannot be a valid Arvados object"
                )
            )
        return obj


class ArvCLIArgumentParser(argparse.ArgumentParser):
    """Argument parser for `arv` commands.
    """
    global_args = frozenset((
        "dry_run",
        "verbose",
        "format",
        "subcommand",
        "method"
    ))
    external_command_modules = {
        "keep ls": "arvados.commands.ls",
        "keep get": "arvados.commands.get",
        "keep put": "arvados.commands.put",
        "keep docker": "arvados.commands.keepdocker",
        "ws": "arvados.commands.ws",
        "copy": "arvados.commands.arv_copy"
    }

    def __init__(self, resource_dictionary, **kwargs):
        """Arguments:

        * resource dictionary: dict --- Dict containing the resources defined
          in the discovery document; can be obtained as the
          `_resourceDesc["resources"]` attribute of an Arvados API client
          object.
        """
        super().__init__(description="Arvados command line client", **kwargs)
        # Common flags to the main command.
        self.add_argument("-n", "--dry-run", action="store_true",
                          help="Don't actually do anything")
        self.add_argument("-v", "--verbose", action="store_true",
                          help="Print some things on stderr")
        # Default output format is JSON, while "-s" or "--short" can be
        # used as a shorthand for "--format=uuid". If both are specified, the
        # last one takes effect.
        self.add_argument(
            "-f", "--format",
            choices=["json", "yaml", "uuid"],
            default="json",
            help="Set output format"
        )
        self.add_argument(
            "-s", "--short",
            dest="format",
            action="store_const", const="uuid",
            help="Return only UUIDs (equivalent to --format=uuid)"
        )

        subparsers = self.add_subparsers(
            dest="subcommand",
            help="Subcommands",
            required=True,
            parser_class=functools.partial(
                argparse.ArgumentParser,
                add_help=False,
            )
        )

        keep_parser = subparsers.add_parser("keep")
        keep_parser.add_argument(
            "method",
            choices=["ls", "get", "put", "docker"]
        )

        ws_parser = subparsers.add_parser("ws")
        copy_parser = subparsers.add_parser("copy")

        self.subparsers = subparsers
        self.resource_dictionary = resource_dictionary
        self._subparser_index = {}
        self._subcommand_to_resource = {}

        self.add_resource_subcommands()

        if "sys" in self._subparser_index:
            self._subparser_index["sy"] = self._subparser_index["sys"]
        if "sys" in self._subcommand_to_resource:
            self._subcommand_to_resource["sy"] = (
                self._subcommand_to_resource["sys"]
            )

        # Only those resources that support a "create" method can be valid
        # for the "create" subcommand.
        creatable_targets = set()
        for cli_name, resource in self._subcommand_to_resource.items():
            resource_schema = self.resource_dictionary[resource]
            if "create" in resource_schema.get("methods", {}):
                creatable_targets.add(cli_name)
        create_parser = subparsers.add_parser(
            "create", help="Create Arvados object using external editor"
        )
        create_parser.add_argument(
            "target_resource",
            choices=sorted(creatable_targets),
            metavar="RESOURCE",
            help="Type of the resource to be created"
        )
        create_parser.add_argument(
            "--project-uuid", "-p",
            type=_ArgTypes.group_uuid,
            metavar="UUID",
            help="UUID of the project in which to create the resource"
        )

    def add_resource_subcommands(self):
        """Add resources as subcommands, their associated methods as
        sub-subcommands, and the parameters associated with each method.
        """
        for resource, resource_schema in self.resource_dictionary.items():
            subcommand = _ArgUtil.singularize_resource(resource)
            self._subcommand_to_resource[subcommand] = resource
            resource_subparser = self.subparsers.add_parser(
                subcommand,
                # For backward compatibility with legacy Ruby CLI client.
                aliases=["sy"] if subcommand == "sys" else []
            )
            self._subparser_index[subcommand] = resource_subparser
            methods_dict = resource_schema.get("methods")
            if methods_dict:
                # Create a collection of "sub-subparsers" under the resource
                # subparser for the methods.
                method_subparsers = resource_subparser.add_subparsers(
                    title="Methods",
                    dest="method",
                    parser_class=argparse.ArgumentParser,
                    help=f"Methods for subcommand {subcommand}"
                )
                for method, method_schema in methods_dict.items():
                    # Add each specific method as a (sub-)subparser with its
                    # associated parameters.
                    method_parser = method_subparsers.add_parser(
                        method,
                        help=method_schema.get("description")
                    )
                    for parameter_names, kwargs in _ArgUtil.get_method_options(
                            method_schema
                    ):
                        method_parser.add_argument(*parameter_names, **kwargs)


def _handle_external_command(module_name: str, args: list[str]) -> NoReturn:
    """Import the external module for the subcommand, call the module's
    `main()` function with given arguments, and exit with the main function's
    return value as the exit status code.
    """
    external_mod = importlib.import_module(module_name)
    sys.exit(external_mod.main(args))


def _call_resource_method(method_obj, method_args: Mapping, fmt: str) -> int:
    """Given the API resource method object and parameters, create an API
    request, execute it (do the call), and print the response of the API server
    in the given format.

    Returns 0 if successful, or 1 if any errors are encountered.
    """
    method_call = method_obj(**method_args)
    try:
        result = method_call.execute()
    except arvados.errors.ApiError as err:
        # NOTE: This is not exactly the same output as that generated by
        # the Ruby 'arv' command upon error.
        msg = str(err)
        request_id = method_call.headers.get("X-Request-Id")
        if request_id and not re.search(
            rf"\b{re.escape(request_id)}\b", msg
        ):
            msg += f" ({request_id})"
        print(f"Error: {msg}", file=sys.stderr)
        return 1

    match fmt:
        case "json":
            json.dump(result, sys.stdout, indent=1)
            print()
        case "yaml":
            yaml.dump(result, sys.stdout)
        case "uuid":
            if (
                    result.get("kind", "").endswith("List")
                    and result.get("items")
            ):
                for item in result["items"]:
                    # The received items may have the "uuid" field filtered out
                    # by the "--select" parameter. The ruby "arv" command
                    # simply outputs blank lines, which is not desirable.
                    obj_uuid = item.get("uuid")
                    if obj_uuid is None:
                        print(
                            (
                                "Error: at least one item in response did not"
                                " include a uuid. The full response was:"
                            ),
                            json.dumps(result, indent=1),
                            sep="\n",
                            file=sys.stderr
                        )
                        return 1
                    print(item["uuid"])
            else:
                obj_uuid = result.get("uuid")
                if obj_uuid is None:
                    print(
                        "Error: response did not include a uuid:",
                        json.dumps(result, indent=1),
                        sep="\n",
                        file=sys.stderr
                    )
                    return 1
                print(obj_uuid)
    return 0


def _handle_resource_method(api_client, resource, args) -> NoReturn:
    """Prepare API request by resource name and the already-parsed arguments,
    send the request, and analyze & print out the result.
    """
    arv_resource = getattr(api_client, resource)()
    arv_method = getattr(arv_resource, args.method)
    method_args = {
        k: v
        for k, v in vars(args).items()
        if k not in ArvCLIArgumentParser.global_args
    }

    sys.exit(_call_resource_method(arv_method, method_args, args.format))


def _handle_external_editor_command(api_client, parser, args) -> NoReturn:
    if args.format == "uuid":
        parser.error(
            "Error: --format=uuid or -s option is not supported for"
            " creating/editing Arvados objects with external editor. Please"
            " choose --format=json (default) or --format=yaml."
        )  # Exits with status 2.
    # Refuse to run when we're not in an interactive session. Some editors may
    # be unwilling to quit even when not attached to a terminal (e.g., vim),
    # which would have caused us to wait without making progress. Others (e.g.,
    # nano) may quit immediately with non-zero code, but editor exit codes are
    # flaky and not well-documented or standardized (see
    # https://stackoverflow.com/a/46678151,
    # https://unix.stackexchange.com/a/293461), and we can't rely on them.
    if not sys.stdin.isatty():
        print(
            "Error: 'create'/'edit' subcommands can only run interactively"
            " when input/output are a terminal.",
            file=sys.stderr
        )
        sys.exit(1)

    obj_stub = {"owner_uuid": args.project_uuid} if args.project_uuid else {}
    if args.format == "json":
        editing = JSONEditingProcess(initial_object=obj_stub)
    else:
        editing = YAMLEditingProcess(initial_object=obj_stub)

    with editing:
        api_call_status = None
        while api_call_status is None:
            try:
                editing.edit()
            except OSError as err:
                cmd_str = shlex.join(
                    editing.base_command + [editing.tmp_file.name]
                )
                print(
                    f"Error: failed to execute editor `{cmd_str}`: {err}",
                    file=sys.stderr
                )
                sys.exit(1)

            try:
                edited_obj = editing.load()
            except EditingContentError as err:
                # Invalid input from editor; emit error message and let the
                # user try again.
                print(str(err), file=sys.stderr)
                while (wants_retry := _ask_reedit()) is None:
                    pass
                if wants_retry:
                    # NOTE: Back to the start of the editing loop!
                    continue
                sys.exit(1)  # User won't retry; exit with failure.
            if not edited_obj:
                print(
                    "notice: input is empty; exiting without changes",
                    file=sys.stderr
                )
                sys.exit(0)

            resource = parser._subcommand_to_resource[args.target_resource]
            arv_resource = getattr(api_client, resource)()
            # TODO: This only handles "create" for now.
            api_call_status = _call_resource_method(
                arv_resource.create, {"body": edited_obj}, args.format
            )
            if api_call_status != 0:
                # If the API request failed, try editing again if the user so
                # desires.
                wants_retry = None
                while wants_retry is None:
                    wants_retry = _ask_reedit()
                    if wants_retry:
                        # Editing loop to be restarted; clear last API call
                        # status.
                        api_call_status = None
                        continue
            # End of the editing loop.
        sys.exit(api_call_status)
    # End of the NoReturn function.


def _ask_reedit() -> bool | None:
    """Ask the user if they'd like to continue editing. Returns True for "yes"
    (default, applies also when the user types in a blank newline), False for
    "no", and None for any other answer.
    """
    # Put the prompt to the stderr rather than the stdout because we would like
    # to keep the stdout clean for API server output, which makes testing
    # simpler, too. Note that if we are ever to import `readline`, which we're
    # not doing now, this customized prompting behavior might break cursor
    # positioning and would have to be revisited.
    print(
        "Edit and try again? ([Y]es/no) ", end="", file=sys.stderr, flush=True
    )
    match input().strip().lower():
        case "" | "y" | "ye" | "yes":
            return True
        case "n" | "no":
            return False
        case _:
            return None


def dispatch(arguments=None):
    api_client = arvados.api("v1")
    cmd_parser = ArvCLIArgumentParser(api_client._resourceDesc["resources"])
    args, remaining_args = cmd_parser.parse_known_args(arguments)

    # There's always args.subcommand if we reach here, because "subcommand" is
    # required by the parser. But "method" may be absent, as is in the case of
    # external commands like "ws" or "copy".
    method = getattr(args, "method", "")

    # Are we calling an external command?
    ext_module = cmd_parser.external_command_modules.get(
        f"{args.subcommand} {method}" if method else args.subcommand
    )
    if ext_module is not None:
        _handle_external_command(ext_module, remaining_args)  # Exits.

    # Are we doing an API resource call?
    resource = cmd_parser._subcommand_to_resource.get(args.subcommand)
    if resource is not None:
        # This is to work around an issue with nested subparsers being unable
        # to show subcommand-level help (while help generation for the
        # leafmost, method-level subparser works as expected). For example,
        # "arvcli.py resouce method -h" will be handled by the leafmost parser
        # first and the code will not reach here if that is the CLI given.
        # However, "arvcli.py resource -h" is handled manually here.
        help_wanted = "-h" in remaining_args or "--help" in remaining_args
        if not method or help_wanted:
            subparser = cmd_parser._subparser_index[args.subcommand]
            subparser.print_help(
                file=(sys.stdout if help_wanted else sys.stderr)
            )
            sys.exit(0 if help_wanted else 2)
        # Any further remaining args indicate either malformed or unrecognized
        # global args (e.g. "arvcli.py --bad-arg resource method") or undefined
        # parameters to a valid resouce-method combination.
        elif remaining_args:
            print(
                "Error: unrecognized command-line arguments:",
                ", ".join(remaining_args),
                file=sys.stderr
            )
            print(
                f"Try: {sys.argv[0]} --help",
                f"     {sys.argv[0]} {args.subcommand} {method} --help",
                sep="\n",
                file=sys.stderr
            )
            sys.exit(2)
        else:
            _handle_resource_method(api_client, resource, args)  # Exits.

    # Are we starting an external editor program?
    if args.subcommand in ("create", "edit"):
        _handle_external_editor_command(api_client, cmd_parser, args)  # Exits.

    # NOTE: The code immediately below is not reachable.
    raise RuntimeError("Unexpected arguments: {arguments!r}")


if __name__ == "__main__":
    dispatch()
