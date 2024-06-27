// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContainerRequestResource } from "./container-request";
import { MountType, MountKind } from 'models/mount-types';
import { WorkflowResource, parseWorkflowDefinition, getWorkflow, CwlSecrets } from 'models/workflow';
import { WorkflowInputsData } from './workflow';

export type ProcessResource = ContainerRequestResource;

export const MOUNT_PATH_CWL_WORKFLOW = '/var/lib/cwl/workflow.json';
export const MOUNT_PATH_CWL_INPUT = '/var/lib/cwl/cwl.input.json';


export const createWorkflowMounts = (workflow: WorkflowResource, inputs: WorkflowInputsData): { [path: string]: MountType } => {

    const wfdef = parseWorkflowDefinition(workflow);
    const mounts: {[path: string]: MountType} = {
        '/var/spool/cwl': {
            kind: MountKind.COLLECTION,
            writable: true,
        },
        'stdout': {
            kind: MountKind.MOUNTED_FILE,
            path: '/var/spool/cwl/cwl.output.json',
        },
        '/var/lib/cwl/workflow.json': {
            kind: MountKind.JSON,
            content: wfdef,
        },
        '/var/lib/cwl/cwl.input.json': {
            kind: MountKind.JSON,
            content: inputs,
        }
    };

    return mounts;
};

export const createWorkflowSecretMounts = (workflow: WorkflowResource, inputs: WorkflowInputsData): { [path: string]: MountType } => {

    const wfdef = parseWorkflowDefinition(workflow);
    const secret_mounts: {[path: string]: MountType} = {};

    const wf = getWorkflow(wfdef);
    if (wf?.hints) {
        const secrets = wf.hints.find(item => item.class === 'http://commonwl.org/cwltool#Secrets') as CwlSecrets | undefined;
        if (secrets?.secrets) {
            let secretCount = 0;
            secrets.secrets.forEach((paramId) => {
                const param = paramId.split("/").pop();
                if (!param || !inputs[param]) {
                    return;
                }
                const value: string = inputs[param] as string;
                const mnt = "/secrets/s"+secretCount;
                secret_mounts[mnt] = {
                    "kind": MountKind.TEXT,
                    "content": value
                }
                inputs[param] = {"$include": mnt}
                secretCount++;
            });
        }
    }
    return secret_mounts;
};
