// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Resource, ResourceKind } from "./resource";
import { safeLoad } from 'js-yaml';

export interface WorkflowResource extends Resource {
    kind: ResourceKind.WORKFLOW;
    name: string;
    description: string;
    definition: string;
}
export interface WorkflowResoruceDefinition {
    cwlVersion: string;
    $graph: Array<Workflow | CommandLineTool>;
}
export interface Workflow {
    class: 'Workflow';
    doc?: string;
    id?: string;
    inputs: CommandInputParameter[];
    outputs: any[];
    steps: any[];
}

export interface CommandLineTool {
    class: 'CommandLineTool';
    id: string;
    inputs: CommandInputParameter[];
    outputs: any[];
}

export interface CommandInputParameter {
    id: string;
    label?: string;
    doc?: string | string[];
    default?: any;
    type?: CWLType | CWLType[] | CommandInputEnumSchema | CommandInputArraySchema;
}

export enum CWLType {
    NULL = 'null',
    BOOLEAN = 'boolean',
    INT = 'int',
    LONG = 'long',
    FLOAT = 'float',
    DOUBLE = 'double',
    STRING = 'string',
    FILE = 'File',
    DIRECTORY = 'Directory',
}

export interface CommandInputEnumSchema {
    symbols: string[];
    type: 'enum';
    label?: string;
    name?: string;
}

export interface CommandInputArraySchema {
    items: CWLType;
    type: 'array';
    label?: string;
}

export interface File {
    class: CWLType.FILE;
    location?: string;
    path?: string;
    basename?: string;
}

export interface Directory {
    class: CWLType.DIRECTORY;
    location?: string;
    path?: string;
    basename?: string;
}

export const parseWorkflowDefinition = (workflow: WorkflowResource): WorkflowResoruceDefinition => {
    const definition = safeLoad(workflow.definition);
    return definition;
};

export const getWorkflowInputs = (workflowDefinition: WorkflowResoruceDefinition) => {
    const mainWorkflow = workflowDefinition.$graph.find(item => item.class === 'Workflow' && item.id === '#main');
    return mainWorkflow
        ? mainWorkflow.inputs
        : undefined;
};

export const stringifyInputType = ({ type }: CommandInputParameter) => {
    if (typeof type === 'string') {
        return type;
    } else if (type instanceof Array) {
        return type.join(' | ');
    } else if (typeof type === 'object') {
        if (type.type === 'enum') {
            return 'enum';
        } else if (type.type === 'array') {
            return `${type.items}[]`;
        } else {
            return 'unknown';
        }
    } else {
        return 'unknown';
    }
};
