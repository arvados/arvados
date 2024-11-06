// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { DataExplorer } from "views-components/data-explorer/data-explorer";
import { connect, DispatchProp } from 'react-redux';
import { RootState } from 'store/store';
import { ArvadosTheme } from 'common/custom-theme';
import { ShareMeIcon } from 'components/icon/icon';
import { ResourcesState, getResource } from 'store/resources/resources';
import { ResourceKind } from 'models/resource';
import { navigateTo } from "store/navigation/navigation-action";
import { loadDetailsPanel } from "store/details-panel/details-panel-action";
import { SHARED_WITH_ME_PANEL_ID } from 'store/shared-with-me-panel/shared-with-me-panel-actions';
import {
    openContextMenu,
    resourceUuidToContextMenuKind
} from 'store/context-menu/context-menu-actions';

import { DataTableFilterItem } from 'components/data-table-filters/data-table-filters';
import { GroupContentsResource } from 'services/groups-service/groups-service';
import { toggleOne, deselectAllOthers } from 'store/multiselect/multiselect-actions';

import { ContainerRequestState } from 'models/container-request';


type CssRules = "toolbar" | "button" | "root";

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    toolbar: {
        paddingBottom: theme.spacing(3),
        textAlign: "right"
    },
    button: {
        marginLeft: theme.spacing(1)
    },
    root: {
        width: '100%',
    },
});


export interface ProjectPanelFilter extends DataTableFilterItem {
    type: ResourceKind | ContainerRequestState;
}



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
                    defaultViewIcon={ShareMeIcon}
                    defaultViewMessages={['No shared items']} />
                </div>;
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

            handleRowClick = ({uuid}: GroupContentsResource) => {
                this.props.dispatch<any>(toggleOne(uuid))
                this.props.dispatch<any>(deselectAllOthers(uuid))
                this.props.dispatch<any>(loadDetailsPanel(uuid));
            }
        }
    )
);
