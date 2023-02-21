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
                    "id": "#main/x",
                    "type": "File"
                },
                {
                    "id": "#main/y",
                    "type": "Directory"
                },
                {
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
                    "run": "keep:5b4cf4181c65ad292ccba5b142d758a4+274/wf/submit_wf.cwl"
                }
            ]
        }
    ],
    "cwlVersion": "v1.2"
}
