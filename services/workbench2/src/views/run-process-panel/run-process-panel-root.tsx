// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { Stepper, Step, StepLabel, StepContent } from '@mui/material';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { RunProcessFirstStepDataProps, RunProcessFirstStepActionProps, RunProcessFirstStep } from 'views/run-process-panel/run-process-first-step';
import { RunProcessSecondStepForm } from './run-process-second-step';

export type RunProcessPanelRootDataProps = {
    currentStep: number;
} & RunProcessFirstStepDataProps;

export type RunProcessPanelRootActionProps = RunProcessFirstStepActionProps & {
    runProcess: () => void;
};

type RunProcessPanelRootProps = RunProcessPanelRootDataProps & RunProcessPanelRootActionProps;

type CssRules = 'stepper';

const styles: CustomStyleRulesCallback<CssRules> = theme => ({
    stepper: {
        overflow: "scroll",
        width: "100%",
    }
});

export const RunProcessPanelRoot = withStyles(styles)(
    ({ runProcess, currentStep, onSearch, onSetStep, onSetWorkflow, workflows, selectedWorkflow, classes }: WithStyles<CssRules> & RunProcessPanelRootProps) =>
        <Stepper activeStep={currentStep} orientation="vertical" elevation={2} className={classes.stepper}>
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
                    <RunProcessSecondStepForm
                        goBack={() => onSetStep(0)}
                        runProcess={runProcess} />
                </StepContent>
            </Step>
        </Stepper>);
