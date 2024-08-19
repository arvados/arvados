// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { OutputDetails, ProcessPanel } from "store/process-panel/process-panel";
import { ProcessPanelAction, processPanelActions } from "store/process-panel/process-panel-actions";

const initialState: ProcessPanel = {
    containerRequestUuid: "",
    filters: {},
    inputRaw: null,
    inputParams: null,
    outputData: null,
    nodeInfo: null,
    outputDefinitions: [],
    outputParams: null,
    usageReport: null,
};

export type OutputDataUpdate = {
    uuid: string;
    payload: OutputDetails;
};

export const processPanelReducer = (state = initialState, action: ProcessPanelAction): ProcessPanel =>
    processPanelActions.match(action, {
        RESET_PROCESS_PANEL: () => initialState,
        SET_PROCESS_PANEL_CONTAINER_REQUEST_UUID: containerRequestUuid => ({
            ...state,
            containerRequestUuid,
        }),
        SET_PROCESS_PANEL_FILTERS: statuses => {
            const filters = statuses.reduce((filters, status) => ({ ...filters, [status]: true }), {});
            return { ...state, filters };
        },
        TOGGLE_PROCESS_PANEL_FILTER: status => {
            const filters = { ...state.filters, [status]: !state.filters[status] };
            return { ...state, filters };
        },
        SET_INPUT_RAW: inputRaw => {
            // Since mounts can disappear and reappear, only set inputs
            //   if current state is null or new inputs has content
            if (state.inputRaw === null || (inputRaw && Object.keys(inputRaw).length)) {
                return { ...state, inputRaw };
            } else {
                return state;
            }
        },
        SET_INPUT_PARAMS: inputParams => {
            // Since mounts can disappear and reappear, only set inputs
            //   if current state is null or new inputs has content
            if (state.inputParams === null || (inputParams && inputParams.length)) {
                return { ...state, inputParams };
            } else {
                return state;
            }
        },
        SET_OUTPUT_DATA: (update: OutputDataUpdate) => {
            //never set output to {} unless initializing
            if (state.outputData?.raw && Object.keys(state.outputData?.raw).length && state.containerRequestUuid === update.uuid) {
                return state;
            }
            return { ...state, outputData: update.payload };
        },
        SET_NODE_INFO: ({ nodeInfo }) => {
            return { ...state, nodeInfo };
        },
        SET_OUTPUT_DEFINITIONS: outputDefinitions => {
            // Set output definitions is only additive to avoid clearing when mounts go temporarily missing
            if (outputDefinitions.length) {
                return { ...state, outputDefinitions };
            } else {
                return state;
            }
        },
        SET_OUTPUT_PARAMS: outputParams => {
            return { ...state, outputParams };
        },
        SET_USAGE_REPORT: ({ usageReport }) => {
            return { ...state, usageReport };
        },
	SET_CONTAINER_STATUS: (containerStatus) => {
            return { ...state, containerStatus };
	},
        default: () => state,
    });
