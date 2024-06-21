// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { RootState } from "store/store";
import { ServiceRepository } from "services/services";
import { updateResources } from "store/resources/resources-actions";
import { Process } from "./process";
import { dialogActions } from "store/dialog/dialog-actions";
import { snackbarActions, SnackbarKind } from "store/snackbar/snackbar-actions";
import { projectPanelDataActions } from "store/project-panel/project-panel-action-bind";
import { navigateToRunProcess } from "store/navigation/navigation-action";
import { goToStep, runProcessPanelActions } from "store/run-process-panel/run-process-panel-actions";
import { getResource } from "store/resources/resources";
import { initialize } from "redux-form";
import { RUN_PROCESS_BASIC_FORM, RunProcessBasicFormData } from "views/run-process-panel/run-process-basic-form";
import { RunProcessAdvancedFormData, RUN_PROCESS_ADVANCED_FORM } from "views/run-process-panel/run-process-advanced-form";
import { MOUNT_PATH_CWL_WORKFLOW, MOUNT_PATH_CWL_INPUT } from "models/process";
import { CommandInputParameter, getWorkflow, getWorkflowInputs, getWorkflowOutputs, WorkflowInputsData } from "models/workflow";
import { ProjectResource } from "models/project";
import { UserResource } from "models/user";
import { CommandOutputParameter } from "cwlts/mappings/v1.0/CommandOutputParameter";
import { ContainerResource } from "models/container";
import { ContainerRequestResource, ContainerRequestState } from "models/container-request";
import { FilterBuilder } from "services/api/filter-builder";
import { selectedToArray } from "components/multiselect-toolbar/MultiselectToolbar";
import { Resource, ResourceKind } from "models/resource";
import { ContextMenuResource } from "store/context-menu/context-menu-actions";
import { CommonResourceServiceError, getCommonResourceServiceError } from "services/common-service/common-resource-service";

export const loadProcess =
    (containerRequestUuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository): Promise<Process | undefined> => {
        let containerRequest: ContainerRequestResource | undefined = undefined;
        try {
            containerRequest = await services.containerRequestService.get(containerRequestUuid);
            dispatch<any>(updateResources([containerRequest]));
        } catch {
            return undefined;
        }

        if (containerRequest.outputUuid) {
            try {
                const collection = await services.collectionService.get(containerRequest.outputUuid, false);
                dispatch<any>(updateResources([collection]));
            } catch {}
        }

        if (containerRequest.containerUuid) {
            let container: ContainerResource | undefined = undefined;
            try {
                container = await services.containerService.get(containerRequest.containerUuid, false);
                dispatch<any>(updateResources([container]));
            } catch {}

            try {
                if (container && container.runtimeUserUuid) {
                    const runtimeUser = await services.userService.get(container.runtimeUserUuid, false);
                    dispatch<any>(updateResources([runtimeUser]));
                }
            } catch {}

            return { containerRequest, container };
        }
        return { containerRequest };
    };

export const loadContainers =
    (containerUuids: string[], loadMounts: boolean = true) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        let args: any = {
            filters: new FilterBuilder().addIn("uuid", containerUuids).getFilters(),
            limit: containerUuids.length,
        };
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
    "subrequests_cost",
    "uuid",
];

export const cancelRunningWorkflow = (uuid: string) => async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    try {
        const process = await services.containerRequestService.update(uuid, { priority: 0 });
        dispatch<any>(updateResources([process]));
        if (process.containerUuid) {
            const container = await services.containerService.get(process.containerUuid, false);
            dispatch<any>(updateResources([container]));
        }
        return process;
    } catch (e) {
        throw new Error("Could not cancel the process.");
    }
};

export const resumeOnHoldWorkflow = (uuid: string) => async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    try {
        const process = await services.containerRequestService.update(uuid, { priority: 500 });
        dispatch<any>(updateResources([process]));
        if (process.containerUuid) {
            const container = await services.containerService.get(process.containerUuid, false);
            dispatch<any>(updateResources([container]));
        }
        return process;
    } catch (e) {
        throw new Error("Could not resume the process.");
    }
};

export const startWorkflow = (uuid: string) => async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    try {
        const process = await services.containerRequestService.update(uuid, { state: ContainerRequestState.COMMITTED });
        if (process) {
            dispatch<any>(updateResources([process]));
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Process started", hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
        } else {
            dispatch<any>(snackbarActions.OPEN_SNACKBAR({ message: `Failed to start process`, kind: SnackbarKind.ERROR }));
        }
    } catch (e) {
        dispatch<any>(snackbarActions.OPEN_SNACKBAR({ message: `Failed to start process`, kind: SnackbarKind.ERROR }));
    }
};

export const reRunProcess =
    (processUuid: string, workflowUuid: string) => (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const process = getResource<any>(processUuid)(getState().resources);
        const workflows = getState().runProcessPanel.searchWorkflows;
        const workflow = workflows.find(workflow => workflow.uuid === workflowUuid);
        if (workflow && process) {
            const mainWf = getWorkflow(process.mounts[MOUNT_PATH_CWL_WORKFLOW]);
            if (mainWf) {
                mainWf.inputs = getInputs(process);
            }
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
                acr_container_image: process.containerImage,
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
    if (!data) {
        return undefined;
    }
    const mountInput = data.mounts?.[MOUNT_PATH_CWL_INPUT]?.content;
    const propsInput = data.properties?.cwl_input;
    if (!mountInput && !propsInput) {
        return {};
    }
    return mountInput || propsInput;
};

export const getInputs = (data: any): CommandInputParameter[] => {
    // Definitions from mounts are needed so we return early if missing
    if (!data || !data.mounts || !data.mounts[MOUNT_PATH_CWL_WORKFLOW]) {
        return [];
    }
    const content = getRawInputs(data) as any;
    // Only escape if content is falsy to allow displaying definitions if no inputs are present
    // (Don't check raw content length)
    if (!content) {
        return [];
    }

    const inputs = getWorkflowInputs(data.mounts[MOUNT_PATH_CWL_WORKFLOW].content);
    return inputs
        ? inputs.map((it: any) => ({
              type: it.type,
              id: it.id,
              label: it.label,
              default: content[it.id],
              value: content[it.id.split("/").pop()] || [],
              doc: it.doc,
          }))
        : [];
};

/*
 * Fetches raw outputs from containerRequest properties
 * Assumes containerRequest is loaded
 */
export const getRawOutputs = (data: any): CommandInputParameter[] | undefined => {
    if (!data || !data.properties || !data.properties.cwl_output) {
        return undefined;
    }
    return data.properties.cwl_output;
};

export type InputCollectionMount = {
    path: string;
    pdh: string;
};

export const getInputCollectionMounts = (data: any): InputCollectionMount[] => {
    if (!data || !data.mounts) {
        return [];
    }
    return Object.keys(data.mounts)
        .map(key => ({
            ...data.mounts[key],
            path: key,
        }))
        .filter(mount => mount.kind === "collection" && mount.portable_data_hash && mount.path)
        .map(mount => ({
            path: mount.path,
            pdh: mount.portable_data_hash,
        }));
};

export const getOutputParameters = (data: any): CommandOutputParameter[] => {
    if (!data || !data.mounts || !data.mounts[MOUNT_PATH_CWL_WORKFLOW]) {
        return [];
    }
    const outputs = getWorkflowOutputs(data.mounts[MOUNT_PATH_CWL_WORKFLOW].content);
    return outputs
        ? outputs.map((it: any) => ({
              type: it.type,
              id: it.id,
              label: it.label,
              doc: it.doc,
          }))
        : [];
};

export const openRemoveProcessDialog =
    (resource: ContextMenuResource, numOfProcesses: Number) => (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const confirmationText =
            numOfProcesses === 1
                ? "Are you sure you want to remove this process?"
                : `Are you sure you want to remove these ${numOfProcesses} processes?`;
        const titleText = numOfProcesses === 1 ? "Remove process permanently" : "Remove processes permanently";

        dispatch(
            dialogActions.OPEN_DIALOG({
                id: REMOVE_PROCESS_DIALOG,
                data: {
                    title: titleText,
                    text: confirmationText,
                    confirmButtonLabel: "Remove",
                    uuid: resource.uuid,
                    resource,
                },
            })
        );
    };

export const REMOVE_PROCESS_DIALOG = "removeProcessDialog";

export const removeProcessPermanently = (uuid: string) => async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    const resource = getState().dialog.removeProcessDialog.data.resource;
    const checkedList = getState().multiselect.checkedList;

    const uuidsToRemove: string[] = resource.fromContextMenu ? [resource.uuid] : selectedToArray(checkedList);

    //if no items in checkedlist, default to normal context menu behavior
    if (!uuidsToRemove.length) uuidsToRemove.push(uuid);

    const processesToRemove = uuidsToRemove
        .map(uuid => getResource(uuid)(getState().resources) as Resource)
        .filter(resource => resource.kind === ResourceKind.PROCESS);

    for (const process of processesToRemove) {
        try {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Removing ...", kind: SnackbarKind.INFO }));
            await services.containerRequestService.delete(process.uuid, false);
            dispatch(projectPanelDataActions.REQUEST_ITEMS());
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Removed.", hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
        } catch (e) {
            const error = getCommonResourceServiceError(e);
            if (error === CommonResourceServiceError.PERMISSION_ERROR_FORBIDDEN) {
                dispatch(snackbarActions.OPEN_SNACKBAR({ message: `Access denied`, hideDuration: 2000, kind: SnackbarKind.ERROR }));
            } else {
                dispatch(snackbarActions.OPEN_SNACKBAR({ message: `Deletion failed`, hideDuration: 2000, kind: SnackbarKind.ERROR }));
            }
        }
    }
};
