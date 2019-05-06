// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { RootState } from '~/store/store';
import { connect } from 'react-redux';
import { MainPanelRoot, MainPanelRootDataProps } from '~/views/main-panel/main-panel-root';
import { isSystemWorking } from '~/store/progress-indicator/progress-indicator-reducer';
import { isWorkbenchLoading } from '~/store/workbench/workbench-actions';
import { LinkAccountPanelStatus } from '~/store/link-account-panel/link-account-panel-reducer';

const mapStateToProps = (state: RootState): MainPanelRootDataProps => {
    return {
        user: state.auth.user,
        working: isSystemWorking(state.progressIndicator),
        loading: isWorkbenchLoading(state),
        buildInfo: state.appInfo.buildInfo,
        uuidPrefix: state.auth.localCluster,
        isNotLinking: state.linkAccountPanel.status === LinkAccountPanelStatus.INITIAL
    };
};

const mapDispatchToProps = null;

export const MainPanel = connect(mapStateToProps, mapDispatchToProps)(MainPanelRoot);
