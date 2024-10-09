// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "common/unionize";
import { getInputs,
         getOutputParameters,
         getRawInputs,
         getRawOutputs
} from "store/processes/processes-actions";
import { Dispatch } from "redux";
import { Process, ProcessStatus } from "store/processes/process";
import { RootState } from "store/store";
import { ServiceRepository } from "services/services";
import { navigateTo } from "store/navigation/navigation-action";
import { snackbarActions } from "store/snackbar/snackbar-actions";
import { SnackbarKind } from "../snackbar/snackbar-actions";
import { loadSubprocessPanel, subprocessPanelActions } from "../subprocess-panel/subprocess-panel-actions";
import { initProcessLogsPanel, processLogsPanelActions } from "store/process-logs-panel/process-logs-panel-actions";
import { CollectionFile } from "models/collection-file";
import { ContainerRequestResource } from "models/container-request";
import { CommandOutputParameter } from "cwlts/mappings/v1.0/CommandOutputParameter";
import { CommandInputParameter, getIOParamId, WorkflowInputsData } from "models/workflow";
import { getIOParamDisplayValue, ProcessIOParameter } from "views/process-panel/process-io-card";
import { OutputDetails, NodeInstanceType, NodeInfo, UsageReport } from "./process-panel";
import { AuthState } from "store/auth/auth-reducer";
import { ContextMenuResource } from "store/context-menu/context-menu-actions";
import { OutputDataUpdate } from "./process-panel-reducer";
import { updateResources } from "store/resources/resources-actions";
import { ContainerResource } from "models/container";
import { FilterBuilder } from "services/api/filter-builder";

export const processPanelActions = unionize({
    RESET_PROCESS_PANEL: ofType<{}>(),
    SET_PROCESS_PANEL_CONTAINER_REQUEST_UUID: ofType<string>(),
    SET_PROCESS_PANEL_FILTERS: ofType<string[]>(),
    TOGGLE_PROCESS_PANEL_FILTER: ofType<string>(),
    SET_INPUT_RAW: ofType<WorkflowInputsData | null>(),
    SET_INPUT_PARAMS: ofType<ProcessIOParameter[] | null>(),
    SET_OUTPUT_DATA: ofType<OutputDataUpdate | null>(),
    SET_OUTPUT_DEFINITIONS: ofType<CommandOutputParameter[]>(),
    SET_OUTPUT_PARAMS: ofType<ProcessIOParameter[] | null>(),
    SET_NODE_INFO: ofType<NodeInfo>(),
    SET_USAGE_REPORT: ofType<UsageReport>(),
});

export type ProcessPanelAction = UnionOf<typeof processPanelActions>;

export const toggleProcessPanelFilter = processPanelActions.TOGGLE_PROCESS_PANEL_FILTER;

export const loadProcess =
    (containerRequestUuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository): Promise<Process | undefined> => {
        let containerRequest: ContainerRequestResource | undefined = undefined;
        let container: ContainerResource | undefined = undefined;

        try {
            const containerRequestResult = await services.groupsService.contents(
                '', {
                    filters: new FilterBuilder().addIsA('uuid', 'arvados#containerRequest')
                                                 .addEqual('uuid', containerRequestUuid)
                                                 .getFilters(),
                    include: ["container_uuid"]
            });
            if (containerRequestResult.items.length === 1) {
                containerRequest = containerRequestResult.items[0] as ContainerRequestResource;
                dispatch<any>(updateResources(containerRequestResult.items));

                if (containerRequestResult.included?.length === 1) {
                    container = containerRequestResult.included[0] as ContainerResource;
                    dispatch<any>(updateResources(containerRequestResult.included));
                }
            }
        } catch (e) {
            if (!containerRequest) {
                dispatch(
                    snackbarActions.OPEN_SNACKBAR({
                        message: e.message,
                        hideDuration: 2000,
                        kind: SnackbarKind.ERROR,
                    })
                );
            }
        }

        if (!containerRequest) {
            return undefined;
        }

        if (!container && containerRequest.containerUuid) {
            // Get the container the old fashioned way
            try {
                container = await services.containerService.get(containerRequest.containerUuid, false);
                dispatch<any>(updateResources([container]));
            } catch {}
        }

        if (container && container.runtimeUserUuid) {
            try {
                const runtimeUser = await services.userService.get(container.runtimeUserUuid, false);
                dispatch<any>(updateResources([runtimeUser]));
            } catch {}
        }

        if (containerRequest.outputUuid) {
            try {
                const collection = await services.collectionService.get(containerRequest.outputUuid, false);
                dispatch<any>(updateResources([collection]));
            } catch {}
        }

        return { containerRequest, container };
    };

export const loadProcessPanel = (uuid: string) => async (dispatch: Dispatch, getState: () => RootState) => {
    // Reset subprocess data explorer if navigating to new process
    //  Avoids resetting pagination when refreshing same process
    if (getState().processPanel.containerRequestUuid !== uuid) {
        dispatch(subprocessPanelActions.CLEAR());
    }
    dispatch(processPanelActions.RESET_PROCESS_PANEL());
    dispatch(processLogsPanelActions.RESET_PROCESS_LOGS_PANEL());
    dispatch<ProcessPanelAction>(processPanelActions.SET_PROCESS_PANEL_CONTAINER_REQUEST_UUID(uuid));
    await dispatch<any>(loadProcess(uuid));
    dispatch(initProcessPanelFilters);
    dispatch<any>(initProcessLogsPanel(uuid));
    dispatch<any>(loadSubprocessPanel());
};

export const navigateToOutput = (resource: ContextMenuResource | ContainerRequestResource) => async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
    try {
        await services.collectionService.get(resource.outputUuid || '');
        dispatch<any>(navigateTo(resource.outputUuid || ''));
    } catch {
        dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Output collection was trashed or deleted.", hideDuration: 4000, kind: SnackbarKind.WARNING }));
    }
};

export const loadInputs =
    (containerRequest: ContainerRequestResource) => async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        dispatch<ProcessPanelAction>(processPanelActions.SET_INPUT_RAW(getRawInputs(containerRequest)));
        dispatch<ProcessPanelAction>(processPanelActions.SET_INPUT_PARAMS(formatInputData(getInputs(containerRequest), getState().auth)));
    };

export const loadOutputs =
    (containerRequest: ContainerRequestResource) => async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const noOutputs: OutputDetails = { raw: {} };

        if (!containerRequest.outputUuid) {
            dispatch<ProcessPanelAction>(processPanelActions.SET_OUTPUT_DATA({
                uuid: containerRequest.uuid,
                payload: noOutputs
            }));
            return;
        }
        try {
            const propsOutputs = getRawOutputs(containerRequest);
            const filesPromise = services.collectionService.files(containerRequest.outputUuid);
            const collectionPromise = services.collectionService.get(containerRequest.outputUuid);
            const [files, collection] = await Promise.all([filesPromise, collectionPromise]);

            // If has propsOutput, skip fetching cwl.output.json
            if (propsOutputs !== undefined) {
                dispatch<ProcessPanelAction>(
                    processPanelActions.SET_OUTPUT_DATA({
                        uuid: containerRequest.uuid,
                        payload: {
                            raw: propsOutputs,
                            pdh: collection.portableDataHash,
                        },
                    })
                );
            } else {
                // Fetch outputs from keep
                const outputFile = files.find(file => file.name === "cwl.output.json") as CollectionFile | undefined;
                let outputData = outputFile ? await services.collectionService.getFileContents(outputFile) : undefined;
                if (outputData && (outputData = JSON.parse(outputData)) && collection.portableDataHash) {
                    dispatch<ProcessPanelAction>(
                        processPanelActions.SET_OUTPUT_DATA({
                            uuid: containerRequest.uuid,
                            payload: {
                                raw: outputData,
                                pdh: collection.portableDataHash,
                            },
                        })
                    );
                } else {
                    dispatch<ProcessPanelAction>(processPanelActions.SET_OUTPUT_DATA({ uuid: containerRequest.uuid, payload: noOutputs }));
                }
            }
        } catch {
            dispatch<ProcessPanelAction>(processPanelActions.SET_OUTPUT_DATA({ uuid: containerRequest.uuid, payload: noOutputs }));
        }
    };

export const loadNodeJson =
    (containerRequest: ContainerRequestResource) => async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const noLog = { nodeInfo: null };
        if (!containerRequest.logUuid) {
            dispatch<ProcessPanelAction>(processPanelActions.SET_NODE_INFO(noLog));
            return;
        }
        try {
            const filesPromise = services.collectionService.files(containerRequest.logUuid);
            const collectionPromise = services.collectionService.get(containerRequest.logUuid);
            const [files] = await Promise.all([filesPromise, collectionPromise]);

            // Fetch node.json from keep
            const nodeFile = files.find(file => file.name === "node.json") as CollectionFile | undefined;
            let nodeData = nodeFile ? await services.collectionService.getFileContents(nodeFile) : undefined;
            if (nodeData && (nodeData = JSON.parse(nodeData))) {
                dispatch<ProcessPanelAction>(
                    processPanelActions.SET_NODE_INFO({
                        nodeInfo: nodeData as NodeInstanceType,
                    })
                );
            } else {
                dispatch<ProcessPanelAction>(processPanelActions.SET_NODE_INFO(noLog));
            }

            const usageReportFile = files.find(file => file.name === "usage_report.html") as CollectionFile | null;
            dispatch<ProcessPanelAction>(processPanelActions.SET_USAGE_REPORT({ usageReport: usageReportFile }));
        } catch {
            dispatch<ProcessPanelAction>(processPanelActions.SET_NODE_INFO(noLog));
            dispatch<ProcessPanelAction>(processPanelActions.SET_USAGE_REPORT({ usageReport: null }));
        }
    };

export const loadOutputDefinitions =
    (containerRequest: ContainerRequestResource) => async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        if (containerRequest && containerRequest.mounts) {
            dispatch<ProcessPanelAction>(processPanelActions.SET_OUTPUT_DEFINITIONS(getOutputParameters(containerRequest)));
        }
    };

export const updateOutputParams = () => async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
    const outputDefinitions = getState().processPanel.outputDefinitions;
    const outputData = getState().processPanel.outputData;

    if (outputData && outputData.raw) {
        dispatch<ProcessPanelAction>(
            processPanelActions.SET_OUTPUT_PARAMS(formatOutputData(outputDefinitions, outputData.raw, outputData.pdh, getState().auth))
        );
    }
};

export const openWorkflow = (uuid: string) => (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    dispatch<any>(navigateTo(uuid));
};

export const initProcessPanelFilters = processPanelActions.SET_PROCESS_PANEL_FILTERS([
    ProcessStatus.QUEUED,
    ProcessStatus.COMPLETED,
    ProcessStatus.FAILED,
    ProcessStatus.RUNNING,
    ProcessStatus.ONHOLD,
    ProcessStatus.FAILING,
    ProcessStatus.WARNING,
    ProcessStatus.CANCELLED,
]);

export const formatInputData = (inputs: CommandInputParameter[], auth: AuthState): ProcessIOParameter[] => {
    return inputs.flatMap((input): ProcessIOParameter[] => {
        const processValues = getIOParamDisplayValue(auth, input);
        return processValues.map((thisValue, i) => ({
            id: i === 0 ? getIOParamId(input) : "",
            label: i === 0 ? input.label || "" : "",
            value: thisValue,
        }));
    });
};

export const formatOutputData = (
    definitions: CommandOutputParameter[],
    values: any,
    pdh: string | undefined,
    auth: AuthState
): ProcessIOParameter[] => {
    return definitions.flatMap((output): ProcessIOParameter[] => {
        const processValues = getIOParamDisplayValue(auth, Object.assign(output, { value: values[getIOParamId(output)] || [] }), pdh);
        return processValues.map((thisValue, i) => ({
            id: i === 0 ? getIOParamId(output) : "",
            label: i === 0 ? output.label || "" : "",
            value: thisValue,
        }));
    });
};
