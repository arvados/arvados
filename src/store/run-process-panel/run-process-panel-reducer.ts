// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { RunProcessPanelAction, runProcessPanelActions } from '~/store/run-process-panel/run-process-panel-actions';

interface RunProcessPanel {
    currentStep: number;
}

const initialState: RunProcessPanel = {
    currentStep: 0
};

export const runProcessPanelReducer = (state = initialState, action: RunProcessPanelAction): RunProcessPanel =>
    runProcessPanelActions.match(action, {
        CHANGE_STEP: currentStep => ({ ...state, currentStep }),
        default: () => state
    });