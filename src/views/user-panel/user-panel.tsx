// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { WithStyles, withStyles, Typography } from '@material-ui/core';
import { DataExplorer } from "~/views-components/data-explorer/data-explorer";
import { connect, DispatchProp } from 'react-redux';
import { DataColumns } from '~/components/data-table/data-table';
import { RootState } from '~/store/store';
import { SortDirection } from '~/components/data-table/data-column';
import { openContextMenu } from "~/store/context-menu/context-menu-actions";
import { getResource, ResourcesState } from "~/store/resources/resources";
import {
    ResourceFirstName,
    ResourceLastName,
    ResourceUuid,
    ResourceEmail,
    ResourceIsActive,
    ResourceIsAdmin,
    ResourceUsername
} from "~/views-components/data-explorer/renderers";
import { navigateTo } from "~/store/navigation/navigation-action";
import { loadDetailsPanel } from "~/store/details-panel/details-panel-action";
import { ContextMenuKind } from "~/views-components/context-menu/context-menu";
import { DataTableDefaultView } from '~/components/data-table-default-view/data-table-default-view';
import { createTree } from '~/models/tree';
import { compose } from 'redux';
import { UserResource } from '~/models/user';
import { ShareMeIcon } from '~/components/icon/icon';
import { USERS_PANEL_ID } from '~/store/users/users-actions';

type UserPanelRules = "toolbar" | "button";

const styles = withStyles<UserPanelRules>(theme => ({
    toolbar: {
        paddingBottom: theme.spacing.unit * 3,
        textAlign: "right"
    },
    button: {
        marginLeft: theme.spacing.unit
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
        name: UserPanelColumnNames.REDIRECT_TO_USER,
        selected: true,
        configurable: false,
        sortDirection: SortDirection.NONE,
        filters: createTree(),
        render: () => <Typography noWrap>(none)</Typography>
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

type UserPanelProps = UserPanelDataProps & DispatchProp & WithStyles<UserPanelRules>;

export const UserPanel = compose(
    styles,
    connect((state: RootState) => ({
        resources: state.resources
    })))(
        class extends React.Component<UserPanelProps> {
            render() {
                return <DataExplorer
                    id={USERS_PANEL_ID}
                    onRowClick={this.handleRowClick}
                    onRowDoubleClick={this.handleRowDoubleClick}
                    onContextMenu={this.handleContextMenu}
                    contextMenuColumn={true}
                    isColumnSelectorHidden={true}
                    dataTableDefaultView={
                        <DataTableDefaultView
                            icon={ShareMeIcon}
                            messages={['Your user list is empty.']} />
                    } />;
            }

            handleContextMenu = (event: React.MouseEvent<HTMLElement>, resourceUuid: string) => {
                const resource = getResource<UserResource>(resourceUuid)(this.props.resources);
                if (resource) {
                    this.props.dispatch<any>(openContextMenu(event, {
                        name: '',
                        uuid: resource.uuid,
                        ownerUuid: resource.ownerUuid,
                        kind: resource.kind,
                        menuKind: ContextMenuKind.USER
                    }));
                }
            }

            handleRowDoubleClick = (uuid: string) => {
                this.props.dispatch<any>(navigateTo(uuid));
            }

            handleRowClick = (uuid: string) => {
                this.props.dispatch(loadDetailsPanel(uuid));
            }
        }
    );
