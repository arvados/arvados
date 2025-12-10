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
import os.path
import argparse
import functools
import arvados


def singularize_resource(plural: str) -> str:
    """Return the singular form of a resource term."""
    match plural:
        case "vocabularies":
            return "vocabulary"
        case "sys":
            return "sys"
        case _:
            return plural.removesuffix("s")


def parameter_key_to_argument_name(parameter_key):
    """Convert a parameter key in the discover doc in the form of "foo_bar"
    into the form suitable for use as a CLI argument ("--foo-bar").
    """
    return "--" + parameter_key.replace("_", "-")


def parameter_schema_to_argument(parameter_schema):
    """Convert a parameter's schema as specified in the discover doc into a
    dictionary suitable for passing as the kwargs to add_argument().
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


def make_method_parameters_parser(subcommand, method, method_schema):
    """Create a plain `argparse.ArgumentParser` instance that implements the
    options (i.e. command-line flags in the form of "--foo-bar") for a
    particular resoure- (i.e. subcommand-)method combo, based on the
    method's definition dict object `method_schema` passed as input.
    """
    prog=" ".join([os.path.basename(sys.argv[0]), subcommand, method])
    parameters_parser = argparse.ArgumentParser(
        prog=prog,
        description=method_schema.get("description")
    )
    for parameter_key, parameter_schema in method_schema.get(
        "parameters", {}
    ).items():
        parameters_parser.add_argument(
            parameter_key_to_argument_name(parameter_key),
            **parameter_schema_to_argument(parameter_schema)
        )
    return parameters_parser


class ArvCLIArgumentParser(argparse.ArgumentParser):
    """Argument parser for `arv` commands.
    """
    def __init__(self, **kwargs):
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

    def add_resource_subcommands(self, resource_dictionary):
        """Add resource subcommands based on the resource dictionary object
        passed as argument. The resource dictionary can be obtained as the
        `_resourceDesc["resources"]` attribute of an Arvados API client object.
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
                resource_subparser.add_argument(
                    "method",
                    choices=list(methods_dict.keys())
                )
            resource_subparser.set_defaults(resource_schema=resource_schema)


# NOTE: The two-step parsing emulates the current behavior of the pass-through
# commands, where the detailed parsing of CLI parameters happen in their own
# implementation modules. But, should the functionality of generating a new
# plain parser for parameters, inside the following function, be folded into
# the class `ArvCLIArgumentParser` as a method?
def resource_subcommand_handler_stub(arguments, parsed_options):
    """Stub for handling a resource subcommand.

    For a command-line invocation in the following manner,

    `arvcli.py [--global-options ...] resource method --parameter=value [...]`

    (e.g. arvcli.py user get --uuid=...),

    an argument parser is constructed for the parameters (i.e. those flags
    `--parameter=value`).

    - `arguments`: list of unparsed command-line options that come after the
      resource subcommand and the method positional argument.

    - `parsed_options`: an `argparse.Namespace` instance containing the
      argument parsing result for the global options, the resource
      (subcommand), and the "method" positional argument.
    """
    method = parsed_options.method
    method_schema = parsed_options.resource_schema["methods"][method]
    parser = make_method_parameters_parser(
        parsed_options.subcommand,
        method,
        method_schema
    )
    args = parser.parse_args(arguments)
    return 0


def dispatch(arguments=None):
    cmd_parser = ArvCLIArgumentParser()
    api_client = arvados.api("v1")
    cmd_parser.add_resource_subcommands(api_client._resourceDesc["resources"])
    args, remaining_args = cmd_parser.parse_known_args(arguments)

    needs_global_args = False
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
            main = resource_subcommand_handler_stub
            needs_global_args = True
    # NOTE: The idiosyncratic call signatures may be unified if we alter the
    # main() functions to the pass-through commands to resemble
    #     def main(arguments, ..., **kwargs):
    # so that the additional 'parsed_options=...' parameter is absorbed into
    # the kwargs, and may be even further utilized in the main function body
    # there (currently it's not).
    if needs_global_args:
        status = main(remaining_args, parsed_options=args)
    else:
        status = main(remaining_args)
    sys.exit(status)


if __name__ == "__main__":
    dispatch()
