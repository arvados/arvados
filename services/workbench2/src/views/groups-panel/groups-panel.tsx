// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { connect } from 'react-redux';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { Grid, Button } from "@mui/material";
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { DataExplorer } from "views-components/data-explorer/data-explorer";
import { DataColumns, SortDirection } from 'components/data-table/data-column';
import { GroupMembersCount, ResourceUuid } from 'views-components/data-explorer/renderers';
import { AddIcon } from 'components/icon/icon';
import { ResourceName } from 'views-components/data-explorer/renderers';
import { createTree } from 'models/tree';
import { GROUPS_PANEL_ID, openCreateGroupDialog } from 'store/groups-panel/groups-panel-actions';
import { noop } from 'lodash/fp';
import { ContextMenuKind } from 'views-components/context-menu/menu-item-sort';
import { getResource, ResourcesState } from 'store/resources/resources';
import { GroupResource } from 'models/group';
import { RootState } from 'store/store';
import { openContextMenu } from 'store/context-menu/context-menu-actions';
import { ArvadosTheme } from 'common/custom-theme';
import { loadDetailsPanel } from 'store/details-panel/details-panel-action';
import { toggleOne, deselectAllOthers } from 'store/multiselect/multiselect-actions';

type CssRules = "root";

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
    }
});

export enum GroupsPanelColumnNames {
    GROUP = "Name",
    UUID = "UUID",
    MEMBERS = "Members",
}

export const groupsPanelColumns: DataColumns<string, GroupResource> = [
    {
        name: GroupsPanelColumnNames.GROUP,
        selected: true,
        configurable: true,
        sort: {direction: SortDirection.ASC, field: "name"},
        filters: createTree(),
        render: uuid => <ResourceName uuid={uuid} />
    },
    {
        name: GroupsPanelColumnNames.UUID,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceUuid uuid={uuid} />,
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

const mapDispatchToProps = (dispatch: any) => {
    return {
        onContextMenu: (ev, resource) => dispatch(openContextMenu(ev, resource)),
        onNewGroup: () => dispatch(openCreateGroupDialog()),
        handleRowClick: (uuid: string) => {
            dispatch(toggleOne(uuid))
            dispatch(deselectAllOthers(uuid))
            dispatch(loadDetailsPanel(uuid));
        }
    };
};

export interface GroupsPanelProps {
    onNewGroup: () => void;
    handleRowClick: (uuid: string) => void;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, item: any) => void;
    resources: ResourcesState;
}

export const GroupsPanel = withStyles(styles)(connect(
    mapStateToProps, mapDispatchToProps
)(
    class GroupsPanel extends React.Component<GroupsPanelProps & WithStyles<CssRules>> {

        render() {
            return (
                <div className={this.props.classes.root}>
                    <DataExplorer
                        id={GROUPS_PANEL_ID}
                        data-cy="groups-panel-data-explorer"
                        onRowClick={this.props.handleRowClick}
                        onRowDoubleClick={noop}
                        onContextMenu={this.handleContextMenu}
                        contextMenuColumn={false}
                        hideColumnSelector
                        actions={
                            <Grid container justifyContent='flex-end'>
                                <Button
                                    data-cy="groups-panel-new-group"
                                    variant="contained"
                                    color="primary"
                                    onClick={this.props.onNewGroup}>
                                    <AddIcon /> New group
                                </Button>
                            </Grid>
                        } />
                    </div>
            );
        }

        handleContextMenu = (event: React.MouseEvent<HTMLElement>, resourceUuid: string) => {
            const resource = getResource<GroupResource>(resourceUuid)(this.props.resources);
            if (resource) {
                this.props.onContextMenu(event, {
                    name: resource.name,
                    uuid: resource.uuid,
                    description: resource.description,
                    ownerUuid: resource.ownerUuid,
                    kind: resource.kind,
                    menuKind: ContextMenuKind.GROUPS
                });
            }
        }
    }));
