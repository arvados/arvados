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
                    "id": "#main/collection_per_tool.cwl",
                    "in": [],
                    "label": "collection_per_tool.cwl",
                    "out": [],
                    "run": "keep:473135c3f4af514f85027e9e348cea45+179/collection_per_tool.cwl"
                }
            ]
        }
    ],
    "cwlVersion": "v1.2"
}
