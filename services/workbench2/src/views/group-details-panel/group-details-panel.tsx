// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { connect } from 'react-redux';
import { getResource } from 'store/resources/resources';
import { DataExplorer } from "views-components/data-explorer/data-explorer";
import { noop } from 'lodash/fp';
import { RootState } from 'store/store';
import { GROUP_DETAILS_MEMBERS_PANEL_ID, GROUP_DETAILS_PERMISSIONS_PANEL_ID, openAddGroupMembersDialog, getCurrentGroupDetailsPanelUuid } from 'store/group-details-panel/group-details-panel-actions';
import { openContextMenuAndSelect } from 'store/context-menu/context-menu-actions';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { Grid, Button, Tabs, Tab, Paper } from '@mui/material';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { AddIcon, UserPanelIcon, KeyIcon } from 'components/icon/icon';
import { GroupResource, isBuiltinGroup } from 'models/group';
import { ArvadosTheme } from 'common/custom-theme';
import { getUserUuid } from 'common/getuser';
import {
    GroupDetailsPanelMembersColumnNames,
    GroupDetailsPanelPermissionsColumnNames,
    groupDetailsMembersPanelColumns,
    groupDetailsPermissionsPanelColumns,
} from './group-details-panel-columns';

export { GroupDetailsPanelMembersColumnNames, GroupDetailsPanelPermissionsColumnNames, groupDetailsMembersPanelColumns, groupDetailsPermissionsPanelColumns };

type CssRules = "root" | "content";

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
    },
    content: {
        // reserve space for the tab bar
        height: `calc(100% - ${theme.spacing(7)})`,
    }
});

const MEMBERS_DEFAULT_MESSAGE = 'Members list is empty.';
const PERMISSIONS_DEFAULT_MESSAGE = 'Permissions list is empty.';

const mapStateToProps = (state: RootState) => {
    const groupUuid = getCurrentGroupDetailsPanelUuid(state.properties);
    const group = groupUuid ? getResource<GroupResource>(groupUuid)(state.resources) : undefined;
    const userUuid = getUserUuid(state);

    return {
        userUuid,
        group,
    };
};

const mapDispatchToProps = {
    onContextMenu: openContextMenuAndSelect,
    onAddUser: openAddGroupMembersDialog,
};

export interface GroupDetailsPanelProps {
    onContextMenu: (event: React.MouseEvent<HTMLElement>, item: any) => void;
    onAddUser: () => void;
    userUuid: string;
    group: GroupResource | undefined;
}

type GroupDetailsPanelState = {
    value: number;
    groupCanManage: boolean;
}

export const GroupDetailsPanel = withStyles(styles)(connect(
    mapStateToProps, mapDispatchToProps
)(
    class GroupDetailsPanel extends React.Component<GroupDetailsPanelProps & WithStyles<CssRules>> {
        state: GroupDetailsPanelState = {
            value: 0,
            groupCanManage: false,
        };

        componentDidMount() {
            this.setState({ value: 0 });
        }

        shouldComponentUpdate(nextProps: Readonly<GroupDetailsPanelProps>, nextState: Readonly<GroupDetailsPanelState>, nextContext: any): boolean {
            return this.props.group !== nextProps.group
                || this.state.value !== nextState.value
                || this.state.groupCanManage !== nextState.groupCanManage;
        }

        componentDidUpdate(prevProps: Readonly<GroupDetailsPanelProps>, prevState: Readonly<{}>, snapshot?: any): void {
            if (this.props.group && (prevProps.userUuid!== this.props.userUuid || prevProps.group !== this.props.group)) {
                this.setState({ groupCanManage: this.groupCanManage(this.props.userUuid, this.props.group) });
            }
        }

        groupCanManage = (userUuid: string, group: GroupResource) => {
            return userUuid && !isBuiltinGroup(group?.uuid || '') ? group.canManage : false
        }

        render() {
            const { value } = this.state;
            return (
                <Paper className={this.props.classes.root}>
                    <Tabs value={value} onChange={this.handleChange} variant="fullWidth">
                        <Tab data-cy="group-details-members-tab" label="MEMBERS" />
                        <Tab data-cy="group-details-permissions-tab" label="PERMISSIONS" />
                    </Tabs>
                    <div className={this.props.classes.content}>
                        {value === 0 &&
                            <DataExplorer
                                id={GROUP_DETAILS_MEMBERS_PANEL_ID}
                                data-cy="group-members-data-explorer"
                                onRowClick={noop}
                                onRowDoubleClick={noop}
                                onContextMenu={noop}
                                contextMenuColumn={false}
                                defaultViewIcon={UserPanelIcon}
                                defaultViewMessages={[MEMBERS_DEFAULT_MESSAGE]}
                                hideColumnSelector
                                hideSearchInput
                                actions={
                                    this.state.groupCanManage &&
                                    <Grid container justifyContent='flex-end'>
                                        <Button
                                            data-cy="group-member-add"
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
                                data-cy="group-permissions-data-explorer"
                                onRowClick={noop}
                                onRowDoubleClick={noop}
                                onContextMenu={noop}
                                contextMenuColumn={false}
                                defaultViewIcon={KeyIcon}
                                defaultViewMessages={[PERMISSIONS_DEFAULT_MESSAGE]}
                                hideColumnSelector
                                hideSearchInput
                                paperProps={{
                                    elevation: 0,
                                }} />
                        }
                    </div>
                </Paper>
            );
        }

        handleChange = (event: React.MouseEvent<HTMLElement>, value: number) => {
            this.setState({ value });
        }
    }));
