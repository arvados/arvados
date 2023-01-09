// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { RootState } from 'store/store';
import { ServiceRepository } from 'services/services';
import { updateResources } from 'store/resources/resources-actions';
import { Process } from './process';
import { dialogActions } from 'store/dialog/dialog-actions';
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { projectPanelActions } from 'store/project-panel/project-panel-action';
import { navigateToRunProcess } from 'store/navigation/navigation-action';
import { goToStep, runProcessPanelActions } from 'store/run-process-panel/run-process-panel-actions';
import { getResource } from 'store/resources/resources';
import { initialize } from "redux-form";
import { RUN_PROCESS_BASIC_FORM, RunProcessBasicFormData } from "views/run-process-panel/run-process-basic-form";
import { RunProcessAdvancedFormData, RUN_PROCESS_ADVANCED_FORM } from "views/run-process-panel/run-process-advanced-form";
import { MOUNT_PATH_CWL_WORKFLOW, MOUNT_PATH_CWL_INPUT } from 'models/process';
import { CommandInputParameter, getWorkflow, getWorkflowInputs, getWorkflowOutputs, WorkflowInputsData } from "models/workflow";
import { ProjectResource } from "models/project";
import { UserResource } from "models/user";
import { CommandOutputParameter } from "cwlts/mappings/v1.0/CommandOutputParameter";
import { ContainerRequestState } from "models/container-request";

export const loadProcess = (containerRequestUuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository): Promise<Process> => {
        const containerRequest = await services.containerRequestService.get(containerRequestUuid);
        dispatch<any>(updateResources([containerRequest]));

        if (containerRequest.outputUuid) {
            const collection = await services.collectionService.get(containerRequest.outputUuid);
            dispatch<any>(updateResources([collection]));
        }

        if (containerRequest.containerUuid) {
            const container = await services.containerService.get(containerRequest.containerUuid);
            dispatch<any>(updateResources([container]));
            if (container.runtimeUserUuid) {
                const runtimeUser = await services.userService.get(container.runtimeUserUuid);
                dispatch<any>(updateResources([runtimeUser]));
            }
            return { containerRequest, container };
        }
        return { containerRequest };
    };

export const loadContainers = (filters: string, loadMounts: boolean = true) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        let args: any = { filters };
        if (!loadMounts) {
            args.select = containerFieldsNoMounts;
        }
        const { items } = await services.containerService.list(args);
        dispatch<any>(updateResources(items));
        return items;
    };

// Until the api supports unselecting fields, we need a list of all other fields to omit mounts
const containerFieldsNoMounts = [
    "auth_uuid",
    "command",
    "container_image",
    "cost",
    "created_at",
    "cwd",
    "environment",
    "etag",
    "exit_code",
    "finished_at",
    "gateway_address",
    "href",
    "interactive_session_started",
    "kind",
    "lock_count",
    "locked_by_uuid",
    "log",
    "modified_at",
    "modified_by_client_uuid",
    "modified_by_user_uuid",
    "output_path",
    "output_properties",
    "output_storage_classes",
    "output",
    "owner_uuid",
    "priority",
    "progress",
    "runtime_auth_scopes",
    "runtime_constraints",
    "runtime_status",
    "runtime_user_uuid",
    "scheduling_parameters",
    "started_at",
    "state",
    "uuid",
]

export const cancelRunningWorkflow = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        try {
            const process = await services.containerRequestService.update(uuid, { priority: 0 });
            return process;
        } catch (e) {
            throw new Error('Could not cancel the process.');
        }
    };

export const startWorkflow = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        try {
            const process = await services.containerRequestService.update(uuid, { priority: 1, state: ContainerRequestState.COMMITTED });
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Process started', hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
            return process;
        } catch (e) {
            throw new Error('Could not cancel the process.');
        }
    };

export const reRunProcess = (processUuid: string, workflowUuid: string) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const process = getResource<any>(processUuid)(getState().resources);
        const workflows = getState().runProcessPanel.searchWorkflows;
        const workflow = workflows.find(workflow => workflow.uuid === workflowUuid);
        if (workflow && process) {
            const mainWf = getWorkflow(process.mounts[MOUNT_PATH_CWL_WORKFLOW]);
            if (mainWf) { mainWf.inputs = getInputs(process); }
            const stringifiedDefinition = JSON.stringify(process.mounts[MOUNT_PATH_CWL_WORKFLOW].content);
            const newWorkflow = { ...workflow, definition: stringifiedDefinition };

            const owner = getResource<ProjectResource | UserResource>(workflow.ownerUuid)(getState().resources);
            const basicInitialData: RunProcessBasicFormData = { name: `Copy of: ${process.name}`, description: process.description, owner };
            dispatch<any>(initialize(RUN_PROCESS_BASIC_FORM, basicInitialData));

            const advancedInitialData: RunProcessAdvancedFormData = {
                output: process.outputName,
                runtime: process.schedulingParameters.max_run_time,
                ram: process.runtimeConstraints.ram,
                vcpus: process.runtimeConstraints.vcpus,
                keep_cache_ram: process.runtimeConstraints.keep_cache_ram,
                acr_container_image: process.containerImage
            };
            dispatch<any>(initialize(RUN_PROCESS_ADVANCED_FORM, advancedInitialData));

            dispatch<any>(navigateToRunProcess);
            dispatch<any>(goToStep(1));
            dispatch(runProcessPanelActions.SET_STEP_CHANGED(true));
            dispatch(runProcessPanelActions.SET_SELECTED_WORKFLOW(newWorkflow));
        } else {
            dispatch<any>(snackbarActions.OPEN_SNACKBAR({ message: `You can't re-run this process`, kind: SnackbarKind.ERROR }));
        }
    };

/*
 * Fetches raw inputs from containerRequest mounts with fallback to properties
 * Returns undefined if containerRequest not loaded
 * Returns {} if inputs not found in mounts or props
 */
export const getRawInputs = (data: any): WorkflowInputsData | undefined => {
    if (!data) { return undefined; }
    const mountInput = data.mounts?.[MOUNT_PATH_CWL_INPUT]?.content;
    const propsInput = data.properties?.cwl_input;
    if (!mountInput && !propsInput) { return {}; }
    return (mountInput || propsInput);
}

export const getInputs = (data: any): CommandInputParameter[] => {
    // Definitions from mounts are needed so we return early if missing
    if (!data || !data.mounts || !data.mounts[MOUNT_PATH_CWL_WORKFLOW]) { return []; }
    const content  = getRawInputs(data) as any;
    // Only escape if content is falsy to allow displaying definitions if no inputs are present
    // (Don't check raw content length)
    if (!content) { return []; }

    const inputs = getWorkflowInputs(data.mounts[MOUNT_PATH_CWL_WORKFLOW].content);
    return inputs ? inputs.map(
        (it: any) => (
            {
                type: it.type,
                id: it.id,
                label: it.label,
                default: content[it.id],
                value: content[it.id.split('/').pop()] || [],
                doc: it.doc
            }
        )
    ) : [];
};

/*
 * Fetches raw outputs from containerRequest properties
 * Assumes containerRequest is loaded
 */
export const getRawOutputs = (data: any): CommandInputParameter[] | undefined => {
    if (!data || !data.properties || !data.properties.cwl_output) { return undefined; }
    return (data.properties.cwl_output);
}

export type InputCollectionMount = {
    path: string;
    pdh: string;
}

export const getInputCollectionMounts = (data: any): InputCollectionMount[] => {
    if (!data || !data.mounts) { return []; }
    return Object.keys(data.mounts)
        .map(key => ({
            ...data.mounts[key],
            path: key,
        }))
        .filter(mount => mount.kind === 'collection' &&
                mount.portable_data_hash &&
                mount.path)
        .map(mount => ({
            path: mount.path,
            pdh: mount.portable_data_hash,
        }));
};

export const getOutputParameters = (data: any): CommandOutputParameter[] => {
    if (!data || !data.mounts || !data.mounts[MOUNT_PATH_CWL_WORKFLOW]) { return []; }
    const outputs = getWorkflowOutputs(data.mounts[MOUNT_PATH_CWL_WORKFLOW].content);
    return outputs ? outputs.map(
        (it: any) => (
            {
                type: it.type,
                id: it.id,
                label: it.label,
                doc: it.doc
            }
        )
    ) : [];
};

export const openRemoveProcessDialog = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(dialogActions.OPEN_DIALOG({
            id: REMOVE_PROCESS_DIALOG,
            data: {
                title: 'Remove process permanently',
                text: 'Are you sure you want to remove this process?',
                confirmButtonLabel: 'Remove',
                uuid
            }
        }));
    };

export const REMOVE_PROCESS_DIALOG = 'removeProcessDialog';

export const removeProcessPermanently = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Removing ...', kind: SnackbarKind.INFO }));
        await services.containerRequestService.delete(uuid);
        dispatch(projectPanelActions.REQUEST_ITEMS());
        dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Removed.', hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
    };
