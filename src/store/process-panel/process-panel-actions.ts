// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "common/unionize";
import { getRawOutputs, loadProcess } from 'store/processes/processes-actions';
import { Dispatch } from 'redux';
import { ProcessStatus } from 'store/processes/process';
import { RootState } from 'store/store';
import { ServiceRepository } from "services/services";
import { navigateTo, navigateToWorkflows } from 'store/navigation/navigation-action';
import { snackbarActions } from 'store/snackbar/snackbar-actions';
import { SnackbarKind } from '../snackbar/snackbar-actions';
import { showWorkflowDetails } from 'store/workflow-panel/workflow-panel-actions';
import { loadSubprocessPanel } from "../subprocess-panel/subprocess-panel-actions";
import { initProcessLogsPanel, processLogsPanelActions } from "store/process-logs-panel/process-logs-panel-actions";
import { CollectionFile } from "models/collection-file";
import { ContainerRequestResource } from "models/container-request";

export const processPanelActions = unionize({
    SET_PROCESS_PANEL_CONTAINER_REQUEST_UUID: ofType<string>(),
    SET_PROCESS_PANEL_FILTERS: ofType<string[]>(),
    TOGGLE_PROCESS_PANEL_FILTER: ofType<string>(),
});

export type ProcessPanelAction = UnionOf<typeof processPanelActions>;

export const toggleProcessPanelFilter = processPanelActions.TOGGLE_PROCESS_PANEL_FILTER;

export const loadProcessPanel = (uuid: string) =>
    async (dispatch: Dispatch) => {
        dispatch(processLogsPanelActions.RESET_PROCESS_LOGS_PANEL());
        dispatch<ProcessPanelAction>(processPanelActions.SET_PROCESS_PANEL_CONTAINER_REQUEST_UUID(uuid));
        await dispatch<any>(loadProcess(uuid));
        dispatch(initProcessPanelFilters);
        dispatch<any>(initProcessLogsPanel(uuid));
        dispatch<any>(loadSubprocessPanel());
    };

export const navigateToOutput = (uuid: string) =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        try {
            await services.collectionService.get(uuid);
            dispatch<any>(navigateTo(uuid));
        } catch {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'This collection does not exists!', hideDuration: 2000, kind: SnackbarKind.ERROR }));
        }
    };

export const loadOutputs = (containerRequest: ContainerRequestResource, setOutputs) =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const noOutputs = {rawOutputs: {}};
        if (!containerRequest.outputUuid) {setOutputs(noOutputs); return;};
        try {
            const propsOutputs = getRawOutputs(containerRequest);
            const filesPromise = services.collectionService.files(containerRequest.outputUuid);
            const collectionPromise = services.collectionService.get(containerRequest.outputUuid);
            const [files, collection] = await Promise.all([filesPromise, collectionPromise]);

            // If has propsOutput, skip fetching cwl.output.json
            if (propsOutputs !== undefined) {
                setOutputs({rawOutputs: propsOutputs, pdh: collection.portableDataHash});
            } else {
                // Fetch outputs from keep
                const outputFile = files.find((file) => file.name === 'cwl.output.json') as CollectionFile | undefined;
                let outputData = outputFile ? await services.collectionService.getFileContents(outputFile) : undefined;
                if (outputData && (outputData = JSON.parse(outputData)) && collection.portableDataHash) {
                    setOutputs({
                        rawOutputs: outputData,
                        pdh: collection.portableDataHash,
                    });
                } else {
                    setOutputs(noOutputs);
                }
            }
        } catch {
            setOutputs(noOutputs);
        }
    };

export const openWorkflow = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch<any>(navigateToWorkflows);
        dispatch<any>(showWorkflowDetails(uuid));
    };

export const initProcessPanelFilters = processPanelActions.SET_PROCESS_PANEL_FILTERS([
    ProcessStatus.QUEUED,
    ProcessStatus.COMPLETED,
    ProcessStatus.FAILED,
    ProcessStatus.RUNNING,
    ProcessStatus.ONHOLD,
    ProcessStatus.FAILING,
    ProcessStatus.WARNING,
    ProcessStatus.CANCELLED
]);
