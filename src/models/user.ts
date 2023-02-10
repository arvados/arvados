// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Resource, ResourceKind, RESOURCE_UUID_REGEX } from 'models/resource';

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

export const getUserFullname = (user: User) => {
    return user.firstName && user.lastName
        ? `${user.firstName} ${user.lastName}`
        : "";
};

export const getUserDisplayName = (user: User, withEmail = false, withUuid = false) => {
    const displayName = getUserFullname(user) || user.email || user.username || user.uuid;
    let parts: string[] = [displayName];
    if (withEmail && user.email && displayName !== user.email) {
        parts.push(`<${user.email}>`);
    }
    if (withUuid) {
        parts.push(`(${user.uuid})`);
    }
    return parts.join(' ');
};

export const getUserDetailsString = (user: User) => {
    let parts: string[] = [];
    const userCluster = getUserClusterID(user);
    user.username.length && parts.push(user.username);
    user.email.length && parts.push(`<${user.email}>`);
    userCluster && userCluster.length && parts.push(`(${userCluster})`);
    return parts.join(' ');
};

export const getUserClusterID = (user: User): string | undefined => {
    const match = RESOURCE_UUID_REGEX.exec(user.uuid);
    const parts = match ? match[0].split('-') : [];
    return parts.length === 3 ? parts[0] : undefined;
};

export interface UserResource extends Resource, User {
    kind: ResourceKind.USER;
    defaultOwnerUuid: string;
    writableBy: string[];
}
