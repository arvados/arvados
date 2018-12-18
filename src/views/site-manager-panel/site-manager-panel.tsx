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
import { Session } from "~/models/session";
import { toggleSession } from "~/store/auth/auth-action-session";

const mapStateToProps = (state: RootState): SiteManagerPanelRootDataProps => {
    return {
        sessions: state.auth.sessions
    };
};

const mapDispatchToProps = (dispatch: Dispatch): SiteManagerPanelRootActionProps => ({
    toggleSession: (session: Session) => {
        dispatch<any>(toggleSession(session));
    }
});

export const SiteManagerPanel = connect(mapStateToProps, mapDispatchToProps)(SiteManagerPanelRoot);
