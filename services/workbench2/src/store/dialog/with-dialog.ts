// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { connect } from 'react-redux';
import { DialogState } from './dialog-reducer';
import { Dispatch } from 'redux';
import { dialogActions } from './dialog-actions';

export type WithDialogStateProps<T> = {
    open: boolean;
    data: T;
};

export type WithDialogDispatchProps = {
    closeDialog: () => void;
};

export type WithDialogProps<T> = WithDialogStateProps<T> & WithDialogDispatchProps;
export const withDialog = (id: string) =>
    // TODO: How to make compiler happy with & P instead of & any?
    // eslint-disable-next-line
    <T, P>(component: React.ComponentType<WithDialogProps<T> & any>) =>
        connect(mapStateToProps(id), mapDispatchToProps(id))(component);

const emptyData = {};

export const mapStateToProps = (id: string) => <T>(state: { dialog: DialogState }): WithDialogStateProps<T> => {
    const dialog = state.dialog[id];
    return dialog ? dialog : { open: false, data: emptyData };
};

export const mapDispatchToProps = (id: string) => (dispatch: Dispatch): WithDialogDispatchProps => ({
    closeDialog: () => {
        dispatch(dialogActions.CLOSE_DIALOG({ id }));
    }
});
