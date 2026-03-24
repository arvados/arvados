// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { ArvadosTheme } from 'common/custom-theme';
import { ShareMeIcon } from 'components/icon/icon';
import { API_CLIENT_AUTHORIZATION_PANEL_ID } from '../../store/api-client-authorizations/api-client-authorizations-actions';
import { DataExplorer } from 'views-components/data-explorer/data-explorer';
import { ResourcesState } from 'store/resources/resources';

type CssRules = 'root';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
    }
});

const DEFAULT_MESSAGE = 'Your api client authorization list is empty.';

export interface ApiClientAuthorizationPanelRootActionProps {
    onItemClick: (item: string) => void;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, item: string) => void;
    onItemDoubleClick: (item: string) => void;
}

export interface ApiClientAuthorizationPanelRootDataProps {
    resources: ResourcesState;
}

type ApiClientAuthorizationPanelRootProps = ApiClientAuthorizationPanelRootActionProps
    & ApiClientAuthorizationPanelRootDataProps & WithStyles<CssRules>;

export const ApiClientAuthorizationPanelRoot = withStyles(styles)(
    ({ classes, onItemDoubleClick, onItemClick, onContextMenu }: ApiClientAuthorizationPanelRootProps) =>
        <div className={classes.root}><DataExplorer
            id={API_CLIENT_AUTHORIZATION_PANEL_ID}
            onRowClick={onItemClick}
            onRowDoubleClick={onItemDoubleClick}
            onContextMenu={onContextMenu}
            contextMenuColumn={true}
            hideColumnSelector
            hideSearchInput
            defaultViewIcon={ShareMeIcon}
            defaultViewMessages={[DEFAULT_MESSAGE]} />
        </div>
);
