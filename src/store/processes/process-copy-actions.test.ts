// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { configure } from 'enzyme';
import Adapter from 'enzyme-adapter-react-16';
import { copyProcess } from './process-copy-actions';
import { CommonService } from 'services/common-service/common-service';
import { snakeCase } from 'lodash';

configure({ adapter: new Adapter() });

describe('ProcessCopyAction', () => {
    // let props;
    let dispatch: any, getState: any, services: any;

    let sampleFailedProcess = {
        command: [
        "arvados-cwl-runner",
        "--api=containers",
        "--local",
        "--project-uuid=zzzzz-j7d0g-yr18k784zplfeza",
        "/var/lib/cwl/workflow.json#main",
        "/var/lib/cwl/cwl.input.json",
        ],
        container_count: 1,
        container_count_max: 10,
        container_image: "arvados/jobs",
        container_uuid: "zzzzz-dz642-b9j9dtk1yikp9h0",
        created_at: "2023-01-23T22:50:50.788284000Z",
        cumulative_cost: 0.00120553009559028,
        cwd: "/var/spool/cwl",
        description: "test decsription",
        environment: {},
        etag: "2es6px6q7uo0yqi2i291x8gd6",
        expires_at: null,
        filters: null,
        href: "/container_requests/zzzzz-xvhdp-111111111111111",
        kind: "arvados#containerRequest",
        log_uuid: "zzzzz-4zz18-a1gxqy9o6zyrdy8",
        modified_at: "2023-01-24T21:13:54.772612000Z",
        modified_by_client_uuid: "zzzzz-ozdt8-q6dzdi1lcc03155",
        modified_by_user_uuid: "jutro-tpzed-vllbpebicy84rd5",
        mounts: {
        "/var/lib/cwl/cwl.input.json": {
            capacity: 0,
            commit: "",
            content: {
            input: {
                basename: "logo.ai.no.whitespace.png",
                class: "File",
                location:
                "keep:5d3238c4db721a92c98b0305a47b0485+75/logo.ai.no.whitespace.png",
            },
            reverse_sort: true,
            },
            device_type: "",
            exclude_from_output: false,
            git_url: "",
            kind: "json",
            path: "",
            portable_data_hash: "",
            repository_name: "",
            uuid: "",
            writable: false,
        },
        "/var/lib/cwl/workflow.json": {
            capacity: 0,
            commit: "",
            content: {
            $graph: [
                {
                class: "Workflow",
                doc: "Reverse the lines in a document, then sort those lines.",
                id: "#main",
                inputs: [
                    {
                    default: null,
                    doc: "The input file to be processed.",
                    id: "#main/input",
                    type: "File",
                    },
                    {
                    default: true,
                    doc: "If true, reverse (decending) sort",
                    id: "#main/reverse_sort",
                    type: "boolean",
                    },
                ],
                outputs: [
                    {
                    doc: "The output with the lines reversed and sorted.",
                    id: "#main/output",
                    outputSource: "#main/sorted/output",
                    type: "File",
                    },
                ],
                steps: [
                    {
                    id: "#main/rev",
                    in: [{ id: "#main/rev/input", source: "#main/input" }],
                    out: ["#main/rev/output"],
                    run: "#revtool.cwl",
                    },
                    {
                    id: "#main/sorted",
                    in: [
                        { id: "#main/sorted/input", source: "#main/rev/output" },
                        {
                        id: "#main/sorted/reverse",
                        source: "#main/reverse_sort",
                        },
                    ],
                    out: ["#main/sorted/output"],
                    run: "#sorttool.cwl",
                    },
                ],
                },
                {
                baseCommand: "rev",
                class: "CommandLineTool",
                doc: "Reverse each line using the `rev` command",
                hints: [{ class: "ResourceRequirement", ramMin: 8 }],
                id: "#revtool.cwl",
                inputs: [
                    { id: "#revtool.cwl/input", inputBinding: {}, type: "File" },
                ],
                outputs: [
                    {
                    id: "#revtool.cwl/output",
                    outputBinding: { glob: "output.txt" },
                    type: "File",
                    },
                ],
                stdout: "output.txt",
                },
                {
                baseCommand: "sort",
                class: "CommandLineTool",
                doc: "Sort lines using the `sort` command",
                hints: [{ class: "ResourceRequirement", ramMin: 8 }],
                id: "#sorttool.cwl",
                inputs: [
                    {
                    id: "#sorttool.cwl/reverse",
                    inputBinding: { position: 1, prefix: "-r" },
                    type: "boolean",
                    },
                    {
                    id: "#sorttool.cwl/input",
                    inputBinding: { position: 2 },
                    type: "File",
                    },
                ],
                outputs: [
                    {
                    id: "#sorttool.cwl/output",
                    outputBinding: { glob: "output.txt" },
                    type: "File",
                    },
                ],
                stdout: "output.txt",
                },
            ],
            cwlVersion: "v1.0",
            },
            device_type: "",
            exclude_from_output: false,
            git_url: "",
            kind: "json",
            path: "",
            portable_data_hash: "",
            repository_name: "",
            uuid: "",
            writable: false,
        },
        "/var/spool/cwl": {
            capacity: 0,
            commit: "",
            content: null,
            device_type: "",
            exclude_from_output: false,
            git_url: "",
            kind: "collection",
            path: "",
            portable_data_hash: "",
            repository_name: "",
            uuid: "",
            writable: true,
        },
        stdout: {
            capacity: 0,
            commit: "",
            content: null,
            device_type: "",
            exclude_from_output: false,
            git_url: "",
            kind: "file",
            path: "/var/spool/cwl/cwl.output.json",
            portable_data_hash: "",
            repository_name: "",
            uuid: "",
            writable: false,
        },
        },
        name: "Copy of: Copy of: Copy of: revsort.cwl",
        output_name: "Output from revsort.cwl",
        output_path: "/var/spool/cwl",
        output_properties: { key: "val" },
        output_storage_classes: ["default"],
        output_ttl: 999999,
        output_uuid: "zzzzz-4zz18-wolwlyfxmlhmgd4",
        owner_uuid: "zzzzz-j7d0g-yr18k784zplfeza",
        priority: 500,
        properties: {
        template_uuid: "zzzzz-7fd4e-7xsza0vgfe785cy",
        workflowName: "revsort.cwl",
        },
        requesting_container_uuid: null,
        runtime_constraints: {
        API: true,
        cuda: { device_count: 0, driver_version: "", hardware_capability: "" },
        keep_cache_disk: 0,
        keep_cache_ram: 0,
        ram: 1342177280,
        vcpus: 1,
        },
        runtime_token: "",
        scheduling_parameters: {
        max_run_time: 0,
        partitions: [],
        preemptible: false,
        },
        state: "Final",
        use_existing: false,
        uuid: "zzzzz-xvhdp-111111111111111",
    };

    let expectedContainerRequest = {
        command: [
        "arvados-cwl-runner",
        "--api=containers",
        "--local",
        "--project-uuid=zzzzz-j7d0g-yr18k784zplfeza",
        "/var/lib/cwl/workflow.json#main",
        "/var/lib/cwl/cwl.input.json",
        ],
        container_count_max: 10,
        container_image: "arvados/jobs",
        cwd: "/var/spool/cwl",
        description: "test decsription",
        environment: {},
        kind: "arvados#containerRequest",
        mounts: {
        "/var/lib/cwl/cwl.input.json": {
            capacity: 0,
            commit: "",
            content: {
            input: {
                basename: "logo.ai.no.whitespace.png",
                class: "File",
                location:
                "keep:5d3238c4db721a92c98b0305a47b0485+75/logo.ai.no.whitespace.png",
            },
            reverse_sort: true,
            },
            device_type: "",
            exclude_from_output: false,
            git_url: "",
            kind: "json",
            path: "",
            portable_data_hash: "",
            repository_name: "",
            uuid: "",
            writable: false,
        },
        "/var/lib/cwl/workflow.json": {
            capacity: 0,
            commit: "",
            content: {
            $graph: [
                {
                class: "Workflow",
                doc: "Reverse the lines in a document, then sort those lines.",
                id: "#main",
                inputs: [
                    {
                    default: null,
                    doc: "The input file to be processed.",
                    id: "#main/input",
                    type: "File",
                    },
                    {
                    default: true,
                    doc: "If true, reverse (decending) sort",
                    id: "#main/reverse_sort",
                    type: "boolean",
                    },
                ],
                outputs: [
                    {
                    doc: "The output with the lines reversed and sorted.",
                    id: "#main/output",
                    outputSource: "#main/sorted/output",
                    type: "File",
                    },
                ],
                steps: [
                    {
                    id: "#main/rev",
                    in: [{ id: "#main/rev/input", source: "#main/input" }],
                    out: ["#main/rev/output"],
                    run: "#revtool.cwl",
                    },
                    {
                    id: "#main/sorted",
                    in: [
                        {
                        id: "#main/sorted/input",
                        source: "#main/rev/output",
                        },
                        {
                        id: "#main/sorted/reverse",
                        source: "#main/reverse_sort",
                        },
                    ],
                    out: ["#main/sorted/output"],
                    run: "#sorttool.cwl",
                    },
                ],
                },
                {
                baseCommand: "rev",
                class: "CommandLineTool",
                doc: "Reverse each line using the `rev` command",
                hints: [{ class: "ResourceRequirement", ramMin: 8 }],
                id: "#revtool.cwl",
                inputs: [
                    {
                    id: "#revtool.cwl/input",
                    inputBinding: {},
                    type: "File",
                    },
                ],
                outputs: [
                    {
                    id: "#revtool.cwl/output",
                    outputBinding: { glob: "output.txt" },
                    type: "File",
                    },
                ],
                stdout: "output.txt",
                },
                {
                baseCommand: "sort",
                class: "CommandLineTool",
                doc: "Sort lines using the `sort` command",
                hints: [{ class: "ResourceRequirement", ramMin: 8 }],
                id: "#sorttool.cwl",
                inputs: [
                    {
                    id: "#sorttool.cwl/reverse",
                    inputBinding: { position: 1, prefix: "-r" },
                    type: "boolean",
                    },
                    {
                    id: "#sorttool.cwl/input",
                    inputBinding: { position: 2 },
                    type: "File",
                    },
                ],
                outputs: [
                    {
                    id: "#sorttool.cwl/output",
                    outputBinding: { glob: "output.txt" },
                    type: "File",
                    },
                ],
                stdout: "output.txt",
                },
            ],
            cwlVersion: "v1.0",
            },
            device_type: "",
            exclude_from_output: false,
            git_url: "",
            kind: "json",
            path: "",
            portable_data_hash: "",
            repository_name: "",
            uuid: "",
            writable: false,
        },
        "/var/spool/cwl": {
            capacity: 0,
            commit: "",
            content: null,
            device_type: "",
            exclude_from_output: false,
            git_url: "",
            kind: "collection",
            path: "",
            portable_data_hash: "",
            repository_name: "",
            uuid: "",
            writable: true,
        },
        stdout: {
            capacity: 0,
            commit: "",
            content: null,
            device_type: "",
            exclude_from_output: false,
            git_url: "",
            kind: "file",
            path: "/var/spool/cwl/cwl.output.json",
            portable_data_hash: "",
            repository_name: "",
            uuid: "",
            writable: false,
        },
        },
        name: "newname.cwl",
        output_name: "Output from revsort.cwl",
        output_path: "/var/spool/cwl",
        output_properties: { key: "val" },
        output_storage_classes: ["default"],
        output_ttl: 999999,
        owner_uuid: "zzzzz-j7d0g-000000000000000",
        priority: 500,
        properties: {
        template_uuid: "zzzzz-7fd4e-7xsza0vgfe785cy",
        workflowName: "revsort.cwl",
        },
        runtime_constraints: {
        API: true,
        cuda: {
            device_count: 0,
            driver_version: "",
            hardware_capability: "",
        },
        keep_cache_disk: 0,
        keep_cache_ram: 0,
        ram: 1342177280,
        vcpus: 1,
        },
        scheduling_parameters: {
        max_run_time: 0,
        partitions: [],
        preemptible: false,
        },
        state: "Uncommitted",
        use_existing: false,
    };

    beforeEach(() => {
        dispatch = jest.fn();
        services = {
            containerRequestService: {
                get: jest.fn().mockImplementation(async () => (CommonService.mapResponseKeys({data: sampleFailedProcess}))),
                create: jest.fn().mockImplementation(async (data) => (CommonService.mapKeys(snakeCase)(data))),
            },
        };
        getState = () => ({
            auth: {},
        });
    });

    it("should request the failed process and return a copy with the proper fields", async () => {
        // when
        const newprocess = await copyProcess({
            name: "newname.cwl",
            uuid: "zzzzz-xvhdp-111111111111111",
            ownerUuid: "zzzzz-j7d0g-000000000000000",
        })(dispatch, getState, services);

        // then
        expect(services.containerRequestService.get).toHaveBeenCalledWith("zzzzz-xvhdp-111111111111111");
        expect(newprocess).toEqual(expectedContainerRequest);

    });
});
