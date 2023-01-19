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
                        ]
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
                    "id": "#main/submit_wf.cwl",
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
                    "label": "submit_wf.cwl",
                    "out": [],
                    "run": "keep:62bd293aa703ddb1897c94711dead399+128/wf/submit_wf.cwl"
                }
            ]
        }
    ],
    "cwlVersion": "v1.2"
}
