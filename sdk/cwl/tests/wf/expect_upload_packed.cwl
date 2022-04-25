# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

{
    "$graph": [
        {
            "baseCommand": "cat",
            "class": "CommandLineTool",
            "id": "#submit_tool.cwl",
            "inputs": [
                {
                    "default": {
                        "basename": "blub.txt",
                        "class": "File",
                        "location": "keep:5d373e7629203ce39e7c22af98a0f881+52/blub.txt",
                        "nameext": ".txt",
                        "nameroot": "blub"
                    },
                    "id": "#submit_tool.cwl/x",
                    "inputBinding": {
                        "position": 1
                    },
                    "type": "File"
                }
            ],
            "outputs": [],
            "requirements": [
                {
                    "class": "DockerRequirement",
                    "dockerPull": "debian:buster-slim",
                    "http://arvados.org/cwl#dockerCollectionPDH": "999999999999999999999999999999d4+99"
                }
            ]
        },
        {
            "class": "Workflow",
            "hints": [
                {
                    "acrContainerImage": "999999999999999999999999999999d3+99",
                    "class": "http://arvados.org/cwl#WorkflowRunnerResources"
                }
            ],
            "id": "#main",
            "inputs": [
                {
                    "default": {
                        "basename": "blorp.txt",
                        "class": "File",
                        "location": "keep:169f39d466a5438ac4a90e779bf750c7+53/blorp.txt",
                        "nameext": ".txt",
                        "nameroot": "blorp",
                        "size": 16
                    },
                    "id": "#main/x",
                    "type": "File"
                },
                {
                    "default": {
                        "basename": "99999999999999999999999999999998+99",
                        "class": "Directory",
                        "location": "keep:99999999999999999999999999999998+99"
                    },
                    "id": "#main/y",
                    "type": "Directory"
                },
                {
                    "default": {
                        "basename": "anonymous",
                        "class": "Directory",
                        "listing": [
                            {
                                "basename": "renamed.txt",
                                "class": "File",
                                "location": "keep:99999999999999999999999999999998+99/file1.txt",
                                "nameext": ".txt",
                                "nameroot": "renamed",
                                "size": 0
                            }
                        ],
                        "location": "_:df80736f-f14d-4b10-b2e3-03aa27f034b2"
                    },
                    "id": "#main/z",
                    "type": "Directory"
                }
            ],
            "outputs": [],
            "steps": [
                {
                    "id": "#main/step1",
                    "in": [
                        {
                            "id": "#main/step1/x",
                            "source": "#main/x"
                        }
                    ],
                    "out": [],
                    "run": "#submit_tool.cwl"
                }
            ]
        }
    ],
    "cwlVersion": "v1.0"
}
