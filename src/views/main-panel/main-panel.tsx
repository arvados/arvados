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
        sessionIdleTimeout: parse(state.auth.config.clusterConfig.Workbench.IdleTimeout, 's') || 0
    };
};

const mapDispatchToProps = null;

export const MainPanel = connect(mapStateToProps, mapDispatchToProps)(MainPanelRoot);
