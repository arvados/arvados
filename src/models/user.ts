// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export interface User {
    email: string;
    firstName: string;
    lastName: string;
    uuid: string;
    ownerUuid: string;
}

export const getUserFullname = (user?: User) => {
    return user ? `${user.firstName} ${user.lastName}` : "";
};