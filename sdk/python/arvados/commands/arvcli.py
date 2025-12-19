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
import arvados


def singularize_resource(plural: str) -> str:
    """Returns the singular form of a resource term in the original plural."""
    match plural:
        case "vocabularies":
            return "vocabulary"
        case "sys":
            return "sys"
        case _:
            return plural.removesuffix("s")


def parameter_key_to_argument_name(parameter_key: str) -> str:
    """Convert a parameter key in the discovery document to CLI parameter form.

    Arguments:
        * `parameter_key`: Parameter key in the form as they appear in the
          discovery document, typically like `foo_bar`.

    Return value:
        * Parameter in the conventional CLI form, for example, `--foo-bar`.
    """
    return "--" + parameter_key.replace("_", "-")


def parameter_schema_to_argument(parameter_schema):
    """Convert a parameter's schema as specified in the discover document into
    kwargs to add_argument().

    Arguments:
        * `parameter_schema`: dict containing the schema that defines the
          parameter's type and behavior, from the discovery document.

    Return value:
        * A dict that contains the keyword arguments ready to be passed to
          `argparser.ArgumentParser.add_argument()`
    """
    parameter_kwargs = {"required": parameter_schema.get("required")}
    parameter_kwargs["help"] = parameter_schema.get("description")
    # The "type" member refers to one of the JSON values types, out of
    # string/integer/array/object/boolean.
    # NOTE: Arrays and objects are treated as strings for Python
    # argument-parsing purposes, but in the future basic validation can be done
    # at argument-parsing time to ensure they're valid JSON strings of the
    # required type. In addition, the "metavar" may hint at the required JSON
    # type (array/object).
    # NOTE: Currently, enum-like value choices are not implemented, as the enum
    # values cannot be directly inferred from the discover doc.
    match parameter_schema.get("type"):
        case "boolean":
            # Using the 'action="store_true" (or "store_false")' mechanism
            # results in flag-like action rather than an option that takes a
            # true or false value.
            # NOTE: currently there are only boolean parameters with
            # {"default": "false"}; the best way to generate command-line UI
            # for generic case is yet to be determined.
            if "default" in parameter_schema:
                match parameter_schema["default"]:
                    case "true":
                        parameter_kwargs["action"] = "store_false"
                    case "false":
                        parameter_kwargs["action"] = "store_true"
            else:
                parameter_kwargs["type"] = bool
        case "integer":
            parameter_kwargs["type"] = int
            parameter_kwargs["metavar"] = "N"
            if "default" in parameter_schema:
                parameter_kwargs["default"] = int(parameter_schema["default"])
        case _:
            parameter_kwargs["type"] = str
            parameter_kwargs["metavar"] = "STR"
            if "default" in parameter_schema:
                parameter_kwargs["default"] = parameter_schema["default"]
    return parameter_kwargs


class ArvCLIArgumentParser(argparse.ArgumentParser):
    """Argument parser for `arv` commands.
    """
    def __init__(self, resource_dictionary, **kwargs):
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

        self.add_resource_subcommands(resource_dictionary)

    def add_resource_subcommands(self, resource_dictionary):
        """Add resources as subcommands, their associated methods as
        sub-subcommands, and the parameters associated with each method.

        Arguments:
            * `resource dictionary`: Dict containing the resources defined in
            the discovery document; can be obtained as the
            `_resourceDesc["resources"]` attribute of an Arvados API client
            object.
        """
        for resource, resource_schema in resource_dictionary.items():
            subcommand = singularize_resource(resource)
            resource_subparser = self.subparsers.add_parser(
                subcommand,
                # For backward compatibility with legacy Ruby CLI client.
                aliases=["sy"] if subcommand == "sys" else []
            )
            methods_dict = resource_schema.get("methods")
            if methods_dict:
                # Create a collection of "sub-subparsers" under the resource
                # subparser for the methods.
                method_subparsers = resource_subparser.add_subparsers(
                    title="Methods",
                    dest="method",
                    parser_class=argparse.ArgumentParser,
                    help="Methods for subcommand {}".format(subcommand),
                    required=True
                )
                for method, method_schema in methods_dict.items():
                    # Add each specific method as a (sub-)subparser with its
                    # associated parameters.
                    # FIXME: the value of the "description" member doesn't get
                    # displayed when help is requested from the cmdline, e.g.
                    # "arvcli.py user list -h"
                    method_parser = method_subparsers.add_parser(
                        method,
                        help=method_schema.get("description")
                    )
                    for parameter_key, parameter_schema in method_schema.get(
                        "parameters", {}
                    ).items():
                        method_parser.add_argument(
                            parameter_key_to_argument_name(parameter_key),
                            **parameter_schema_to_argument(parameter_schema)
                        )


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
            sys.exit(0)
    status = main(remaining_args)
    sys.exit(status)


if __name__ == "__main__":
    dispatch()
