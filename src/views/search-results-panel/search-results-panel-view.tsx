// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { SortDirection } from 'components/data-table/data-column';
import { DataColumns } from 'components/data-table/data-table';
import { DataTableFilterItem } from 'components/data-table-filters/data-table-filters';
import { ResourceKind } from 'models/resource';
import { ContainerRequestState } from 'models/container-request';
import { SEARCH_RESULTS_PANEL_ID } from 'store/search-results-panel/search-results-panel-actions';
import { DataExplorer } from 'views-components/data-explorer/data-explorer';
import {
    ResourceCluster,
    ResourceFileSize,
    ResourceLastModifiedDate,
    ResourceName,
    ResourceOwnerWithName,
    ResourceStatus,
    ResourceType
} from 'views-components/data-explorer/renderers';
import { createTree } from 'models/tree';
import { getInitialResourceTypeFilters } from 'store/resource-type-filters/resource-type-filters';
import { SearchResultsPanelProps } from "./search-results-panel";
import { Routes } from 'routes/routes';
import { Link } from 'react-router-dom';
import { StyleRulesCallback, withStyles, WithStyles } from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';

export enum SearchResultsPanelColumnNames {
    CLUSTER = "Cluster",
    NAME = "Name",
    STATUS = "Status",
    TYPE = 'Type',
    OWNER = "Owner",
    FILE_SIZE = "File size",
    LAST_MODIFIED = "Last modified"
}

export type CssRules = 'siteManagerLink';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    siteManagerLink: {
        marginRight: theme.spacing.unit * 2,
        float: 'right'
    }
});

export interface WorkflowPanelFilter extends DataTableFilterItem {
    type: ResourceKind | ContainerRequestState;
}

export const searchResultsPanelColumns: DataColumns<string> = [
    {
        name: SearchResultsPanelColumnNames.CLUSTER,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: (uuid: string) => <ResourceCluster uuid={uuid} />
    },
    {
        name: SearchResultsPanelColumnNames.NAME,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.NONE,
        filters: createTree(),
        render: (uuid: string) => <ResourceName uuid={uuid} />
    },
    {
        name: SearchResultsPanelColumnNames.STATUS,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceStatus uuid={uuid} />
    },
    {
        name: SearchResultsPanelColumnNames.TYPE,
        selected: true,
        configurable: true,
        filters: getInitialResourceTypeFilters(),
        render: (uuid: string) => <ResourceType uuid={uuid} />,
    },
    {
        name: SearchResultsPanelColumnNames.OWNER,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceOwnerWithName uuid={uuid} />
    },
    {
        name: SearchResultsPanelColumnNames.FILE_SIZE,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceFileSize uuid={uuid} />
    },
    {
        name: SearchResultsPanelColumnNames.LAST_MODIFIED,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.DESC,
        filters: createTree(),
        render: uuid => <ResourceLastModifiedDate uuid={uuid} />
    }
];

export const SearchResultsPanelView = withStyles(styles, { withTheme: true })(
    (props: SearchResultsPanelProps & WithStyles<CssRules, true>) => {
        const homeCluster = props.user.uuid.substr(0, 5);
        const loggedIn = props.sessions.filter((ss) => ss.loggedIn && ss.userIsActive);
        return <span data-cy='search-results'><DataExplorer
            id={SEARCH_RESULTS_PANEL_ID}
            onRowClick={props.onItemClick}
            onRowDoubleClick={props.onItemDoubleClick}
            onContextMenu={props.onContextMenu}
            contextMenuColumn={false}
            hideSearchInput
            title={
                <div>
                    {loggedIn.length === 1 ?
                        <span>Searching local cluster <ResourceCluster uuid={props.localCluster} /></span>
                        : <span>Searching clusters: {loggedIn.map((ss) => <span key={ss.clusterId}>
                            <a href={props.remoteHostsConfig[ss.clusterId] && props.remoteHostsConfig[ss.clusterId].workbench2Url} style={{ textDecoration: 'none' }}> <ResourceCluster uuid={ss.clusterId} /></a>
                        </span>)}</span>}
                    {loggedIn.length === 1 && props.localCluster !== homeCluster ?
                        <span>To search multiple clusters, <a href={props.remoteHostsConfig[homeCluster] && props.remoteHostsConfig[homeCluster].workbench2Url}> start from your home Workbench.</a></span>
                        : <span style={{ marginLeft: "2em" }}>Use <Link to={Routes.SITE_MANAGER} >Site Manager</Link> to manage which clusters will be searched.</span>}
                </div >
            }
        /></span>;
    });
