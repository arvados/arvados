// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { RootState } from 'store/store';
import { connect } from 'react-redux';
import parse from 'parse-duration';
import { MainPanelRoot, MainPanelRootDataProps } from 'views/main-panel/main-panel-root';
import { isSystemWorking } from 'store/progress-indicator/progress-indicator-reducer';
import { isWorkbenchLoading } from 'store/workbench/workbench-actions';
import { LinkAccountPanelStatus } from 'store/link-account-panel/link-account-panel-reducer';
import { matchLinkAccountRoute } from 'routes/routes';
import { toggleSidePanel } from "store/side-panel/side-panel-action";
import { propertiesActions } from 'store/properties/properties-actions';

const mapStateToProps = (state: RootState): MainPanelRootDataProps => {
    return {
        user: state.auth.user,
        working: isSystemWorking(state.progressIndicator),
        loading: isWorkbenchLoading(state),
        buildInfo: state.appInfo.buildInfo,
        uuidPrefix: state.auth.localCluster,
        isNotLinking: state.linkAccountPanel.status === LinkAccountPanelStatus.NONE || state.linkAccountPanel.status === LinkAccountPanelStatus.INITIAL,
        isLinkingPath: state.router.location ? matchLinkAccountRoute(state.router.location.pathname) !== null : false,
        siteBanner: state.auth.config.clusterConfig.Workbench.SiteName,
        sessionIdleTimeout: parse(state.auth.config.clusterConfig.Workbench.IdleTimeout, 's') || 0,
        sidePanelIsCollapsed: state.sidePanel.collapsedState,
        isTransitioning: state.detailsPanel.isTransitioning,
        isDetailsPanelOpen: state.detailsPanel.isOpened,
        currentSideWidth: state.sidePanel.currentSideWidth,
        currentRoute: state.router.location ? state.router.location.pathname : '',
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
