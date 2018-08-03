// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { connect } from 'react-redux';
import { DialogState } from './dialog-reducer';
import { Dispatch } from 'redux';
import { dialogActions } from './dialog-actions';

export type WithDialog<T> = {
    open: boolean;
    data?: T;
};

export type WithDialogActions = {
    closeDialog: () => void;
};

export const withDialog = (id: string) =>
    <T>(component: React.ComponentType<WithDialog<T> & WithDialogActions>) =>
        connect(mapStateToProps(id), mapDispatchToProps(id))(component);

export const mapStateToProps = (id: string) => <T>(state: { dialog: DialogState }): WithDialog<T> => {
    const dialog = state.dialog[id];
    return dialog ? dialog : { open: false };
};

export const mapDispatchToProps = (id: string) => (dispatch: Dispatch): WithDialogActions => ({
    closeDialog: () => {
        dispatch(dialogActions.CLOSE_DIALOG({ id }));
    }
});