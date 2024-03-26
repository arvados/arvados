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

_ALIASED_METHODS = frozenset([
    'destroy',
    'index',
    'show',
])
_DEPRECATED_NOTICE = '''

.. WARNING:: Deprecated
   This resource is deprecated in the Arvados API.
'''
_DEPRECATED_RESOURCES = frozenset([
    'Humans',
    'JobTasks',
    'Jobs',
    'KeepDisks',
    'Nodes',
    'PipelineInstances',
    'PipelineTemplates',
    'Specimens'
    'Traits',
])
_DEPRECATED_SCHEMAS = frozenset([
    *(name[:-1] for name in _DEPRECATED_RESOURCES),
    *(f'{name[:-1]}List' for name in _DEPRECATED_RESOURCES),
])

_LIST_PYDOC = '''

This is the dictionary object returned when you call `{cls_name}s.list`.
If you just want to iterate all objects that match your search criteria,
consider using `arvados.util.keyset_list_all`.
If you work with this raw object, the keys of the dictionary are documented
below, along with their types. The `items` key maps to a list of matching
`{cls_name}` objects.
'''
_MODULE_PYDOC = '''Arvados API client reference documentation

This module provides reference documentation for the interface of the
Arvados API client, including method signatures and type information for
returned objects. However, the functions in `arvados.api` will return
different classes at runtime that are generated dynamically from the Arvados
API discovery document. The classes in this module do not have any
implementation, and you should not instantiate them in your code.

If you're just starting out, `ArvadosAPIClient` documents the methods
available from the client object. From there, you can follow the trail into
resource methods, request objects, and finally the data dictionaries returned
by the API server.
'''
_SCHEMA_PYDOC = '''

This is the dictionary object that represents a single {cls_name} in Arvados
and is returned by most `{cls_name}s` methods.
The keys of the dictionary are documented below, along with their types.
Not every key may appear in every dictionary returned by an API call.
When a method doesn't return all the data, you can use its `select` parameter
to list the specific keys you need. Refer to the API documentation for details.
'''

_MODULE_PRELUDE = '''
import googleapiclient.discovery
import googleapiclient.http
import httplib2
import sys
from typing import Any, Dict, Generic, List, Optional, TypeVar
if sys.version_info < (3, 8):
    from typing_extensions import TypedDict
else:
    from typing import TypedDict

# ST represents an API response type
ST = TypeVar('ST', bound=TypedDict)
'''
_REQUEST_CLASS = '''
class ArvadosAPIRequest(googleapiclient.http.HttpRequest, Generic[ST]):
    """Generic API request object

    When you call an API method in the Arvados Python SDK, it returns a
    request object. You usually call `execute()` on this object to submit the
    request to your Arvados API server and retrieve the response. `execute()`
    will return the type of object annotated in the subscript of
    `ArvadosAPIRequest`.
    """

    def execute(self, http: Optional[httplib2.Http]=None, num_retries: int=0) -> ST:
        """Execute this request and return the response

        Arguments:

        * http: httplib2.Http | None --- The HTTP client object to use to
          execute the request. If not specified, uses the HTTP client object
          created with the API client object.

        * num_retries: int --- The maximum number of times to retry this
          request if the server returns a retryable failure. The API client
          object also has a maximum number of retries specified when it is
          instantiated (see `arvados.api.api_client`). This request is run
          with the larger of that number and this argument. Default 0.
        """

'''

# Annotation represents a valid Python type annotation. Future development
# could expand this to include other valid types like `type`.
Annotation = str
_TYPE_MAP: Mapping[str, Annotation] = {
    # Map the API's JavaScript-based type names to Python annotations.
    # Some of these may disappear after Arvados issue #19795 is fixed.
    'Array': 'List',
    'array': 'List',
    'boolean': 'bool',
    # datetime fields are strings in ISO 8601 format.
    'datetime': 'str',
    'Hash': 'Dict[str, Any]',
    'integer': 'int',
    'object': 'Dict[str, Any]',
    'string': 'str',
    'text': 'str',
}

def get_type_annotation(name: str) -> str:
    return _TYPE_MAP.get(name, name)

def to_docstring(s: str, indent: int) -> str:
    prefix = ' ' * indent
    s = s.replace('"""', '""\"')
    s = re.sub(r'(\n+)', r'\1' + prefix, s)
    s = s.strip()
    if '\n' in s:
        return f'{prefix}"""{s}\n{prefix}"""'
    else:
        return f'{prefix}"""{s}"""'

def transform_name(s: str, sep: str, fix_part: Callable[[str], str]) -> str:
    return sep.join(fix_part(part) for part in s.split('_'))

def classify_name(s: str) -> str:
    return transform_name(s, '', TITLECASE)

def humanize_name(s: str) -> str:
    return transform_name(s, ' ', LOWERCASE)

class Parameter(inspect.Parameter):
    def __init__(self, name: str, spec: Mapping[str, Any]) -> None:
        self.api_name = name
        self._spec = spec
        if keyword.iskeyword(name):
            name += '_'
        super().__init__(
            name,
            inspect.Parameter.KEYWORD_ONLY,
            annotation=get_type_annotation(self._spec['type']),
            # In normal Python the presence of a default tells you whether or
            # not an argument is required. In the API the `required` flag tells
            # us that, and defaults are specified inconsistently. Don't show
            # defaults in the signature: it adds noise and makes things more
            # confusing for the reader about what's required and what's
            # optional. The docstring can explain in better detail, including
            # the default value.
            default=inspect.Parameter.empty,
        )

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
            default_doc = f"Default {default_value!r}."
        description = self._spec['description']
        doc_parts = [f'{self.api_name}: {self.annotation}']
        if description or default_doc:
            doc_parts.append('---')
            if description:
                doc_parts.append(description)
            if default_doc:
                doc_parts.append(default_doc)
        return f'''
* {' '.join(doc_parts)}
'''


class Method:
    def __init__(
            self,
            name: str,
            spec: Mapping[str, Any],
            annotate: Callable[[Annotation], Annotation]=str,
    ) -> None:
        self.name = name
        self._spec = spec
        self._annotate = annotate
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
            inspect.Parameter('self', inspect.Parameter.POSITIONAL_OR_KEYWORD),
            *self._required_params,
            *self._optional_params,
        ]
        try:
            returns = get_type_annotation(self._spec['response']['$ref'])
        except KeyError:
            returns = 'Dict[str, Any]'
        returns = self._annotate(returns)
        return inspect.Signature(parameters, return_annotation=returns)

    def doc(self, doc_slice: slice=slice(None)) -> str:
        doc_lines = self._spec['description'].splitlines(keepends=True)[doc_slice]
        if not doc_lines[-1].endswith('\n'):
            doc_lines.append('\n')
        if self._required_params:
            doc_lines.append("\nRequired parameters:\n")
            doc_lines.extend(param.doc() for param in self._required_params)
        if self._optional_params:
            doc_lines.append("\nOptional parameters:\n")
            doc_lines.extend(param.doc() for param in self._optional_params)
        return f'''
    def {self.name}{self.signature()}:
{to_docstring(''.join(doc_lines), 8)}
'''


def document_schema(name: str, spec: Mapping[str, Any]) -> str:
    description = spec['description']
    if name in _DEPRECATED_SCHEMAS:
        description += _DEPRECATED_NOTICE
    if name.endswith('List'):
        desc_fmt = _LIST_PYDOC
        cls_name = name[:-4]
    else:
        desc_fmt = _SCHEMA_PYDOC
        cls_name = name
    description += desc_fmt.format(cls_name=cls_name)
    lines = [
        f"class {name}(TypedDict, total=False):",
        to_docstring(description, 4),
    ]
    for field_name, field_spec in spec['properties'].items():
        field_type = get_type_annotation(field_spec['type'])
        try:
            subtype = field_spec['items']['$ref']
        except KeyError:
            pass
        else:
            field_type += f"[{get_type_annotation(subtype)}]"

        field_line = f"    {field_name}: {field_type!r}"
        try:
            field_line += f" = {field_spec['default']!r}"
        except KeyError:
            pass
        lines.append(field_line)

        field_doc: str = field_spec.get('description', '')
        if field_spec['type'] == 'datetime':
            field_doc += "\n\nString in ISO 8601 datetime format. Pass it to `ciso8601.parse_datetime` to build a `datetime.datetime`."
        if field_doc:
            lines.append(to_docstring(field_doc, 4))
    lines.append('\n')
    return '\n'.join(lines)

def document_resource(name: str, spec: Mapping[str, Any]) -> str:
    class_name = classify_name(name)
    docstring = f"Methods to query and manipulate Arvados {humanize_name(name)}"
    if class_name in _DEPRECATED_RESOURCES:
        docstring += _DEPRECATED_NOTICE
    methods = [
        Method(key, meth_spec, 'ArvadosAPIRequest[{}]'.format)
        for key, meth_spec in spec['methods'].items()
        if key not in _ALIASED_METHODS
    ]
    return f'''class {class_name}:
{to_docstring(docstring, 4)}
{''.join(method.doc(slice(1)) for method in sorted(methods, key=NAME_KEY))}
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
            args.discovery_url = pathlib.Path(args.discovery_url).resolve().as_uri()
    # Our output is Python source, so it should be UTF-8 regardless of locale.
    if args.output_file == STDSTREAM_PATH:
        args.out_file = open(sys.stdout.fileno(), 'w', encoding='utf-8', closefd=False)
    else:
        args.out_file = args.output_file.open('w', encoding='utf-8')
    return args

def main(arglist: Optional[Sequence[str]]=None) -> int:
    args = parse_arguments(arglist)
    with urllib.request.urlopen(args.discovery_url) as discovery_file:
        status = discovery_file.getcode()
        if not (status is None or 200 <= status < 300):
            print(
                f"error getting {args.discovery_url}: server returned {discovery_file.status}",
                file=sys.stderr,
            )
            return os.EX_IOERR
        discovery_document = json.load(discovery_file)
    print(
        to_docstring(_MODULE_PYDOC, indent=0),
        _MODULE_PRELUDE,
        sep='\n', file=args.out_file,
    )

    schemas = sorted(discovery_document['schemas'].items())
    for name, schema_spec in schemas:
        print(document_schema(name, schema_spec), file=args.out_file)

    resources = sorted(discovery_document['resources'].items())
    for name, resource_spec in resources:
        print(document_resource(name, resource_spec), file=args.out_file)

    print(
        _REQUEST_CLASS,
        '''class ArvadosAPIClient(googleapiclient.discovery.Resource):''',
        sep='\n', file=args.out_file,
    )
    for name, _ in resources:
        class_name = classify_name(name)
        docstring = f"Return an instance of `{class_name}` to call methods via this client"
        if class_name in _DEPRECATED_RESOURCES:
            docstring += _DEPRECATED_NOTICE
        method_spec = {
            'description': docstring,
            'parameters': {},
            'response': {
                '$ref': class_name,
            },
        }
        print(Method(name, method_spec).doc(), file=args.out_file)

    args.out_file.close()
    return os.EX_OK

if __name__ == '__main__':
    sys.exit(main())
