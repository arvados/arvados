// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { connect } from 'react-redux';
import { RootState } from 'store/store';

export type WithProgressStateProps = {
    working: boolean;
};

export const withProgress = (id: string) =>
    (component: React.ComponentType<WithProgressStateProps>) =>
        connect(mapStateToProps(id))(component);

export const mapStateToProps = (id: string) => (state: RootState): WithProgressStateProps => {
    const progress = state.progressIndicator.find(p => p.id === id);
    return { working: progress ? progress.working : false };
};
