// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose, Dispatch } from 'redux';
import { connect } from 'react-redux';

import * as React from 'react';
import { connectSharingDialog } from '~/store/sharing-dialog/sharing-dialog-actions';
import { WithDialogProps } from '~/store/dialog/with-dialog';
import { RootState } from '~/store/store';

import SharingDialogComponent, { SharingDialogDataProps, SharingDialogActionProps } from './sharing-dialog-component';
import { SharingDialogContent } from './sharing-dialog-content';
import { connectAdvancedViewSwitch, AdvancedViewSwitchInjectedProps } from './advanced-view-switch';
import { isDirty } from 'redux-form';

const mapStateToProps = (state: RootState, { advancedViewOpen, ...props }: WithDialogProps<string> & AdvancedViewSwitchInjectedProps): SharingDialogDataProps => ({
    ...props,
    saveEnabled: isDirty('SHARING_PUBLIC_ACCESS_FORM')(state) ||
        isDirty('SHARING_MANAGEMENT_FORM')(state) ||
        isDirty('SHARING_INVITATION_FORM')(state),
    advancedEnabled: !advancedViewOpen,
    children: <SharingDialogContent {...{ advancedViewOpen }} />,
});

const mapDispatchToProps = (_: Dispatch, { toggleAdvancedView, ...props }: WithDialogProps<string> & AdvancedViewSwitchInjectedProps): SharingDialogActionProps => ({
    ...props,
    onClose: props.closeDialog,
    onExited: toggleAdvancedView,
    onSave: () => { console.log('save'); },
    onAdvanced: toggleAdvancedView,
});

export const SharingDialog = compose(
    connectAdvancedViewSwitch,
    connectSharingDialog,
    connect(mapStateToProps, mapDispatchToProps)
)(SharingDialogComponent);

