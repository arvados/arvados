// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Resource, ResourceKind } from "./resource";
import { safeLoad } from 'js-yaml';
import { CommandOutputParameter } from "cwlts/mappings/v1.0/CommandOutputParameter";

export interface WorkflowResource extends Resource {
    kind: ResourceKind.WORKFLOW;
    name: string;
    description: string;
    definition: string;
}
export interface WorkflowResourceDefinition {
    cwlVersion: string;
    $graph?: Array<Workflow | CommandLineTool>;
}
export interface Workflow {
    class: 'Workflow';
    doc?: string;
    id?: string;
    inputs: CommandInputParameter[];
    outputs: any[];
    steps: any[];
    hints?: ProcessRequirement[];
}

export interface CommandLineTool {
    class: 'CommandLineTool';
    id: string;
    inputs: CommandInputParameter[];
    outputs: any[];
    hints?: ProcessRequirement[];
}

export type ProcessRequirement = GenericProcessRequirement | WorkflowRunnerResources;

export interface GenericProcessRequirement {
    class: string;
}

export interface WorkflowRunnerResources {
    class: 'http://arvados.org/cwl#WorkflowRunnerResources';
    ramMin?: number;
    coresMin?: number;
    keep_cache?: number;
    acrContainerImage?: string;
}

export type CommandInputParameter =
    BooleanCommandInputParameter |
    IntCommandInputParameter |
    LongCommandInputParameter |
    FloatCommandInputParameter |
    DoubleCommandInputParameter |
    StringCommandInputParameter |
    FileCommandInputParameter |
    DirectoryCommandInputParameter |
    StringArrayCommandInputParameter |
    IntArrayCommandInputParameter |
    FloatArrayCommandInputParameter |
    FileArrayCommandInputParameter |
    DirectoryArrayCommandInputParameter |
    EnumCommandInputParameter;

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

export interface CommandInputArraySchema<ItemType> {
    items: ItemType;
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

export interface SecretInclude {
    $include: string;
}

export interface GenericCommandInputParameter<Type, Value> {
    id: string;
    label?: string;
    doc?: string | string[];
    default?: Value;
    type?: Type | Array<Type | CWLType.NULL>;
    value?: Value;
    disabled?: boolean;
    secret?: boolean;
}
export type GenericArrayCommandInputParameter<Type, Value> = GenericCommandInputParameter<CommandInputArraySchema<Type>, Value[]>;

export type BooleanCommandInputParameter = GenericCommandInputParameter<CWLType.BOOLEAN, boolean>;
export type IntCommandInputParameter = GenericCommandInputParameter<CWLType.INT, number>;
export type LongCommandInputParameter = GenericCommandInputParameter<CWLType.LONG, number>;
export type FloatCommandInputParameter = GenericCommandInputParameter<CWLType.FLOAT, number>;
export type DoubleCommandInputParameter = GenericCommandInputParameter<CWLType.DOUBLE, number>;
export type StringCommandInputParameter = GenericCommandInputParameter<CWLType.STRING, string>;
export type FileCommandInputParameter = GenericCommandInputParameter<CWLType.FILE, File>;
export type DirectoryCommandInputParameter = GenericCommandInputParameter<CWLType.DIRECTORY, Directory>;
export type EnumCommandInputParameter = GenericCommandInputParameter<CommandInputEnumSchema, string>;

export type StringArrayCommandInputParameter = GenericArrayCommandInputParameter<CWLType.STRING, string>;
export type IntArrayCommandInputParameter = GenericArrayCommandInputParameter<CWLType.INT, string>;
export type FloatArrayCommandInputParameter = GenericArrayCommandInputParameter<CWLType.FLOAT, string>;
export type FileArrayCommandInputParameter = GenericArrayCommandInputParameter<CWLType.FILE, File>;
export type DirectoryArrayCommandInputParameter = GenericArrayCommandInputParameter<CWLType.DIRECTORY, Directory>;
export type SecretCommandInputParameter = GenericArrayCommandInputParameter<CWLType.STRING, SecretInclude>;


export type WorkflowInputsData = {
    [key: string]: boolean | number | string | File | Directory | SecretInclude;
};
export const parseWorkflowDefinition = (workflow: WorkflowResource): WorkflowResourceDefinition => {
    const definition = safeLoad(workflow.definition);
    return definition;
};

export const getWorkflow = (workflowDefinition: WorkflowResourceDefinition) => {
    if (!workflowDefinition.$graph) { return undefined; }
    const mainWorkflow = workflowDefinition.$graph.find(item => item.id === '#main');
    return mainWorkflow
        ? mainWorkflow
        : undefined;
};

export interface CwlSecrets {
    class: 'http://commonwl.org/cwltool#Secrets';
    secrets: string[];
}

export const getWorkflowInputs = (workflowDefinition: WorkflowResourceDefinition) => {
    if (!workflowDefinition) { return undefined; }
    const wf = getWorkflow(workflowDefinition);
    if (!wf) { return undefined; }
    const inputs = wf.inputs;
    if (wf.hints) {
        const secrets = wf.hints.find(item => item.class === 'http://commonwl.org/cwltool#Secrets') as CwlSecrets | undefined;
        if (secrets?.secrets) {
            inputs.forEach((param) => {
                param.secret = secrets.secrets.includes(param.id);
            });
        }
    }

    return inputs;
};


export const getWorkflowOutputs = (workflowDefinition: WorkflowResourceDefinition) => {
    if (!workflowDefinition) { return undefined; }
    return getWorkflow(workflowDefinition)
        ? getWorkflow(workflowDefinition)!.outputs
        : undefined;
};

export const getInputLabel = (input: CommandInputParameter) => {
    return `${input.label || input.id.split('/').pop()}`;
};

export const getIOParamId = (input: CommandInputParameter | CommandOutputParameter) => {
    return `${input.id.split('/').pop()}`;
};

export const isRequiredInput = ({ type }: CommandInputParameter) => {
    if (type instanceof Array) {
        for (const t of type) {
            if (t === CWLType.NULL) {
                return false;
            }
        }
    }
    return true;
};

export const isPrimitiveOfType = (input: GenericCommandInputParameter<any, any>, type: CWLType) =>
    input.type instanceof Array
        ? input.type.indexOf(type) > -1
        : input.type === type;

export const isArrayOfType = (input: GenericCommandInputParameter<any, any>, type: CWLType) =>
    input.type instanceof Array
        ? (input.type.filter(t => typeof t === 'object' &&
            t.type === 'array' &&
            t.items === type).length > 0)
        : (typeof input.type === 'object' &&
            input.type.type === 'array' &&
            input.type.items === type);

export const getEnumType = (input: GenericCommandInputParameter<any, any>) => {
    if (input.type instanceof Array) {
        const f = input.type.filter(t => typeof t === 'object' &&
            !(t instanceof Array) &&
            t.type === 'enum');
        if (f.length > 0) {
            return f[0];
        }
    } else {
        if ((typeof input.type === 'object' &&
            !(input.type instanceof Array) &&
            input.type.type === 'enum')) {
            return input.type;
        }
    }
    return null;
};

export const isSecret = (input: GenericCommandInputParameter<any, any>) =>
    (typeof input.value === 'object') && input.value.$include?.startsWith("/secrets/");

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
