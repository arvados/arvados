// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { RootState } from '~/store/store';
import { Dispatch } from 'redux';
import { connect } from 'react-redux';
import {
    SiteManagerPanelRoot, SiteManagerPanelRootActionProps,
    SiteManagerPanelRootDataProps
} from "~/views/site-manager-panel/site-manager-panel-root";

const mapStateToProps = (state: RootState): SiteManagerPanelRootDataProps => {
    return {
        sessions: state.auth.sessions,
        user: state.auth.user!!
    };
};

const mapDispatchToProps = (dispatch: Dispatch): SiteManagerPanelRootActionProps => ({
});

export const SiteManagerPanel = connect(mapStateToProps, mapDispatchToProps)(SiteManagerPanelRoot);
