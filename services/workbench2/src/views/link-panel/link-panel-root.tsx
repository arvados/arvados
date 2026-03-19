// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { LINK_PANEL_ID } from 'store/link-panel/link-panel-actions';
import { DataExplorer } from 'views-components/data-explorer/data-explorer';
import { ResourcesState } from 'store/resources/resources';
import { ShareMeIcon } from 'components/icon/icon';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { ArvadosTheme } from 'common/custom-theme';

type CssRules = "root";

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
    }
});

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
