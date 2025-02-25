// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { RootState } from 'store/store';
import { connect } from 'react-redux';
import { MainPanelRoot, MainPanelRootDataProps } from 'views/main-panel/main-panel-root';
import { toggleSidePanel } from "store/side-panel/side-panel-action";
import { propertiesActions } from 'store/properties/properties-actions';

const mapStateToProps = (state: RootState): MainPanelRootDataProps => {
    return {
        user: state.auth.user,
        progressIndicator: state.progressIndicator,
        buildInfo: state.appInfo.buildInfo,
        uuidPrefix: state.auth.localCluster,
        linkAccountPanel: state.linkAccountPanel,
        config: state.auth.config,
        sidePanelIsCollapsed: state.sidePanel.collapsedState,
        isDetailsPanelOpen: state.detailsPanel.isOpened,
        router: state.router,
    };
};

const mapDispatchToProps = (dispatch) => {
    return {
        toggleSidePanel: (collapsedState)=>{
            return dispatch(toggleSidePanel(collapsedState))
        },
        setCurrentRouteUuid: (uuid: string) => {
            return dispatch(propertiesActions.SET_PROPERTY({key: 'currentRouteUuid', value: uuid}))}
    }
};

export const MainPanel = connect(mapStateToProps, mapDispatchToProps)(MainPanelRoot);
