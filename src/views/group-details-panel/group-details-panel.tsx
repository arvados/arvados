// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { connect } from 'react-redux';

import { DataExplorer } from "views-components/data-explorer/data-explorer";
import { DataColumns } from 'components/data-table/data-table';
import { ResourceUuid, ResourceEmail, ResourceUsername, ResourceLinkName, ResourceLinkHead, ResourceLinkTail } from 'views-components/data-explorer/renderers';
import { createTree } from 'models/tree';
import { noop } from 'lodash/fp';
import { RootState } from 'store/store';
import { GROUP_DETAILS_MEMBERS_PANEL_ID, GROUP_DETAILS_PERMISSIONS_PANEL_ID, openAddGroupMembersDialog } from 'store/group-details-panel/group-details-panel-actions';
import { openContextMenu } from 'store/context-menu/context-menu-actions';
import { ResourcesState, getResource } from 'store/resources/resources';
import { ContextMenuKind } from 'views-components/context-menu/context-menu';
import { PermissionResource } from 'models/permission';
import { Grid, Button, Tabs, Tab, Paper } from '@material-ui/core';
import { AddIcon } from 'components/icon/icon';

export enum GroupDetailsPanelMembersColumnNames {
    FULL_NAME = "Name",
    UUID = "UUID",
    EMAIL = "Email",
    USERNAME = "Username",
}

export enum GroupDetailsPanelPermissionsColumnNames {
    HEAD = "Head",
    NAME = "Name",
    UUID = "UUID",
}

export const groupDetailsMembersPanelColumns: DataColumns<string> = [
    {
        name: GroupDetailsPanelMembersColumnNames.FULL_NAME,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceLinkTail uuid={uuid} />
    },
    {
        name: GroupDetailsPanelMembersColumnNames.USERNAME,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceUsername uuid={uuid} />
    },
    {
        name: GroupDetailsPanelMembersColumnNames.UUID,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceUuid uuid={uuid} />
    },
    {
        name: GroupDetailsPanelMembersColumnNames.EMAIL,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceEmail uuid={uuid} />
    },
];

export const groupDetailsPermissionsPanelColumns: DataColumns<string> = [
    {
        name: GroupDetailsPanelPermissionsColumnNames.HEAD,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceLinkHead uuid={uuid} />
    },
    {
        name: GroupDetailsPanelPermissionsColumnNames.NAME,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceLinkName uuid={uuid} />
    },
    {
        name: GroupDetailsPanelPermissionsColumnNames.UUID,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceUuid uuid={uuid} />
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
        state = {
          value: 0,
        };

        componentDidMount() {
            this.setState({ value: 0 });
        }

        render() {
            const { value } = this.state;
            return (
                <Paper>
                  <Tabs value={value} onChange={this.handleChange} variant="fullWidth">
                      <Tab label="MEMBERS" />
                      <Tab label="PERMISSIONS" />
                  </Tabs>
                  {value === 0 &&
                      <DataExplorer
                          id={GROUP_DETAILS_MEMBERS_PANEL_ID}
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
                          }
                          paperProps={{
                              elevation: 0,
                          }} />
                  }
                  {value === 1 &&
                      <DataExplorer
                          id={GROUP_DETAILS_PERMISSIONS_PANEL_ID}
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
                          }
                          paperProps={{
                              elevation: 0,
                          }} />
                  }
                </Paper>
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

        handleChange = (event: React.MouseEvent<HTMLElement>, value: number) => {
            this.setState({ value });
        }
    });
