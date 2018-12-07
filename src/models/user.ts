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
    identityUrl: string;
    prefs: UserPrefs;
    isAdmin: boolean;
}

export const getUserFullname = (user?: User) => {
    return user ? `${user.firstName} ${user.lastName}` : "";
};

export interface UserResource extends Resource {
    kind: ResourceKind.USER;
    email: string;
    username: string;
    firstName: string;
    lastName: string;
    identityUrl: string;
    isAdmin: boolean;
    prefs: UserPrefs;
    defaultOwnerUuid: string;
    isActive: boolean;
    writableBy: string[];
}