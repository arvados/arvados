// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Resource, ResourceKind } from '~/models/resource';

export type UserPrefs = {
    profile?: {
        organization?: string,
        organization_email?: string,
        lab?: string,
        website_url?: string,
        role?: string
    }
};

export interface User {
    email: string;
    firstName: string;
    lastName: string;
    uuid: string;
    ownerUuid: string;
    username: string;
    prefs: UserPrefs;
    isAdmin: boolean;
    isActive: boolean;
}

export const getUserFullname = (user?: User) => {
    return user ? `${user.firstName} ${user.lastName}` : "";
};

export interface UserResource extends Resource, User {
    kind: ResourceKind.USER;
    defaultOwnerUuid: string;
    writableBy: string[];
}
