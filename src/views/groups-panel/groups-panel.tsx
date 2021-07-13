// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { connect } from 'react-redux';
import { Grid, Button, Typography } from "@material-ui/core";
import { DataExplorer } from "views-components/data-explorer/data-explorer";
import { DataColumns } from 'components/data-table/data-table';
import { SortDirection } from 'components/data-table/data-column';
import { ResourceOwner } from 'views-components/data-explorer/renderers';
import { AddIcon } from 'components/icon/icon';
import { ResourceName } from 'views-components/data-explorer/renderers';
import { createTree } from 'models/tree';
import { GROUPS_PANEL_ID, openCreateGroupDialog } from 'store/groups-panel/groups-panel-actions';
import { noop } from 'lodash/fp';
import { ContextMenuKind } from 'views-components/context-menu/context-menu';
import { getResource, ResourcesState, filterResources } from 'store/resources/resources';
import { GroupResource } from 'models/group';
import { RootState } from 'store/store';
import { openContextMenu } from 'store/context-menu/context-menu-actions';
import { ResourceKind } from 'models/resource';
import { LinkClass, LinkResource } from 'models/link';
import { navigateToGroupDetails } from 'store/navigation/navigation-action';

export enum GroupsPanelColumnNames {
    GROUP = "Name",
    OWNER = "Owner",
    MEMBERS = "Members",
}

export const groupsPanelColumns: DataColumns<string> = [
    {
        name: GroupsPanelColumnNames.GROUP,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.ASC,
        filters: createTree(),
        render: uuid => <ResourceName uuid={uuid} />
    },
    {
        name: GroupsPanelColumnNames.OWNER,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceOwner uuid={uuid} />,
    },
    {
        name: GroupsPanelColumnNames.MEMBERS,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <GroupMembersCount uuid={uuid} />,
    },
];

const mapStateToProps = (state: RootState) => {
    return {
        resources: state.resources
    };
};

const mapDispatchToProps = {
    onContextMenu: openContextMenu,
    onRowDoubleClick: (uuid: string) =>
        navigateToGroupDetails(uuid),
    onNewGroup: openCreateGroupDialog,
};

export interface GroupsPanelProps {
    onNewGroup: () => void;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, item: any) => void;
    onRowDoubleClick: (item: string) => void;
    resources: ResourcesState;
}

export const GroupsPanel = connect(
    mapStateToProps, mapDispatchToProps
)(
    class GroupsPanel extends React.Component<GroupsPanelProps> {

        render() {
            return (
                <DataExplorer
                    id={GROUPS_PANEL_ID}
                    onRowClick={noop}
                    onRowDoubleClick={this.props.onRowDoubleClick}
                    onContextMenu={this.handleContextMenu}
                    contextMenuColumn={true}
                    hideColumnSelector
                    actions={
                        <Grid container justify='flex-end'>
                            <Button
                                variant="contained"
                                color="primary"
                                onClick={this.props.onNewGroup}>
                                <AddIcon /> New group
                        </Button>
                        </Grid>
                    } />
            );
        }

        handleContextMenu = (event: React.MouseEvent<HTMLElement>, resourceUuid: string) => {
            const resource = getResource<GroupResource>(resourceUuid)(this.props.resources);
            if (resource) {
                this.props.onContextMenu(event, {
                    name: '',
                    uuid: resource.uuid,
                    ownerUuid: resource.ownerUuid,
                    kind: resource.kind,
                    menuKind: ContextMenuKind.GROUPS
                });
            }
        }
    });


const GroupMembersCount = connect(
    (state: RootState, props: { uuid: string }) => {

        const permissions = filterResources((resource: LinkResource) =>
            resource.kind === ResourceKind.LINK &&
            resource.linkClass === LinkClass.PERMISSION &&
            resource.tailUuid === props.uuid
        )(state.resources);

        return {
            children: permissions.length,
        };

    }
)(Typography);
