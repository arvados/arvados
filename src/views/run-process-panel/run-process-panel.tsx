// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { connect } from 'react-redux';
import { RootState } from '~/store/store';
import { RunProcessPanelRootDataProps, RunProcessPanelRootActionProps, RunProcessPanelRoot } from '~/views/run-process-panel/run-process-panel-root';
import { goToStep } from '~/store/run-process-panel/run-process-panel-actions';

const mapStateToProps = ({ runProcessPanel }: RootState): RunProcessPanelRootDataProps => {
   return {
       currentStep: runProcessPanel.currentStep
   };
};

const mapDispatchToProps = (dispatch: Dispatch): RunProcessPanelRootActionProps => ({
    onSetStep: (step: number) => {
        dispatch<any>(goToStep(step));
    }
});

export const RunProcessPanel = connect(mapStateToProps, mapDispatchToProps)(RunProcessPanelRoot);