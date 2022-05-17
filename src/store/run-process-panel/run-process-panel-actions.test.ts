// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { runProcess } from "./run-process-panel-actions";

jest.mock("../navigation/navigation-action", () => ({
    navigateTo: (link: any) => link,
}));

jest.mock("models/process", () => ({
    createWorkflowMounts: jest.fn(),
}));

jest.mock("redux-form", () => ({
    reduxForm: () => (c: any) => c,
    getFormValues: (name: string) => () => {
        switch (name) {
            case "runProcessBasicForm":
                return {
                    name: "basicFormTestName",
                    description: "basicFormTestDescription",
                };
            case "runProcessInputsForm":
                return {};
            default:
                return null;
        }
    },
}));

describe("run-process-panel-actions", () => {
    describe("runProcess", () => {
        const newProcessUUID = 'newProcessUUID';
        let dispatch: any, getState: any, services: any;

        beforeEach(() => {
            dispatch = jest.fn();
            services = {
                containerRequestService: {
                    create: jest.fn().mockImplementation(async () => ({
                        uuid: newProcessUUID,
                    })),
                },
            };
        });

        it("should return when userUuid is null", async () => {
            // given
            getState = () => ({
                auth: {},
            });

            // when
            await runProcess(dispatch, getState, services);

            // then
            expect(dispatch).not.toHaveBeenCalled();
        });

        it("should run workflow with project-uuid", async () => {
            // given
            getState = () => ({
                auth: {
                    user: {
                        email: "test@gmail.com",
                        firstName: "TestFirstName",
                        lastName: "TestLastName",
                        uuid: "ce8i5-tpzed-yid70bw31f51234",
                        ownerUuid: "ce8i5-tpzed-000000000000000",
                        isAdmin: false,
                        isActive: true,
                        username: "testfirstname",
                        prefs: {
                            profile: {},
                        },
                    },
                },
                runProcessPanel: {
                    processPathname: "/projects/ce8i5-tpzed-yid70bw31f51234",
                    processOwnerUuid: "ce8i5-tpzed-yid70bw31f51234",
                    selectedWorkflow: {
                        href: "/workflows/ce8i5-7fd4e-2tlnerdkxnl4fjt",
                        kind: "arvados#workflow",
                        etag: "8gh5xlhlgo61yqscyl1spw8tc",
                        uuid: "ce8i5-7fd4e-2tlnerdkxnl4fjt",
                        ownerUuid: "ce8i5-tpzed-o4njwilpp4ov321",
                        createdAt: "2020-07-15T19:40:50.296041000Z",
                        modifiedByClientUuid: "ce8i5-ozdt8-libnr89sc5nq111",
                        modifiedByUserUuid: "ce8i5-tpzed-o4njwilpp4ov321",
                        modifiedAt: "2020-07-15T19:40:50.296376000Z",
                        name: "revsort.cwl",
                        description:
                            "Reverse the lines in a document, then sort those lines.",
                        definition:
                            '{\n    "$graph": [\n        {\n            "class": "Workflow",\n            "doc": "Reverse the lines in a document, then sort those lines.",\n            "id": "#main",\n            "hints":[{"class":"http://arvados.org/cwl#WorkflowRunnerResources","acrContainerImage":"arvados/jobs:2.0.4", "ramMin": 16000}], "inputs": [\n                {\n                    "default": null,\n                    "doc": "The input file to be processed.",\n                    "id": "#main/input",\n                    "type": "File"\n                },\n                {\n                    "default": true,\n                    "doc": "If true, reverse (decending) sort",\n                    "id": "#main/reverse_sort",\n                    "type": "boolean"\n                }\n            ],\n            "outputs": [\n                {\n                    "doc": "The output with the lines reversed and sorted.",\n                    "id": "#main/output",\n                    "outputSource": "#main/sorted/output",\n                    "type": "File"\n                }\n            ],\n            "steps": [\n                {\n                    "id": "#main/rev",\n                    "in": [\n                        {\n                            "id": "#main/rev/input",\n                            "source": "#main/input"\n                        }\n                    ],\n                    "out": [\n                        "#main/rev/output"\n                    ],\n                    "run": "#revtool.cwl"\n                },\n                {\n                    "id": "#main/sorted",\n                    "in": [\n                        {\n                            "id": "#main/sorted/input",\n                            "source": "#main/rev/output"\n                        },\n                        {\n                            "id": "#main/sorted/reverse",\n                            "source": "#main/reverse_sort"\n                        }\n                    ],\n                    "out": [\n                        "#main/sorted/output"\n                    ],\n                    "run": "#sorttool.cwl"\n                }\n            ]\n        },\n        {\n            "baseCommand": "rev",\n            "class": "CommandLineTool",\n            "doc": "Reverse each line using the `rev` command",\n            "hints": [\n                {\n                    "class": "ResourceRequirement",\n                    "ramMin": 8\n                }\n            ],\n            "id": "#revtool.cwl",\n            "inputs": [\n                {\n                    "id": "#revtool.cwl/input",\n                    "inputBinding": {},\n                    "type": "File"\n                }\n            ],\n            "outputs": [\n                {\n                    "id": "#revtool.cwl/output",\n                    "outputBinding": {\n                        "glob": "output.txt"\n                    },\n                    "type": "File"\n                }\n            ],\n            "stdout": "output.txt"\n        },\n        {\n            "baseCommand": "sort",\n            "class": "CommandLineTool",\n            "doc": "Sort lines using the `sort` command",\n            "hints": [\n                {\n                    "class": "ResourceRequirement",\n                    "ramMin": 8\n                }\n            ],\n            "id": "#sorttool.cwl",\n            "inputs": [\n                {\n                    "id": "#sorttool.cwl/reverse",\n                    "inputBinding": {\n                        "position": 1,\n                        "prefix": "-r"\n                    },\n                    "type": "boolean"\n                },\n                {\n                    "id": "#sorttool.cwl/input",\n                    "inputBinding": {\n                        "position": 2\n                    },\n                    "type": "File"\n                }\n            ],\n            "outputs": [\n                {\n                    "id": "#sorttool.cwl/output",\n                    "outputBinding": {\n                        "glob": "output.txt"\n                    },\n                    "type": "File"\n                }\n            ],\n            "stdout": "output.txt"\n        }\n    ],\n    "cwlVersion": "v1.0"\n}',
                    },
                },
            });

            // when
            await runProcess(dispatch, getState, services);

            // then
            expect(services.containerRequestService.create).toHaveBeenCalledWith({
                command: [
                    "arvados-cwl-runner",
                    "--api=containers",
                    "--local",
                    "--project-uuid=ce8i5-tpzed-yid70bw31f51234",
                    "/var/lib/cwl/workflow.json#main",
                    "/var/lib/cwl/cwl.input.json",
                ],
                containerImage: "arvados/jobs:2.0.4",
                cwd: "/var/spool/cwl",
                description: "basicFormTestDescription",
                mounts: undefined,
                name: "basicFormTestName",
                outputName: undefined,
                outputPath: "/var/spool/cwl",
                ownerUuid: "ce8i5-tpzed-yid70bw31f51234",
                priority: 1,
                properties: {
                    workflowName: "revsort.cwl",
                    template_uuid: "ce8i5-7fd4e-2tlnerdkxnl4fjt",
                },
                runtimeConstraints: {
                    API: true,
                    ram: 16256 * (1024 * 1024),
                    vcpus: 1,
                },
                schedulingParameters: { max_run_time: undefined },
                state: "Committed",
                useExisting: false
            });

            // and
            expect(dispatch).toHaveBeenCalledWith(newProcessUUID);
        });
    });
});
