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
            "requirements": [
                {
                    "class": "SubworkflowFeatureRequirement"
                }
            ],
            "steps": [
                {
                    "id": "#main/step",
                    "in": [
                        {
                            "id": "#main/step/x",
                            "source": "#main/x"
                        },
                        {
                            "id": "#main/step/y",
                            "source": "#main/y"
                        },
                        {
                            "id": "#main/step/z",
                            "source": "#main/z"
                        }
                    ],
                    "out": [],
                    "run": "keep:f1c2b0c514a5fb9b2a8b5b38a31bab66+61/workflow.json#main"
                }
            ]
        }
    ],
    "cwlVersion": "v1.2"
}
