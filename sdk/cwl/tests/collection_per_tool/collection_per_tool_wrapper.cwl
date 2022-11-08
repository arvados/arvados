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
            "requirements": [
                {
                    "class": "SubworkflowFeatureRequirement"
                }
            ],
            "steps": [
                {
                    "id": "#main/step",
                    "in": [],
                    "out": [],
                    "run": "keep:92045991f69a417f2f26660db67911ef+61/workflow.json#main"
                }
            ]
        }
    ],
    "cwlVersion": "v1.2"
}
