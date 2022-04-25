# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

{
    "$graph": [
        {
            "class": "Workflow",
            "hints": [
                {
                    "acrContainerImage": "999999999999999999999999999999d3+99",
                    "class": "http://arvados.org/cwl#WorkflowRunnerResources"
                }
            ],
            "id": "#main",
            "inputs": [],
            "outputs": [],
            "steps": [
                {
                    "id": "#main/step1",
                    "in": [],
                    "out": [],
                    "run": "#step1.cwl"
                },
                {
                    "id": "#main/step2",
                    "in": [],
                    "out": [],
                    "run": "#step2.cwl"
                }
            ]
        },
        {
            "arguments": [
                "echo",
                "$(inputs.a)",
                "$(inputs.b)"
            ],
            "class": "CommandLineTool",
            "id": "#step1.cwl",
            "inputs": [
                {
                    "default": {
                        "basename": "a.txt",
                        "class": "File",
                        "location": "keep:b9fca8bf06b170b8507b80b2564ee72b+57/a.txt",
                        "nameext": ".txt",
                        "nameroot": "a"
                    },
                    "id": "#step1.cwl/a",
                    "type": "File"
                },
                {
                    "default": {
                        "basename": "b.txt",
                        "class": "File",
                        "location": "keep:b9fca8bf06b170b8507b80b2564ee72b+57/b.txt",
                        "nameext": ".txt",
                        "nameroot": "b"
                    },
                    "id": "#step1.cwl/b",
                    "type": "File"
                }
            ],
            "outputs": []
        },
        {
            "arguments": [
                "echo",
                "$(inputs.c)",
                "$(inputs.b)"
            ],
            "class": "CommandLineTool",
            "id": "#step2.cwl",
            "inputs": [
                {
                    "default": {
                        "basename": "b.txt",
                        "class": "File",
                        "location": "keep:8e2d09a066d96cdffdd2be41579e4e2e+57/b.txt",
                        "nameext": ".txt",
                        "nameroot": "b"
                    },
                    "id": "#step2.cwl/b",
                    "type": "File"
                },
                {
                    "default": {
                        "basename": "c.txt",
                        "class": "File",
                        "location": "keep:8e2d09a066d96cdffdd2be41579e4e2e+57/c.txt",
                        "nameext": ".txt",
                        "nameroot": "c"
                    },
                    "id": "#step2.cwl/c",
                    "type": "File"
                }
            ],
            "outputs": []
        }
    ],
    "cwlVersion": "v1.0"
}
