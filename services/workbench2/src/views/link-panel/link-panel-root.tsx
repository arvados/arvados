// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { LINK_PANEL_ID } from 'store/link-panel/link-panel-actions';
import { DataExplorer } from 'views-components/data-explorer/data-explorer';
import { SortDirection } from 'components/data-table/data-column';
import { DataColumns } from 'components/data-table/data-table';
import { ResourcesState } from 'store/resources/resources';
import { ShareMeIcon } from 'components/icon/icon';
import { createTree } from 'models/tree';
import {
    ResourceLinkUuid, ResourceLinkHead, ResourceLinkTail,
    ResourceLinkClass, ResourceLinkName }
from 'views-components/data-explorer/renderers';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';
import { LinkResource } from 'models/link';

type CssRules = "root";

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
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
        render: uuid => <ResourceLinkName uuid={uuid} />
    },
    {
        name: LinkPanelColumnNames.LINK_CLASS,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceLinkClass uuid={uuid} />
    },
    {
        name: LinkPanelColumnNames.TAIL,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceLinkTail uuid={uuid} />
    },
    {
        name: LinkPanelColumnNames.HEAD,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceLinkHead uuid={uuid} />
    },
    {
        name: LinkPanelColumnNames.UUID,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceLinkUuid uuid={uuid} />
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
