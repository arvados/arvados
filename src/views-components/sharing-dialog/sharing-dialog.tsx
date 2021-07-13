// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose, Dispatch } from 'redux';
import { connect } from 'react-redux';

import React from 'react';
import { connectSharingDialog, saveSharingDialogChanges, connectSharingDialogProgress, sendSharingInvitations } from 'store/sharing-dialog/sharing-dialog-actions';
import { WithDialogProps } from 'store/dialog/with-dialog';
import { RootState } from 'store/store';

import SharingDialogComponent, { SharingDialogDataProps, SharingDialogActionProps } from './sharing-dialog-component';
import { SharingDialogContent } from './sharing-dialog-content';
import { connectAdvancedViewSwitch, AdvancedViewSwitchInjectedProps } from './advanced-view-switch';
import { hasChanges } from 'store/sharing-dialog/sharing-dialog-types';
import { WithProgressStateProps } from 'store/progress-indicator/with-progress';

type Props = WithDialogProps<string> & AdvancedViewSwitchInjectedProps & WithProgressStateProps;

const mapStateToProps = (state: RootState, { advancedViewOpen, working, ...props }: Props): SharingDialogDataProps => ({
    ...props,
    saveEnabled: hasChanges(state),
    loading: working,
    advancedEnabled: !advancedViewOpen,
    children: <SharingDialogContent {...{ advancedViewOpen }} />,
});

const mapDispatchToProps = (dispatch: Dispatch, { toggleAdvancedView, advancedViewOpen, ...props }: Props): SharingDialogActionProps => ({
    ...props,
    onClose: props.closeDialog,
    onExited: () => {
        if (advancedViewOpen) {
            toggleAdvancedView();
        }
    },
    onSave: () => {
        if (advancedViewOpen) {
            dispatch<any>(saveSharingDialogChanges);
        } else {
            dispatch<any>(sendSharingInvitations);
        }
    },
    onAdvanced: toggleAdvancedView,
});

export const SharingDialog = compose(
    connectAdvancedViewSwitch,
    connectSharingDialog,
    connectSharingDialogProgress,
    connect(mapStateToProps, mapDispatchToProps)
)(SharingDialogComponent);

