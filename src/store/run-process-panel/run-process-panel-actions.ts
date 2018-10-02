// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { unionize, ofType, UnionOf } from "~/common/unionize";
import { ServiceRepository } from "~/services/services";
import { RootState } from '~/store/store';
import { WorkflowResource } from '~/models/workflow';

export const runProcessPanelActions = unionize({
    SET_CURRENT_STEP: ofType<number>(),
    SET_WORKFLOWS: ofType<WorkflowResource[]>(),
    SET_SELECTED_WORKFLOW: ofType<WorkflowResource>(),
    SEARCH_WORKFLOWS: ofType<string>()
});

export interface RunProcessSecondStepDataFormProps {
    name: string;
    description: string;
}

export const RUN_PROCESS_SECOND_STEP_FORM_NAME = 'runProcessSecondStepFormName';

export type RunProcessPanelAction = UnionOf<typeof runProcessPanelActions>;

export const loadRunProcessPanel = () =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        try {
            const response = await services.workflowService.list();
            dispatch(runProcessPanelActions.SET_WORKFLOWS(response.items));
        } catch (e) {
            return;
        }
    };

export const setWorkflow = (workflow: WorkflowResource) => 
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        dispatch(runProcessPanelActions.SET_SELECTED_WORKFLOW(workflow));
    };

export const goToStep = (step: number) => runProcessPanelActions.SET_CURRENT_STEP(step);

export const searchWorkflows = (term: string) => runProcessPanelActions.SEARCH_WORKFLOWS(term);