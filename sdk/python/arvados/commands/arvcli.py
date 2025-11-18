# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

"""Main executable for Arvados CLI SDK, the ``arv`` command.

This script implements the ``arv`` command's argument parser. The ``arv``
command is meant to be invoked in the following manner:

$ arv [--flags] subcommand|resource [...options]

where ``--flags`` are common CLI options for the various subcommands.

The ``ArvCLIArgumentParser`` class, specializing the standard Python
``argparse.ArgumentParser``, provides the support for this CLI usage.
"""


import argparse


class _HelplessArgumentParser(argparse.ArgumentParser):
    """Convenient wrapper class for ArgumentParser that does not consume the
    -h/--help parameter, for use as the subcommands' parser class.
    """
    def __init__(self, **kwargs):
        super().__init__(add_help=False, **kwargs)


class ArvCLIArgumentParser(argparse.ArgumentParser):
    """Argument parser for ``arv`` commands.
    """
    def __init__(self, **kwargs):
        super().__init__(description="Arvados command line client",
                         **kwargs)
        # Common flags to the main command.
        self.add_argument("-n", "--dry-run", action="store_true",
                          help="Don't actually do anything")
        self.add_argument("-v", "--verbose", action="store_true",
                          help="Print some things on stderr")
        # Default output format is JSON, while "-s" or "--short" can be
        # used as a shorthand for "--format=uuid". Specifying both -f and
        # -s is an error.
        format_args = self.add_mutually_exclusive_group()
        format_args.add_argument("-f", "--format",
                                 choices=["json", "yaml", "uuid"],
                                 default="json",
                                 help="Set output format")
        format_args.add_argument("-s", "--short",
                                 dest="format",
                                 action="store_const", const="uuid",
                                 help=("Return only UUIDs "
                                       "(equilvalent to --format=uuid)"))

        subparsers = self.add_subparsers(dest="subcommand",
                                         help="Subcommands",
                                         parser_class=_HelplessArgumentParser)

        keep_parser = subparsers.add_parser("keep")
        keep_parser.add_argument("method",
                                 choices=["ls", "get", "put", "docker"])

        ws_parser = subparsers.add_parser("ws")
        copy_parser = subparsers.add_parser("copy")


def dispatch():
    import sys

    cmd_parser = ArvCLIArgumentParser()
    args, remaining_args = cmd_parser.parse_known_args()

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
    try:
        sys.exit(main(remaining_args))
    except NameError as e:
        if e.name != "main":
            raise e


if __name__ == "__main__":
    dispatch()
