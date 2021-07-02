// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { RootState } from 'store/store';
import { connect } from 'react-redux';
import { NotFoundPanelRoot, NotFoundPanelRootDataProps, NotFoundPanelOwnProps } from 'views/not-found-panel/not-found-panel-root';

const mapStateToProps = (state: RootState): NotFoundPanelRootDataProps => {
    return {
        location: state.router.location,
        clusterConfig: state.auth.config.clusterConfig,
    };
};

const mapDispatchToProps = null;

export const NotFoundPanel = connect(mapStateToProps, mapDispatchToProps)
    (NotFoundPanelRoot) as any;
