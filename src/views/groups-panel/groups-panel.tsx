// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { connect } from 'react-redux';
import { Grid, Button } from "@material-ui/core";

import { DataExplorer } from "~/views-components/data-explorer/data-explorer";
import { DataColumns } from '~/components/data-table/data-table';
import { SortDirection } from '~/components/data-table/data-column';
import { ResourceOwner } from '~/views-components/data-explorer/renderers';
import { AddIcon } from '~/components/icon/icon';
import { ResourceName } from '~/views-components/data-explorer/renderers';
import { createTree } from '~/models/tree';
import { GROUPS_PANEL_ID, openCreateGroupDialog } from '~/store/groups-panel/groups-panel-actions';
import { noop } from 'lodash/fp';

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
        render: uuid => <span>0</span>,
    },
];

export interface GroupsPanelProps {
    onNewGroup: () => void;
}

export const GroupsPanel = connect(
    null,
    {
        onNewGroup: openCreateGroupDialog
    }
)(
    class GroupsPanel extends React.Component<GroupsPanelProps> {

        render() {
            return (
                <DataExplorer
                    id={GROUPS_PANEL_ID}
                    onRowClick={noop}
                    onRowDoubleClick={noop}
                    onContextMenu={noop}
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
    });
