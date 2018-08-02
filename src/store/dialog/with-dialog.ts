// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { connect } from 'react-redux';
import { DialogState } from './dialog-reducer';

export type WithDialog<T> = {
    open: boolean;
    data?: T;
};

export const withDialog = (id: string) =>
    <T>(component: React.ComponentType<WithDialog<T>>) =>
        connect(mapStateToProps(id))(component);

export const mapStateToProps = (id: string) => <T>(state: { dialog: DialogState }): WithDialog<T> => {
    const dialog = state.dialog[id];
    return dialog ? dialog : { open: false };
};
