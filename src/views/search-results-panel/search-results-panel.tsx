// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { connect } from "react-redux";
import { navigateTo } from 'store/navigation/navigation-action';
// import { openContextMenu, resourceKindToContextMenuKind } from 'store/context-menu/context-menu-actions';
// import { ResourceKind } from 'models/resource';
import { loadDetailsPanel } from 'store/details-panel/details-panel-action';
import { SearchResultsPanelView } from 'views/search-results-panel/search-results-panel-view';
import { RootState } from 'store/store';
import { SearchBarAdvancedFormData } from 'models/search-bar';
import { User } from "models/user";
import { Config } from 'common/config';
import { Session } from "models/session";
import { snackbarActions, SnackbarKind } from "store/snackbar/snackbar-actions";

export interface SearchResultsPanelDataProps {
    data: SearchBarAdvancedFormData;
    user: User;
    sessions: Session[];
    remoteHostsConfig: { [key: string]: Config };
    localCluster: string;
}

export interface SearchResultsPanelActionProps {
    onItemClick: (item: string) => void;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, item: string) => void;
    onDialogOpen: (ownerUuid: string) => void;
    onItemDoubleClick: (item: string) => void;
    onPathDisplay: (path: string) => void;
}

export type SearchResultsPanelProps = SearchResultsPanelDataProps & SearchResultsPanelActionProps;

const mapStateToProps = (rootState: RootState) => {
    return {
        user: rootState.auth.user,
        sessions: rootState.auth.sessions,
        remoteHostsConfig: rootState.auth.remoteHostsConfig,
        localCluster: rootState.auth.localCluster,
    };
};

const mapDispatchToProps = (dispatch: Dispatch): SearchResultsPanelActionProps => ({
    onContextMenu: (event, resourceUuid) => { return; },
    onDialogOpen: (ownerUuid: string) => { return; },
    onItemClick: (resourceUuid: string) => {
        dispatch<any>(loadDetailsPanel(resourceUuid));
    },
    onItemDoubleClick: uuid => {
        dispatch<any>(navigateTo(uuid));
    },
    onPathDisplay: (path: string) => {
        dispatch(snackbarActions.SHIFT_MESSAGES());
        dispatch(snackbarActions.OPEN_SNACKBAR({
            message: path,
            kind: SnackbarKind.INFO,
            hideDuration: 9999999999,
        }));
    },
});

export const SearchResultsPanel = connect(mapStateToProps, mapDispatchToProps)(SearchResultsPanelView);
