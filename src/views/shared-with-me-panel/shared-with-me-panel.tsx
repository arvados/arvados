// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core';
import { DataExplorer } from "views-components/data-explorer/data-explorer";
import { connect, DispatchProp } from 'react-redux';
import { RootState } from 'store/store';
import { ArvadosTheme } from 'common/custom-theme';
import { ShareMeIcon } from 'components/icon/icon';
import { ResourcesState, getResource } from 'store/resources/resources';
import { navigateTo } from "store/navigation/navigation-action";
import { loadDetailsPanel } from "store/details-panel/details-panel-action";
import { DataTableDefaultView } from 'components/data-table-default-view/data-table-default-view';
import { SHARED_WITH_ME_PANEL_ID } from 'store/shared-with-me-panel/shared-with-me-panel-actions';
import {
    openContextMenu,
    resourceUuidToContextMenuKind
} from 'store/context-menu/context-menu-actions';
import { GroupContentsResource } from 'services/groups-service/groups-service';

type CssRules = "toolbar" | "button" | "root";

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    toolbar: {
        paddingBottom: theme.spacing.unit * 3,
        textAlign: "right"
    },
    button: {
        marginLeft: theme.spacing.unit
    },
    root: {
        width: '100%',
    },
});

interface SharedWithMePanelDataProps {
    resources: ResourcesState;
    userUuid: string;
}

type SharedWithMePanelProps = SharedWithMePanelDataProps & DispatchProp & WithStyles<CssRules>;

export const SharedWithMePanel = withStyles(styles)(
    connect((state: RootState) => ({
        resources: state.resources,
        userUuid: state.auth.user!.uuid,
    }))(
        class extends React.Component<SharedWithMePanelProps> {
            render() {
                return <div className={this.props.classes.root}><DataExplorer
                    id={SHARED_WITH_ME_PANEL_ID}
                    onRowClick={this.handleRowClick}
                    onRowDoubleClick={this.handleRowDoubleClick}
                    onContextMenu={this.handleContextMenu}
                    contextMenuColumn={false}
                    dataTableDefaultView={<DataTableDefaultView icon={ShareMeIcon} />} /></div>;
            }

            handleContextMenu = (event: React.MouseEvent<HTMLElement>, resourceUuid: string) => {
                const { resources } = this.props;
                const resource = getResource<GroupContentsResource>(resourceUuid)(resources);
                const menuKind = this.props.dispatch<any>(resourceUuidToContextMenuKind(resourceUuid));
                if (menuKind && resource) {
                    this.props.dispatch<any>(openContextMenu(event, {
                        name: resource.name,
                        uuid: resource.uuid,
                        description: resource.description,
                        ownerUuid: resource.ownerUuid,
                        isTrashed: ('isTrashed' in resource) ? resource.isTrashed: false,
                        kind: resource.kind,
                        menuKind
                    }));
                }
                this.props.dispatch<any>(loadDetailsPanel(resourceUuid));
            }

            handleRowDoubleClick = (uuid: string) => {
                this.props.dispatch<any>(navigateTo(uuid));
            }

            handleRowClick = (uuid: string) => {
                this.props.dispatch<any>(loadDetailsPanel(uuid));
            }
        }
    )
);
