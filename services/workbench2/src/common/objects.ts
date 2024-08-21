// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0
import { union, keys as keys_1, filter } from "lodash";

export function getModifiedKeys(a: any, b: any) {
    const keys = union(keys_1(a), keys_1(b));
    return filter(keys, key => a[key] !== b[key]);
}

export function getModifiedKeysValues(a: any, b: any) {
    const keys = getModifiedKeys(a, b);
    const obj = {};
    keys.forEach(k => {
        obj[k] = a[k];
    });
    return obj;
}

export function sortByKey<T>(arr: T[], key: string): T[] {
    return arr.sort((a, b) => {
        if (a[key] < b[key]) {
            return -1;
        }
        if (a[key] > b[key]) {
            return 1;
        }
        return 0; // If equal
    });
}
  
  