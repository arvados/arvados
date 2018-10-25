// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose, Dispatch } from 'redux';
import { connect } from 'react-redux';

import { connectSharingDialog } from '~/store/sharing-dialog/sharing-dialog-actions';
import { WithDialogProps } from '~/store/dialog/with-dialog';
import { RootState } from '~/store/store';

import SharingDialogComponent, { SharingDialogDataProps, SharingDialogActionProps } from './sharing-dialog-component';

const mapStateToProps = (_: RootState, props: WithDialogProps<string>): SharingDialogDataProps => ({
    ...props,
    saveEnabled: false,
    children: null,
});

const mapDispatchToProps = (_: Dispatch, props: WithDialogProps<string>): SharingDialogActionProps => ({
    ...props,
    onClose: props.closeDialog,
    onSave: () => { console.log('save'); },
    onAdvanced: () => { console.log('advanced'); },
});

export const SharingDialog = compose(
    connectSharingDialog,
    connect(mapStateToProps, mapDispatchToProps)
)(SharingDialogComponent);
