{
    "$graph": [
        {
            "baseCommand": "cat",
            "class": "CommandLineTool",
            "id": "#submit_tool.cwl",
            "inputs": [
                {
                    "default": {
                        "class": "File",
                        "location": "keep:99999999999999999999999999999991+99/tool/blub.txt"
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
                    "dockerImageId": "debian:8",
                    "dockerPull": "debian:8"
                }
            ]
        },
        {
            "class": "Workflow",
            "id": "#main",
            "inputs": [
                {
                    "default": {
                        "basename": "blorp.txt",
                        "class": "File",
                        "location": "keep:99999999999999999999999999999991+99/input/blorp.txt"
                    },
                    "id": "#main/x",
                    "type": "File"
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