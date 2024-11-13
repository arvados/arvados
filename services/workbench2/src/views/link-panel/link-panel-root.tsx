// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { LINK_PANEL_ID } from 'store/link-panel/link-panel-actions';
import { DataExplorer } from 'views-components/data-explorer/data-explorer';
import { DataColumns, SortDirection } from 'components/data-table/data-column';
import { ResourcesState } from 'store/resources/resources';
import { ShareMeIcon } from 'components/icon/icon';
import { createTree } from 'models/tree';
import {
    renderUuidWithCopy, ResourceLinkHead, ResourceLinkTail,
    renderLinkClass, renderLinkName}
from 'views-components/data-explorer/renderers';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { ArvadosTheme } from 'common/custom-theme';
import { LinkResource } from 'models/link';

type CssRules = "root";

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
    }
});

export enum LinkPanelColumnNames {
    NAME = "Name",
    LINK_CLASS = "Link Class",
    TAIL = "Tail",
    HEAD = 'Head',
    UUID = "UUID"
}

export const linkPanelColumns: DataColumns<string, LinkResource> = [
    {
        name: LinkPanelColumnNames.NAME,
        selected: true,
        configurable: true,
        sort: {direction: SortDirection.NONE, field: "name"},
        filters: createTree(),
        render: (resource) => renderLinkName(resource as LinkResource)
    },
    {
        name: LinkPanelColumnNames.LINK_CLASS,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: (resource) => renderLinkClass(resource as LinkResource)
    },
    {
        name: LinkPanelColumnNames.TAIL,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: (resource) => <ResourceLinkTail resource={resource as LinkResource} />
    },
    {
        name: LinkPanelColumnNames.HEAD,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: (resource) => <ResourceLinkHead resource={resource as LinkResource} />
    },
    {
        name: LinkPanelColumnNames.UUID,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: (resource: LinkResource) => renderUuidWithCopy({ uuid: resource.uuid })
    }
];

export interface LinkPanelRootDataProps {
    resources: ResourcesState;
}

export interface LinkPanelRootActionProps {
    onItemClick: (item: string) => void;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, item: string) => void;
    onItemDoubleClick: (item: string) => void;
}

export type LinkPanelRootProps = LinkPanelRootDataProps & LinkPanelRootActionProps & WithStyles<CssRules>;

export const LinkPanelRoot = withStyles(styles)((props: LinkPanelRootProps) => {
    return <div className={props.classes.root}><DataExplorer
        id={LINK_PANEL_ID}
        onRowClick={props.onItemClick}
        onRowDoubleClick={props.onItemDoubleClick}
        onContextMenu={props.onContextMenu}
        contextMenuColumn={true}
        hideColumnSelector
        hideSearchInput
        defaultViewIcon={ShareMeIcon}
        defaultViewMessages={['Your link list is empty.']} />
    </div>;
});
