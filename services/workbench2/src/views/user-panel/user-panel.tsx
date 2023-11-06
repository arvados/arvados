// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { WithStyles, withStyles, Paper, Typography } from '@material-ui/core';
import { DataExplorer } from "views-components/data-explorer/data-explorer";
import { connect, DispatchProp } from 'react-redux';
import { DataColumns } from 'components/data-table/data-table';
import { RootState } from 'store/store';
import { SortDirection } from 'components/data-table/data-column';
import { openUserContextMenu } from "store/context-menu/context-menu-actions";
import { getResource, ResourcesState } from "store/resources/resources";
import {
    UserResourceFullName,
    ResourceUuid,
    ResourceEmail,
    ResourceIsAdmin,
    ResourceUsername,
    UserResourceAccountStatus,
} from "views-components/data-explorer/renderers";
import { navigateToUserProfile } from "store/navigation/navigation-action";
import { createTree } from 'models/tree';
import { compose, Dispatch } from 'redux';
import { UserResource } from 'models/user';
import { ShareMeIcon } from 'components/icon/icon';
import { USERS_PANEL_ID, openUserCreateDialog } from 'store/users/users-actions';
import { noop } from 'lodash';

type UserPanelRules = "button" | 'root';

const styles = withStyles<UserPanelRules>(theme => ({
    button: {
        marginTop: theme.spacing.unit,
        marginRight: theme.spacing.unit * 2,
        textAlign: 'right',
        alignSelf: 'center'
    },
    root: {
        width: '100%',
    },
}));

export enum UserPanelColumnNames {
    NAME = "Name",
    UUID = "Uuid",
    EMAIL = "Email",
    STATUS = "Account Status",
    ADMIN = "Admin",
    REDIRECT_TO_USER = "Redirect to user",
    USERNAME = "Username"
}

export const userPanelColumns: DataColumns<string, UserResource> = [
    {
        name: UserPanelColumnNames.NAME,
        selected: true,
        configurable: true,
        sort: {direction: SortDirection.NONE, field: "firstName"},
        filters: createTree(),
        render: uuid => <UserResourceFullName uuid={uuid} link={true} />
    },
    {
        name: UserPanelColumnNames.UUID,
        selected: true,
        configurable: true,
        sort: {direction: SortDirection.NONE, field: "uuid"},
        filters: createTree(),
        render: uuid => <ResourceUuid uuid={uuid} />
    },
    {
        name: UserPanelColumnNames.EMAIL,
        selected: true,
        configurable: true,
        sort: {direction: SortDirection.NONE, field: "email"},
        filters: createTree(),
        render: uuid => <ResourceEmail uuid={uuid} />
    },
    {
        name: UserPanelColumnNames.STATUS,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <UserResourceAccountStatus uuid={uuid} />
    },
    {
        name: UserPanelColumnNames.ADMIN,
        selected: true,
        configurable: false,
        filters: createTree(),
        render: uuid => <ResourceIsAdmin uuid={uuid} />
    },
    {
        name: UserPanelColumnNames.USERNAME,
        selected: true,
        configurable: false,
        sort: {direction: SortDirection.NONE, field: "username"},
        filters: createTree(),
        render: uuid => <ResourceUsername uuid={uuid} />
    }
];

interface UserPanelDataProps {
    resources: ResourcesState;
}

interface UserPanelActionProps {
    openUserCreateDialog: () => void;
    handleRowClick: (uuid: string) => void;
    handleContextMenu: (event, resource: UserResource) => void;
}

const mapStateToProps = (state: RootState) => {
    return {
        resources: state.resources
    };
};

const mapDispatchToProps = (dispatch: Dispatch) => ({
    openUserCreateDialog: () => dispatch<any>(openUserCreateDialog()),
    handleRowClick: (uuid: string) => dispatch<any>(navigateToUserProfile(uuid)),
    handleContextMenu: (event, resource: UserResource) => dispatch<any>(openUserContextMenu(event, resource)),
});

type UserPanelProps = UserPanelDataProps & UserPanelActionProps & DispatchProp & WithStyles<UserPanelRules>;

export const UserPanel = compose(
    styles,
    connect(mapStateToProps, mapDispatchToProps))(
        class extends React.Component<UserPanelProps> {
            render() {
                return <Paper className={this.props.classes.root}>
                    <DataExplorer
                        id={USERS_PANEL_ID}
                        title={
                            <>
                                <Typography>
                                    User records are created automatically on first log in.
                                </Typography>
                                <Typography>
                                    To add a new user, add them to your configured log in provider.
                                </Typography>
                            </>}
                        onRowClick={noop}
                        onRowDoubleClick={noop}
                        onContextMenu={this.handleContextMenu}
                        contextMenuColumn={true}
                        hideColumnSelector
                        paperProps={{
                            elevation: 0,
                        }}
                        defaultViewIcon={ShareMeIcon}
                        defaultViewMessages={['Your user list is empty.']} />
                </Paper>;
            }

            handleContextMenu = (event: React.MouseEvent<HTMLElement>, resourceUuid: string) => {
                event.stopPropagation();
                const resource = getResource<UserResource>(resourceUuid)(this.props.resources);
                if (resource) {
                    this.props.handleContextMenu(event, resource);
                }
            }
        }
    );
