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


import sys
import argparse
import functools
import json
import arvados
import arvados.commands._util as cmd_util


class _ArgTypes:
    """Private namespace class for JSON-related CLI argument types."""
    @staticmethod
    def _validate_type(obj_type, obj):
        if isinstance(obj, obj_type):
            return obj
        raise ValueError(f"{obj!r} is not of type {obj_type!s}.")

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
            if default is not None:
                parameter_kwargs["default"] = default
                if parameter_dict.get("type") != "boolean":
                    parameter_kwargs["help"] += f" Default: {default!s}."
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
                        parameter_dict.get("default", "null")
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


class ArvCLIArgumentParser(argparse.ArgumentParser):
    """Argument parser for `arv` commands.
    """
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
                add_help=False
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
                    help="Methods for subcommand {}".format(subcommand)
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


def dispatch(arguments=None):
    api_client = arvados.api("v1")
    cmd_parser = ArvCLIArgumentParser(api_client._resourceDesc["resources"])
    args, remaining_args = cmd_parser.parse_known_args(arguments)

    match args.subcommand:
        case "keep":
            match args.method:
                case "ls":
                    from arvados.commands.ls import main
                case "get":
                    from arvados.commands.get import main
                case "put":
                    from arvados.commands.put import main
                case "docker":
                    from arvados.commands.keepdocker import main
        case "ws":
            from arvados.commands.ws import main
        case "copy":
            from arvados.commands.arv_copy import main
        case _:
            # FIXME
            print("Called API resource {!r}, method {!r}".format(
                args.subcommand, args.method
            ))
            for k, v in vars(args).items():
                print("{!r}={!r}".format(k, v))
            help_wanted = "-h" in remaining_args or "--help" in remaining_args
            if args.method is None or help_wanted:
                subparser = cmd_parser._subparser_index.get(args.subcommand)
                if subparser:
                    subparser.print_help()
                sys.exit(0 if help_wanted else 2)
            sys.exit(0)
    status = main(remaining_args)
    sys.exit(status)


if __name__ == "__main__":
    dispatch()
