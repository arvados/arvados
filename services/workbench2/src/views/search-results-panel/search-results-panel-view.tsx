// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useEffect, useCallback, useState } from 'react';
import { DataTableFilterItem } from 'components/data-table-filters/data-table-filters';
import { extractUuidKind, ResourceKind } from 'models/resource';
import { ContainerRequestState } from 'models/container-request';
import { SEARCH_RESULTS_PANEL_ID } from 'store/search-results-panel/search-results-panel-actions';
import { DataExplorer } from 'views-components/data-explorer/data-explorer';
import { ResourceCluster } from 'views-components/data-explorer/renderers';
import servicesProvider from 'common/service-provider';
import { SearchResultsPanelProps } from "./search-results-panel";
import { Routes } from 'routes/routes';
import { Link } from 'react-router-dom';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { ArvadosTheme } from 'common/custom-theme';
import { getSearchSessions } from 'store/search-bar/search-bar-actions';
import { camelCase } from 'lodash';

export type CssRules = 'siteManagerLink' | 'searchResults';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    searchResults: {
        width: '100%'
    },
    siteManagerLink: {
        marginRight: theme.spacing(2),
        float: 'right'
    }
});

export interface WorkflowPanelFilter extends DataTableFilterItem {
    type: ResourceKind | ContainerRequestState;
}

export const SearchResultsPanelView = withStyles(styles, { withTheme: true })(
    (props: SearchResultsPanelProps & WithStyles<CssRules, true>) => {
        const homeCluster = props.user.uuid.substring(0, 5);
        const loggedIn = props.sessions.filter((ss) => ss.loggedIn && ss.userIsActive);
        const [selectedItem, setSelectedItem] = useState('');
        const [itemPath, setItemPath] = useState<string[]>([]);

        useEffect(() => {
            let tmpPath: string[] = [];

            (async () => {
                if (selectedItem !== '') {
                    let searchUuid = selectedItem;
                    let itemKind = extractUuidKind(searchUuid);

                    while (itemKind !== ResourceKind.USER) {
                        const clusterId = searchUuid.split('-')[0];
                        const serviceType = camelCase(itemKind?.replace('arvados#', ''));
                        const service = Object.values(servicesProvider.getServices())
                            .filter(({ resourceType }) => !!resourceType)
                            .find(({ resourceType }) => camelCase(resourceType).indexOf(serviceType) > -1);
                        const sessions = getSearchSessions(clusterId, props.sessions);

                        if (sessions.length > 0) {
                            const session = sessions[0];
                            const { name, ownerUuid } = await (service as any).get(searchUuid, false, undefined, session);
                            tmpPath.push(name);
                            searchUuid = ownerUuid;
                            itemKind = extractUuidKind(searchUuid);
                        } else {
                            break;
                        }
                    }

                    tmpPath.push(props.user.uuid === searchUuid ? 'Projects' : 'Shared with me');
                    setItemPath(tmpPath);
                }
            })();
        }, [selectedItem]);

        const onItemClick = useCallback((uuid) => {
            setSelectedItem(uuid);
            props.onItemClick(uuid);
        }, [props.onItemClick]);

        return <span data-cy='search-results' className={props.classes.searchResults}>
            <DataExplorer
                id={SEARCH_RESULTS_PANEL_ID}
                onRowClick={onItemClick}
                onRowDoubleClick={props.onItemDoubleClick}
                onContextMenu={props.onContextMenu}
                contextMenuColumn={false}
                elementPath={`/ ${itemPath.reverse().join(' / ')}`}
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
