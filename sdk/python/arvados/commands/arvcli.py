# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import argparse


class ArvCLIArgumentParser(argparse.ArgumentParser):
    """Argument parser for @arv@ commands.
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

        # NOTE: Without explicitly naming "parser_class" for the
        # subparsers, this __init__ method run into infinite recursion (by
        # trying to make the subparsers instances of this derived class).
        subparsers = self.add_subparsers(dest="subcommand",
                                         help="Subcommands",
                                         parser_class=argparse.ArgumentParser)

        keep_parser = subparsers.add_parser("keep")
        keep_parser.add_argument("method",
                                 choices=["ls", "get", "put", "docker"])

        ws_parser = subparsers.add_parser("ws")
        copy_parser = subparsers.add_parser("copy")

        self.subparsers = subparsers


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
    sys.exit(main(remaining_args))


if __name__ == "__main__":
    dispatch()
