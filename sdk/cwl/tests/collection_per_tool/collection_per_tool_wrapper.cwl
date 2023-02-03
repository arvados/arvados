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
                    "run": "keep:177002db236f41230905621862cc4230+367/collection_per_tool.cwl"
                }
            ]
        }
    ],
    "cwlVersion": "v1.2"
}
