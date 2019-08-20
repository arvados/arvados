// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContainerRequestResource } from "./container-request";
import { MountType, MountKind } from '~/models/mount-types';
import { WorkflowResource, parseWorkflowDefinition } from '~/models/workflow';
import { WorkflowInputsData } from './workflow';

export type ProcessResource = ContainerRequestResource;

export const MOUNT_PATH_CWL_WORKFLOW = '/var/lib/cwl/workflow.json';
export const MOUNT_PATH_CWL_INPUT = '/var/lib/cwl/cwl.input.json';

export const createWorkflowMounts = (workflow: WorkflowResource, inputs: WorkflowInputsData): { [path: string]: MountType } => {
    return {
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
            content: parseWorkflowDefinition(workflow)
        },
        '/var/lib/cwl/cwl.input.json': {
            kind: MountKind.JSON,
            content: inputs,
        }
    };
};
