// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { unionize, ofType, UnionOf } from "~/common/unionize";
import { ServiceRepository } from "~/services/services";
import { RootState } from '~/store/store';
import { WorkflowResource } from '~/models/workflow';
import { getFormValues } from 'redux-form';
import { RUN_PROCESS_BASIC_FORM, RunProcessBasicFormData } from '~/views/run-process-panel/run-process-basic-form';
import { RUN_PROCESS_INPUTS_FORM } from '~/views/run-process-panel/run-process-inputs-form';
import { WorkflowInputsData } from '~/models/workflow';
import { createWorkflowMounts } from '~/models/process';
import { ContainerRequestState } from '~/models/container-request';
import { navigateToProcess } from '../navigation/navigation-action';
import { RunProcessAdvancedFormData, RUN_PROCESS_ADVANCED_FORM } from '~/views/run-process-panel/run-process-advanced-form';
import { isItemNotInProject, isProjectOrRunProcessRoute } from '~/store/projects/project-create-actions';
import { matchProjectRoute } from '~/routes/routes';

export const runProcessPanelActions = unionize({
    SET_PROCESS_OWNER_UUID: ofType<string>(),
    SET_CURRENT_STEP: ofType<number>(),
    SET_WORKFLOWS: ofType<WorkflowResource[]>(),
    SET_SELECTED_WORKFLOW: ofType<WorkflowResource>(),
    SEARCH_WORKFLOWS: ofType<string>(),
    RESET_RUN_PROCESS_PANEL: ofType<{}>(),
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
            dispatch(runProcessPanelActions.RESET_RUN_PROCESS_PANEL());
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

const isRunProcessRoute = ({ router }: RootState) => {
    const pathname = router.location ? router.location.pathname : '';
    const match = matchProjectRoute(pathname);
    return !!match;
};

export const runProcess = async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
    const state = getState();
    const basicForm = getFormValues(RUN_PROCESS_BASIC_FORM)(state) as RunProcessBasicFormData;
    const inputsForm = getFormValues(RUN_PROCESS_INPUTS_FORM)(state) as WorkflowInputsData;
    const advancedForm = getFormValues(RUN_PROCESS_ADVANCED_FORM)(state) as RunProcessAdvancedFormData;
    const userUuid = getState().auth.user!.uuid;
    const router = getState();
    const properties = getState().properties;
    const { processOwnerUuid, selectedWorkflow } = state.runProcessPanel;
    if (selectedWorkflow) {
        const newProcessData = {
            ownerUuid: isItemNotInProject(properties) || !isProjectOrRunProcessRoute(router) ? userUuid : processOwnerUuid,
            name: basicForm.name,
            description: basicForm.description,
            state: ContainerRequestState.COMMITTED,
            mounts: createWorkflowMounts(selectedWorkflow, normalizeInputKeys(inputsForm)),
            runtimeConstraints: {
                API: true,
                vcpus: 1,
                ram: 1073741824,
            },
            containerImage: 'arvados/jobs',
            cwd: '/var/spool/cwl',
            command: [
                'arvados-cwl-runner',
                '--local',
                '--api=containers',
                `--project-uuid=${processOwnerUuid}`,
                '/var/lib/cwl/workflow.json#main',
                '/var/lib/cwl/cwl.input.json'
            ],
            outputPath: '/var/spool/cwl',
            priority: 1,
            outputName: advancedForm && advancedForm.output ? advancedForm.output : undefined,
        };
        const newProcess = await services.containerRequestService.create(newProcessData);
        dispatch(navigateToProcess(newProcess.uuid));
    }
};

const normalizeInputKeys = (inputs: WorkflowInputsData): WorkflowInputsData =>
    Object.keys(inputs).reduce((normalizedInputs, key) => ({
        ...normalizedInputs,
        [key.split('/').slice(1).join('/')]: inputs[key],
    }), {});
export const searchWorkflows = (term: string) => runProcessPanelActions.SEARCH_WORKFLOWS(term);
