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
from collections.abc import Container, Mapping
from contextlib import AbstractContextManager
from dataclasses import dataclass
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
from typing import Any, NoReturn, TextIO
import arvados
import arvados.commands._util as cmd_util
from googleapiclient import discovery
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
        # In theory this is a special case of "UUIDInfo" but we mostly need it
        # for the nicer error message.
        if arvados.util.group_uuid_pattern.fullmatch(text):
            return text
        raise argparse.ArgumentTypeError(
            f"Invalid UUID for Arvados project or group: {text}"
        )

    @dataclass(frozen=True)
    class UUIDInfo:
        """'Interpreted' Arvados UUID object with resource type info."""
        uuid: str
        resource_type: str  # value in CamelCase
        rtype_lower: str  # value in snake_case

        @classmethod
        def parse(
            cls, type_map: Mapping[str, str], text: str
        ) -> "UUIDInfo":  # self-typing support comes in Python 3.11.
            """Parse the UUID argument `text`. If accepted, returns an
            `UUIDInfo` instance whose `uuid` attribute is the input UUID
            unchanged and the `resource_type` attribute is the type of Arvados
            object (in CamelCase), as determined by the input parameter
            `type_map`, and `rtype_lower` is the alternative form of
            resource type in snake_case.
            """
            if not arvados.util.uuid_pattern.fullmatch(text):
                raise argparse.ArgumentTypeError(
                    f"Invalid Arvados object UUID: {text}"
                )
            type_code = text.split("-")[1]
            if type_code not in type_map:
                available_types = ", ".join(sorted(
                    f"{k} ({v})" for k, v in type_map.items()
                ))
                raise argparse.ArgumentTypeError(
                    f"Invalid object type code {type_code!r} in Arvados"
                    f" object UUID {text}: valid type codes are"
                    f" {available_types}"
                )
            type_key = type_map[type_code]
            return cls(text, type_key, _ArgUtil.camel_case_to_snake(type_key))

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
    def camel_case_to_snake(text: str) -> str:
        """Simple converter of CamelCase text to so-called 'snake_case' (lower
        case with underscore). Works if there's no consecutive upper-case
        letters such as "API".
        """
        return text[:1].lower() + "".join(
            f"_{c.lower()}" if c.isupper() else c for c in text[1:]
        )

    @staticmethod
    def get_method_options(
        method_schema: Mapping[str, Any],
        ignored_parameters: Container[str] = ()
    ):
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

        * method_schema: Mapping[str, Any] --- Dict object from the parsed
          discover document that defines a method.
        * ignored_parameters: Container[str] --- If provided, the parameters
          that are in `ignored_parameters` will not be processed.
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
            if parameter_key in ignored_parameters:
                continue
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

    @staticmethod
    def make_uuid_to_resource_map(schemas: dict[str, dict]) -> dict[str, str]:
        """Returns a mapping of Arvados object UUID prefixes to resource names
        (in the schema-key, CamelCase form, e.g. "ContainerRequest") based on
        the input "schemas" portion of the discovery document.
        """
        result = {}
        for schema in schemas.values():
            if (
                    (prefix := schema.get("uuidPrefix"))
                    and (key := schema.get("id"))
            ):
                result[prefix] = key
        return result


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

    def __init__(self, initial_object=None, prefix=None, file_extension=None):
        """Arguments:

        * initial_object: Optional[Any] --- Initial object to be serialized and
          written to the temporary file before the editor process is run. If
          not provided, the file will be opened empty in the editor.
        * prefix: Optional[str] --- String to be used as the prefix
          of the temporary file's basename, followed by a hyphen (`-`)
          character that will be added automatically. If not provided, the
          initial object's `uuid` field will be used if available; otherwise, a
          platform-dependent prefix will be chosen automatically. A UUID as
          part of the filename is for information only, and it may be displayed
          in the editor's UI.
        * file_extension: Optional[str] --- Filename extension (without leading
          dot) of the temporary file, e.g. "json" or "yml". This information
          may be used by the editor to provide syntax highlighting, automatic
          indentation, completion, etc.
        """
        self.initial_object = initial_object

        if prefix:
            self.prefix = f"{prefix}-"
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
        self.tmp_file.truncate(0)
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

    def edit(self) -> subprocess.CompletedProcess:
        """Run external editor and wait for it to finish."""
        self.check_tmp_file()
        return subprocess.run(
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
                original_exception=ValueError(
                    f"JSON input has type '{type(obj).__name__}',"
                    " not a valid Arvados object"
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
                original_exception=ValueError(
                    f"YAML input has type '{type(obj).__name__}',"
                    " not a valid Arvados object"
                )
            )
        return obj


class FullHelpOnErrorArgumentParser(argparse.ArgumentParser):
    """Argument parser subclass that customizes the `error()` method.

    Intended to be used as a base to a parser with complex subparsers, to print
    more-useful information when a required subcommand is missing.
    """
    def error(self, message, with_help=True):
        if with_help:
            self.print_help(sys.stderr)
            print(file=sys.stderr)
        # NOTE: self.prog is to be overridden by child class
        print(f"{self.prog}: error: {message}", file=sys.stderr)
        sys.exit(2)


class ArvCLIArgumentParser(FullHelpOnErrorArgumentParser):
    """Argument parser for `arv` commands.
    """
    prog = "arv"
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

    def __init__(self, discovery_document: dict[str, str | dict], **kwargs):
        """Arguments:

        * discovery_document: dict --- Dict containing the parsed API discovery
          document; can be obtained as the `_rootDesc` attribute of an
          Arvados API client object.
        """
        super().__init__(
            description="Arvados command line client",
            prog=self.prog,
            **kwargs
        )
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
            type=str.lower,
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
            description="Available subcommands and resources",
            required=True,
            metavar="subcommand",  # Suppress huge list in help message.
            parser_class=FullHelpOnErrorArgumentParser
        )

        keep_methods = ["ls", "get", "put", "docker"]
        keep_parser = subparsers.add_parser(
            "keep", help="Arvados Keep client", add_help=False,
            epilog=f"available methods: {', '.join(keep_methods)}"
        )
        keep_parser.add_argument(
            "method",
            metavar="METHOD",
            choices=keep_methods
        )

        subparsers.add_parser(
            "ws", help="Arvados WebSocket client", add_help=False
        )
        subparsers.add_parser(
            "copy",
            help=(
                "Copy collection, workflow, or project between Arvados"
                " instances"
            ),
            add_help=False
        )

        self.subparsers = subparsers
        self.discovery_document = discovery_document
        # Work around googleapiclient's mutation of _rootDesc/_resourceDesc
        # dicts when a resource is created. For instance, currently (as of
        # 2026-06-03) "configs.get" resource-method's parameters get mutated at
        # init time of the API client object (as a side-effect of getting the
        # default storage classes for its KeepClient object).
        self._ignored_parameters = frozenset(
            discovery_document.get("parameters", {}).keys()
            | discovery.STACK_QUERY_PARAMETERS
        )
        self.resource_schemas = discovery_document.get("resources", {})
        self._subparser_index = {}
        self._subcommand_to_resource = {}

        self.add_resource_subcommands()

        if "sys" in self._subcommand_to_resource:
            self._subcommand_to_resource["sy"] = (
                self._subcommand_to_resource["sys"]
            )

        self.add_editor_subcommands()

    def add_resource_subcommands(self):
        """Add resources as subcommands, their associated methods as
        sub-subcommands, and the parameters associated with each method.
        """
        for resource, resource_schema in self.resource_schemas.items():
            subcommand = _ArgUtil.singularize_resource(resource)
            self._subcommand_to_resource[subcommand] = resource
            # XXX: Below, "{resource}" can be a "word" like
            # "api_client_authorizations" that doesn't read well; consider
            # retrieving more natural-language-flavored description from the
            # "schema" portion of the discovery doc?
            subcommand_summary = f"Resource subcommand for {resource}"
            resource_subparser = self.subparsers.add_parser(
                subcommand,
                help=subcommand_summary,
                description=subcommand_summary,
                # For backward compatibility with legacy Ruby CLI client.
                aliases=["sy"] if subcommand == "sys" else []
            )
            methods_dict = resource_schema.get("methods")
            if methods_dict:
                # Create a collection of "sub-subparsers" under the resource
                # subparser for the methods.
                method_subparsers = resource_subparser.add_subparsers(
                    title="methods",
                    dest="method",
                    parser_class=FullHelpOnErrorArgumentParser,
                    required=True,
                    help=f"Methods for subcommand '{subcommand}'"
                )
                for method, method_schema in methods_dict.items():
                    # Add each specific method as a (sub-)subparser with its
                    # associated parameters.
                    method_summary = method_schema.get("description")
                    method_parser = method_subparsers.add_parser(
                        method,
                        description=method_summary,
                        help=method_summary
                    )
                    for parameter_names, kwargs in _ArgUtil.get_method_options(
                        method_schema,
                        ignored_parameters=self._ignored_parameters
                    ):
                        method_parser.add_argument(*parameter_names, **kwargs)

    def add_editor_subcommands(self):
        """Add the "create" and "edit" subcommands."""
        # Only those resources that support a "create" method can be valid
        # for the "create" subcommand.
        creatable_targets = set()
        for cli_name, resource in self._subcommand_to_resource.items():
            if "create" in self.resource_schemas[resource].get("methods", {}):
                creatable_targets.add(cli_name)
        create_parser = self.subparsers.add_parser(
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

        self.uuid_type_map = _ArgUtil.make_uuid_to_resource_map(
            self.discovery_document.get("schemas", {})
        )

        edit_parser = self.subparsers.add_parser(
            "edit", help="Edit Arvados object using external editor"
        )
        edit_parser.add_argument(
            "uuid_info", help="UUID of the object to be edited",
            metavar="UUID",
            type=functools.partial(
                _ArgTypes.UUIDInfo.parse, self.uuid_type_map
            )
        )
        edit_parser.add_argument(
            "fields", nargs="*",
            type=str.lower,  # "type" applies to individual items.
            help="Fields to be edited (case-insensitive)"
        )


def _handle_external_command(module_name: str, args: list[str]) -> NoReturn:
    """Import the external module for the subcommand, call the module's
    `main()` function with given arguments, and exit with the main function's
    return value as the exit status code.
    """
    external_mod = importlib.import_module(module_name)
    sys.exit(external_mod.main(args))


def _format_api_error_msg(err: arvados.errors.ApiError, method_call) -> str:
    """Format API error, with the request-id from the HttpRequest object
    `method_call` if 1) it is available and 2) the original message itself
    doesn't already contain the request-id.
    """
    # NOTE: This is not exactly the same output as that generated by the Ruby
    # 'arv' command upon error.
    msg = str(err)
    request_id = method_call.headers.get("X-Request-Id")
    if request_id and not re.search(rf"\b{re.escape(request_id)}\b", msg):
        msg += f" ({request_id})"
    return msg


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
        msg = _format_api_error_msg(err, method_call)
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


def _select_fields(
    src: Mapping[str, Any], fields: Container[str]
) -> Mapping[str, Any]:
    """Select the items of input dict `src` whose keys are in `fields` (a
    subset of the keys of `src`, or, if `fields` is empty, return the input
    `src` unmodified.
    """
    return {k: src[k] for k in fields} or src


def _prepare_initial_object_to_edit(
    api_client, parser, args
) -> tuple[int, dict | str]:
    """Obtain the initial object for the "arv edit" subcommand, based on the
    commandline args and the current API client & CLI parser instances. If the
    "fields" argument (`args.fields`) is provided, only those fields that are
    specified are returned.

    Return value:

    * status: int --- Status code indicating the result of operation:
      * 0: Success; the second return value is the initial object as a dict.
      * 1: API error; the second return value is the error message.
      * 2: Invalid input; the second return value is the error message.
    * value: dict | str --- Returned, filtered initial object `dict` in case of
      success, or an error-message string in case of failure.
    """
    resource_name = args.uuid_info.rtype_lower

    # Filter the fields for any invalid keys of the particular resource.
    valid_fields = parser.discovery_document.get("schemas", {})[
        args.uuid_info.resource_type
    ]["properties"]
    # Sets doesn't remember insertion order, but we want to put invalid keys in
    # the order given by the user for consistency, so we do a dedup with dict.
    invalid_fields = {f: None for f in args.fields if f not in valid_fields}
    if invalid_fields:
        return 2, (
            f"invalid fields for resource {resource_name!r}:"
            f" {', '.join(map(repr, invalid_fields))}"
        )

    method_call = getattr(
        api_client, parser._subcommand_to_resource[resource_name]
    )().get(uuid=args.uuid_info.uuid)
    try:
        arv_obj = method_call.execute()
    except arvados.errors.ApiError as err:
        return 1, _format_api_error_msg(err, method_call)

    return 0, _select_fields(arv_obj, args.fields)


def _handle_external_editor_command(api_client, parser, args) -> NoReturn:
    """Handle the subcommands "create" or "edit"."""
    if args.subcommand == "create":
        init_obj = {
            "owner_uuid": args.project_uuid
        } if args.project_uuid else {}
        # Tempfile name resembling "new-collection-{random}.{json|yml}".
        prefix = f"new-{args.target_resource}"
    else:
        status, obj_or_msg = _prepare_initial_object_to_edit(
            api_client, parser, args
        )
        if status != 0:
            print("Error: {obj_or_msg}", file=sys.stderr)
            sys.exit(status)
        # Tempfile name resembling
        # "collection-clstr-4zz18-{15chars}-{random}.{json|yml}".
        init_obj = obj_or_msg
        prefix = f"{args.uuid_info.rtype_lower}-{args.uuid_info.uuid}"

    match args.format:
        case "json":
            editing_class = JSONEditingProcess
        case "yaml":
            editing_class = YAMLEditingProcess
        case _:
            raise RuntimeError(
                f"Error: unexpected value for format option: {args.format}"
            )

    with editing_class(initial_object=init_obj, prefix=prefix) as editing:
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

            if args.subcommand == "create":
                resource = parser._subcommand_to_resource[args.target_resource]
            else:
                resource = parser._subcommand_to_resource[
                    args.uuid_info.rtype_lower
                ]

            arv_resource = getattr(api_client, resource)()

            if args.subcommand == "create":
                api_call_status = _call_resource_method(
                    arv_resource.create, {"body": edited_obj}, args.format
                )
            else:
                obj_delta = {
                    k: v
                    for k, v in edited_obj.items()
                    if k not in init_obj or v != init_obj[k]
                }
                if not obj_delta:
                    print(
                        "notice: object is unchanged; did not update",
                        file=sys.stderr
                    )
                    sys.exit(0)
                api_call_status = _call_resource_method(
                    arv_resource.update,
                    {"uuid": args.uuid_info.uuid, "body": obj_delta},
                    args.format
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
    cmd_parser = ArvCLIArgumentParser(api_client._rootDesc)
    args, remaining_args = cmd_parser.parse_known_args(arguments)

    # There's always args.subcommand if we reach here, because "subcommand" is
    # required by the parser. But "method" may be absent, as is in the case of
    # external commands like "ws" or "copy".
    method = getattr(args, "method", "")
    command_key = f"{args.subcommand} {method}" if method else args.subcommand

    # Are we calling an external command?
    ext_module = cmd_parser.external_command_modules.get(command_key)
    if ext_module is not None:
        sys.argv[0] = f"arv {command_key}"
        _handle_external_command(ext_module, remaining_args)  # Exits.

    # Are we doing an API resource call?
    resource = cmd_parser._subcommand_to_resource.get(args.subcommand)
    if resource is not None:
        # Any further remaining args indicate either malformed or unrecognized
        # global args (e.g. "arvcli.py --bad-arg resource method") or undefined
        # parameters to a valid resouce-method combination.
        if remaining_args:
            cmd_parser.error(
                f"unrecognized arguments: {', '.join(remaining_args)}\n"
                f"Try: {cmd_parser.prog} --help\n"
                f"     {cmd_parser.prog} {command_key} --help",
                with_help=False
            )  # Exits with status 2.
        _handle_resource_method(api_client, resource, args)  # Exits.

    # Are we starting an external editor program?
    if args.subcommand in ("create", "edit"):
        if args.format == "uuid":
            cmd_parser.error(
                "--format=uuid or -s option is not supported when creating or"
                " editing Arvados objects with external editor. Please"
                " choose --format=json (default) or --format=yaml."
            )  # Exits with status 2.
        _handle_external_editor_command(api_client, cmd_parser, args)  # Exits.

    # NOTE: The code immediately below is not reachable.
    raise RuntimeError("Unexpected arguments: {arguments!r}")


if __name__ == "__main__":
    dispatch()
