// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { DataColumns, SortDirection } from 'components/data-table/data-column';
import {
    UserResourceFullName,
    ResourceUuid,
    ResourceEmail,
    ResourceIsAdmin,
    ResourceUsername,
    UserResourceAccountStatus
} from "views-components/data-explorer/renderers";
import { createTree } from 'models/tree';
import { UserResource } from 'models/user';


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
        sort: { direction: SortDirection.NONE, field: "firstName" },
        filters: createTree(),
        render: uuid => <UserResourceFullName uuid={uuid} link={true} />
    },
    {
        name: UserPanelColumnNames.UUID,
        selected: true,
        configurable: true,
        sort: { direction: SortDirection.NONE, field: "uuid" },
        filters: createTree(),
        render: uuid => <ResourceUuid uuid={uuid} />
    },
    {
        name: UserPanelColumnNames.EMAIL,
        selected: true,
        configurable: true,
        sort: { direction: SortDirection.NONE, field: "email" },
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
        sort: { direction: SortDirection.NONE, field: "username" },
        filters: createTree(),
        render: uuid => <ResourceUsername uuid={uuid} />
    }
];
