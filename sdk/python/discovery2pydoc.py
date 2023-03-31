#!/usr/bin/env python3
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
"""discovery2pydoc - Build skeleton Python from the Arvados discovery document

This tool reads the Arvados discovery document and writes a Python source file
with classes and methods that correspond to the resources that
google-api-python-client builds dynamically. This source does not include any
implementation, but it does include real method signatures and documentation
strings, so it's useful as documentation for tools that read Python source,
including pydoc and pdoc.

If you run this tool with the path to a discovery document, it uses no
dependencies outside the Python standard library. If it needs to read
configuration to find the discovery document dynamically, it'll load the
`arvados` module to do that.
"""

import argparse
import inspect
import json
import keyword
import operator
import os
import pathlib
import re
import sys
import urllib.parse
import urllib.request

from typing import (
    Any,
    Callable,
    Mapping,
    Optional,
    Sequence,
)

LOWERCASE = operator.methodcaller('lower')
NAME_KEY = operator.attrgetter('name')
STDSTREAM_PATH = pathlib.Path('-')
TITLECASE = operator.methodcaller('title')

def transform_name(s: str, sep: str, fix_part: Callable[[str], str]) -> str:
    return sep.join(fix_part(part) for part in s.split('_'))

def classify_name(s: str) -> str:
    return transform_name(s, '', TITLECASE)

def humanize_name(s: str) -> str:
    return transform_name(s, ' ', LOWERCASE)

class Parameter(inspect.Parameter):
    _TYPE_MAP = {
        # Map the API's JavaScript-based type names to Python annotations
        'array': 'list',
        'boolean': 'bool',
        'integer': 'int',
        'object': 'dict[str, Any]',
        'string': 'str',
    }

    def __init__(self, name: str, spec: Mapping[str, Any]) -> None:
        self.api_name = name
        self._spec = spec
        if keyword.iskeyword(name):
            name += '_'
        super().__init__(
            name,
            inspect.Parameter.KEYWORD_ONLY,
            annotation=self.annotation_from_type(),
            # In normal Python the presence of a default tells you whether or
            # not an argument is required. In the API the `required` flag tells
            # us that, and defaults are specified inconsistently. Don't show
            # defaults in the signature: it adds noise and makes things more
            # confusing for the reader about what's required and what's
            # optional. The docstring can explain in better detail, including
            # the default value.
            default=inspect.Parameter.empty,
        )

    def annotation_from_type(self) -> str:
        src_type = self._spec['type']
        return self._TYPE_MAP.get(src_type, src_type)

    def default_value(self) -> object:
        try:
            src_value: str = self._spec['default']
        except KeyError:
            return None
        if src_value == 'true':
            return True
        elif src_value == 'false':
            return False
        elif src_value.isdigit():
            return int(src_value)
        else:
            return src_value

    def is_required(self) -> bool:
        return self._spec['required']

    def doc(self) -> str:
        default_value = self.default_value()
        if default_value is None:
            default_doc = ''
        else:
            default_doc = f" Default {default_value!r}."
        # If there is no description, use a zero-width space to help Markdown
        # parsers retain the definition list structure.
        description = self._spec['description'] or '\u200b'
        return f'''
        {self.api_name}: {self.annotation}
        : {description}{default_doc}
'''


class Method:
    def __init__(self, name: str, spec: Mapping[str, Any]) -> None:
        self.name = name
        self._spec = spec
        self._required_params = []
        self._optional_params = []
        for param_name, param_spec in spec['parameters'].items():
            param = Parameter(param_name, param_spec)
            if param.is_required():
                param_list = self._required_params
            else:
                param_list = self._optional_params
            param_list.append(param)
        self._required_params.sort(key=NAME_KEY)
        self._optional_params.sort(key=NAME_KEY)

    def signature(self) -> inspect.Signature:
        parameters = [
            inspect.Parameter('self', inspect.Parameter.POSITIONAL_ONLY),
            *self._required_params,
            *self._optional_params,
        ]
        return inspect.Signature(parameters, return_annotation='dict[str, Any]')

    def doc(self) -> str:
        return re.sub(r'\n{3,}', '\n\n', f'''
    def {self.name}{self.signature()}:
        """{self._spec['description'].splitlines()[0]}

{"        Required parameters:" if self._required_params else ""}

{''.join(param.doc() for param in self._required_params)}

{"        Optional parameters:" if self._optional_params else ""}

{''.join(param.doc() for param in self._optional_params)}
        """
''')


def document_resource(name: str, spec: Mapping[str, Any]) -> str:
    methods = [Method(key, meth_spec) for key, meth_spec in spec['methods'].items()]
    return f'''class {classify_name(name)}:
    """Methods to query and manipulate Arvados {humanize_name(name)}"""
{''.join(method.doc() for method in sorted(methods, key=NAME_KEY))}
'''

def parse_arguments(arglist: Optional[Sequence[str]]) -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument(
        '--output-file', '-O',
        type=pathlib.Path,
        metavar='PATH',
        default=STDSTREAM_PATH,
        help="""Path to write output. Specify `-` to use stdout (the default)
""")
    parser.add_argument(
        'discovery_url',
        nargs=argparse.OPTIONAL,
        metavar='URL',
        help="""URL or file path of a discovery document to load.
Specify `-` to use stdin.
If not provided, retrieved dynamically from Arvados client configuration.
""")
    args = parser.parse_args(arglist)
    if args.discovery_url is None:
        from arvados.api import api_kwargs_from_config
        discovery_fmt = api_kwargs_from_config('v1')['discoveryServiceUrl']
        args.discovery_url = discovery_fmt.format(api='arvados', apiVersion='v1')
    elif args.discovery_url == '-':
        args.discovery_url = 'file:///dev/stdin'
    else:
        parts = urllib.parse.urlsplit(args.discovery_url)
        if not (parts.scheme or parts.netloc):
            args.discovery_url = urllib.parse.urlunsplit(parts._replace(scheme='file'))
    if args.output_file == STDSTREAM_PATH:
        args.out_file = sys.stdout
    else:
        args.out_file = args.output_file.open('w')
    return args

def main(arglist: Optional[Sequence[str]]=None) -> int:
    args = parse_arguments(arglist)
    with urllib.request.urlopen(args.discovery_url) as discovery_file:
        if not (discovery_file.status is None or 200 <= discovery_file.status < 300):
            print(
                f"error getting {args.discovery_url}: server returned {discovery_file.status}",
                file=sys.stderr,
            )
            return os.EX_IOERR
        discovery_document = json.load(discovery_file)
    resources = sorted(discovery_document['resources'].items())

    for name, resource_spec in resources:
        print(document_resource(name, resource_spec), file=args.out_file)

    print('''class ArvadosAPIClient:''', file=args.out_file)
    for name, _ in resources:
        method_spec = {
            'description': f"Return an instance of `{classify_name(name)}` to call methods via this client",
            'parameters': {},
        }
        print(Method(name, method_spec).doc(), file=args.out_file)

    return os.EX_OK

if __name__ == '__main__':
    sys.exit(main())
