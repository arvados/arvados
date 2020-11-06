// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export const sortByProperty = (propName: string) => (obj1: any, obj2: any) => {
    const prop1 = obj1[propName];
    const prop2 = obj2[propName];
    
    if (prop1 > prop2) {
        return 1;
    }

    if (prop1 < prop2) {
        return -1;
    }

    return 0;
};
