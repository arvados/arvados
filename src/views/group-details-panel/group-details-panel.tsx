// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { connect } from 'react-redux';

import { DataExplorer } from "views-components/data-explorer/data-explorer";
import { DataColumns } from 'components/data-table/data-table';
import { ResourceFullName, ResourceUuid, ResourceEmail, ResourceUsername } from 'views-components/data-explorer/renderers';
import { createTree } from 'models/tree';
import { noop } from 'lodash/fp';
import { RootState } from 'store/store';
import { GROUP_DETAILS_PANEL_ID, openAddGroupMembersDialog } from 'store/group-details-panel/group-details-panel-actions';
import { openContextMenu } from 'store/context-menu/context-menu-actions';
import { ResourcesState, getResource } from 'store/resources/resources';
import { ContextMenuKind } from 'views-components/context-menu/context-menu';
import { PermissionResource } from 'models/permission';
import { Grid, Button, Tabs, Tab, Paper } from '@material-ui/core';
import { AddIcon } from 'components/icon/icon';

export enum GroupDetailsPanelColumnNames {
    FULL_NAME = "Name",
    UUID = "UUID",
    EMAIL = "Email",
    USERNAME = "Username",
}

export const groupDetailsPanelColumns: DataColumns<string> = [
    {
        name: GroupDetailsPanelColumnNames.FULL_NAME,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceFullName uuid={uuid} />
    },
    {
        name: GroupDetailsPanelColumnNames.USERNAME,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceUsername uuid={uuid} />
    },
    {
        name: GroupDetailsPanelColumnNames.UUID,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceUuid uuid={uuid} />
    },
    {
        name: GroupDetailsPanelColumnNames.EMAIL,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceEmail uuid={uuid} />
    },
];

const mapStateToProps = (state: RootState) => {
    return {
        resources: state.resources
    };
};

const mapDispatchToProps = {
    onContextMenu: openContextMenu,
    onAddUser: openAddGroupMembersDialog,
};

export interface GroupDetailsPanelProps {
    onContextMenu: (event: React.MouseEvent<HTMLElement>, item: any) => void;
    onAddUser: () => void;
    resources: ResourcesState;
}

export const GroupDetailsPanel = connect(
    mapStateToProps, mapDispatchToProps
)(
    class GroupDetailsPanel extends React.Component<GroupDetailsPanelProps> {

        render() {
            return (
                <DataExplorer
                    id={GROUP_DETAILS_PANEL_ID}
                    onRowClick={noop}
                    onRowDoubleClick={noop}
                    onContextMenu={this.handleContextMenu}
                    contextMenuColumn={true}
                    hideColumnSelector
                    hideSearchInput
                    actions={
                        <Grid container justify='flex-end'>
                            <Button
                                variant="contained"
                                color="primary"
                                onClick={this.props.onAddUser}>
                                <AddIcon /> Add user
                        </Button>
                        </Grid>
                    } />
            );
        }

        handleContextMenu = (event: React.MouseEvent<HTMLElement>, resourceUuid: string) => {
            const resource = getResource<PermissionResource>(resourceUuid)(this.props.resources);
            if (resource) {
                this.props.onContextMenu(event, {
                    name: '',
                    uuid: resource.uuid,
                    ownerUuid: resource.ownerUuid,
                    kind: resource.kind,
                    menuKind: ContextMenuKind.GROUP_MEMBER
                });
            }
        }
    });

