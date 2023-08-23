// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

const USER_UUID_REGEX = /.*tpzed.*/;
const GROUP_UUID_REGEX = /.*-j7d0g-.*/;

export enum ObjectTypes {
    USER = "User",
    GROUP = "Group",
    UNKNOWN = "Unknown"
}

export const getUuidObjectType = (uuid: string) => {
    switch (true) {
        case USER_UUID_REGEX.test(uuid):
            return ObjectTypes.USER;
        case GROUP_UUID_REGEX.test(uuid):
            return ObjectTypes.GROUP;
        default:
            return ObjectTypes.UNKNOWN;
    }
};