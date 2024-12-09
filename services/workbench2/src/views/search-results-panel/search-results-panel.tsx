// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { connect } from "react-redux";
import { navigateTo } from 'store/navigation/navigation-action';
import { openSearchResultsContextMenu } from 'store/context-menu/context-menu-actions';
import { loadDetailsPanel } from 'store/details-panel/details-panel-action';
import { SearchResultsPanelView } from 'views/search-results-panel/search-results-panel-view';
import { RootState } from 'store/store';
import { SearchBarAdvancedFormData } from 'models/search-bar';
import { User } from "models/user";
import { Config } from 'common/config';
import { Session } from "models/session";
import { toggleOne, deselectAllOthers } from "store/multiselect/multiselect-actions";
import { GroupContentsResource } from "services/groups-service/groups-service";

export interface SearchResultsPanelDataProps {
    data: SearchBarAdvancedFormData;
    user: User;
    sessions: Session[];
    remoteHostsConfig: { [key: string]: Config };
    localCluster: string;
}

export interface SearchResultsPanelActionProps {
    onItemClick: (resource: GroupContentsResource) => void;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, resource: GroupContentsResource) => void;
    onDialogOpen: (ownerUuid: string) => void;
    onItemDoubleClick: (resource: GroupContentsResource) => void;
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
    onContextMenu: (event, resource) => {
        dispatch<any>(openSearchResultsContextMenu(event, resource.uuid));
    },
    onDialogOpen: (ownerUuid: string) => { return; },
    onItemClick: ({uuid}: GroupContentsResource) => {
        dispatch<any>(toggleOne(uuid))
        dispatch<any>(deselectAllOthers(uuid))
        dispatch<any>(loadDetailsPanel(uuid));
    },
    onItemDoubleClick: ({uuid}: GroupContentsResource) => {
        dispatch<any>(navigateTo(uuid));
    }
});

export const SearchResultsPanel = connect(mapStateToProps, mapDispatchToProps)(SearchResultsPanelView);
