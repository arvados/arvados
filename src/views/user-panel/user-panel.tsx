// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { WithStyles, withStyles, Tabs, Tab, Paper, Button, Grid } from '@material-ui/core';
import { DataExplorer } from "views-components/data-explorer/data-explorer";
import { connect, DispatchProp } from 'react-redux';
import { DataColumns } from 'components/data-table/data-table';
import { RootState } from 'store/store';
import { SortDirection } from 'components/data-table/data-column';
import { openContextMenu } from "store/context-menu/context-menu-actions";
import { getResource, ResourcesState } from "store/resources/resources";
import {
    ResourceFirstName,
    ResourceLastName,
    ResourceUuid,
    ResourceEmail,
    ResourceIsActive,
    ResourceIsAdmin,
    ResourceUsername
} from "views-components/data-explorer/renderers";
import { navigateTo } from "store/navigation/navigation-action";
import { ContextMenuKind } from "views-components/context-menu/context-menu";
import { DataTableDefaultView } from 'components/data-table-default-view/data-table-default-view';
import { createTree } from 'models/tree';
import { compose, Dispatch } from 'redux';
import { UserResource } from 'models/user';
import { ShareMeIcon, AddIcon } from 'components/icon/icon';
import { USERS_PANEL_ID, openUserCreateDialog } from 'store/users/users-actions';
import { noop } from 'lodash';

type UserPanelRules = "button";

const styles = withStyles<UserPanelRules>(theme => ({
    button: {
        marginTop: theme.spacing.unit,
        marginRight: theme.spacing.unit * 2,
        textAlign: 'right',
        alignSelf: 'center'
    },
}));

export enum UserPanelColumnNames {
    FIRST_NAME = "First Name",
    LAST_NAME = "Last Name",
    UUID = "Uuid",
    EMAIL = "Email",
    ACTIVE = "Active",
    ADMIN = "Admin",
    REDIRECT_TO_USER = "Redirect to user",
    USERNAME = "Username"
}

export const userPanelColumns: DataColumns<string> = [
    {
        name: UserPanelColumnNames.FIRST_NAME,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.NONE,
        filters: createTree(),
        render: uuid => <ResourceFirstName uuid={uuid} />
    },
    {
        name: UserPanelColumnNames.LAST_NAME,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.NONE,
        filters: createTree(),
        render: uuid => <ResourceLastName uuid={uuid} />
    },
    {
        name: UserPanelColumnNames.UUID,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.NONE,
        filters: createTree(),
        render: uuid => <ResourceUuid uuid={uuid} />
    },
    {
        name: UserPanelColumnNames.EMAIL,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.NONE,
        filters: createTree(),
        render: uuid => <ResourceEmail uuid={uuid} />
    },
    {
        name: UserPanelColumnNames.ACTIVE,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.NONE,
        filters: createTree(),
        render: uuid => <ResourceIsActive uuid={uuid} />
    },
    {
        name: UserPanelColumnNames.ADMIN,
        selected: true,
        configurable: false,
        sortDirection: SortDirection.NONE,
        filters: createTree(),
        render: uuid => <ResourceIsAdmin uuid={uuid} />
    },
    {
        name: UserPanelColumnNames.USERNAME,
        selected: true,
        configurable: false,
        sortDirection: SortDirection.NONE,
        filters: createTree(),
        render: uuid => <ResourceUsername uuid={uuid} />
    }
];

interface UserPanelDataProps {
    resources: ResourcesState;
}

interface UserPanelActionProps {
    openUserCreateDialog: () => void;
    handleRowDoubleClick: (uuid: string) => void;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, item: any) => void;
}

const mapStateToProps = (state: RootState) => {
    return {
        resources: state.resources
    };
};

const mapDispatchToProps = (dispatch: Dispatch) => ({
    openUserCreateDialog: () => dispatch<any>(openUserCreateDialog()),
    handleRowDoubleClick: (uuid: string) => dispatch<any>(navigateTo(uuid)),
    onContextMenu: (event: React.MouseEvent<HTMLElement>, item: any) => dispatch<any>(openContextMenu(event, item))
});

type UserPanelProps = UserPanelDataProps & UserPanelActionProps & DispatchProp & WithStyles<UserPanelRules>;

export const UserPanel = compose(
    styles,
    connect(mapStateToProps, mapDispatchToProps))(
        class extends React.Component<UserPanelProps> {
            state = {
                value: 0,
            };

            componentDidMount() {
                this.setState({ value: 0 });
            }

            render() {
                const { value } = this.state;
                return <Paper>
                    <Tabs value={value} onChange={this.handleChange} fullWidth>
                        <Tab label="USERS" />
                        <Tab label="ACTIVITY" disabled />
                    </Tabs>
                    {value === 0 &&
                        <span>
                            <DataExplorer
                                id={USERS_PANEL_ID}
                                onRowClick={noop}
                                onRowDoubleClick={noop}
                                onContextMenu={this.handleContextMenu}
                                contextMenuColumn={true}
                                hideColumnSelector
                                actions={
                                    <Grid container justify='flex-end'>
                                        <Button variant="contained" color="primary" onClick={this.props.openUserCreateDialog}>
                                            <AddIcon /> NEW USER
                                        </Button>
                                    </Grid>
                                }
                                paperProps={{
                                    elevation: 0,
                                }}
                                dataTableDefaultView={
                                    <DataTableDefaultView
                                        icon={ShareMeIcon}
                                        messages={['Your user list is empty.']} />
                                } />
                        </span>}
                </Paper>;
            }

            handleChange = (event: React.MouseEvent<HTMLElement>, value: number) => {
                this.setState({ value });
            }

            handleContextMenu = (event: React.MouseEvent<HTMLElement>, resourceUuid: string) => {
                const resource = getResource<UserResource>(resourceUuid)(this.props.resources);
                if (resource) {
                    this.props.onContextMenu(event, {
                        name: '',
                        uuid: resource.uuid,
                        ownerUuid: resource.ownerUuid,
                        kind: resource.kind,
                        menuKind: ContextMenuKind.USER
                    });
                }
            }
        }
    );
