// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Stepper, Step, StepLabel, StepContent } from '@material-ui/core';
import { RunProcessFirstStepDataProps, RunProcessFirstStepActionProps, RunProcessFirstStep } from '~/views/run-process-panel/run-process-first-step';
import { RunProcessSecondStepDataProps, RunProcessSecondStepActionProps, RunProcessSecondStepForm } from '~/views/run-process-panel/run-process-second-step';

export type RunProcessPanelRootDataProps = {
    currentStep: number;
} & RunProcessFirstStepDataProps & RunProcessSecondStepDataProps;

export type RunProcessPanelRootActionProps = RunProcessFirstStepActionProps & RunProcessSecondStepActionProps;

type RunProcessPanelRootProps = RunProcessPanelRootDataProps & RunProcessPanelRootActionProps;

export const RunProcessPanelRoot = ({ currentStep, onSearch, onSetStep, onRunProcess, onSetWorkflow, workflows, selectedWorkflow }: RunProcessPanelRootProps) =>
    <Stepper activeStep={currentStep} orientation="vertical" elevation={2}>
        <Step>
            <StepLabel>Choose a workflow</StepLabel>
            <StepContent>
                <RunProcessFirstStep 
                    workflows={workflows}
                    selectedWorkflow={selectedWorkflow}
                    onSearch={onSearch}
                    onSetStep={onSetStep} 
                    onSetWorkflow={onSetWorkflow} />
            </StepContent>
        </Step>
        <Step>
            <StepLabel>Select inputs</StepLabel>
            <StepContent>
                <RunProcessSecondStepForm />
                {/* <RunProcessSecondStep onSetStep={onSetStep} onRunProcess={onRunProcess} /> */}
            </StepContent>
        </Step>
    </Stepper>;