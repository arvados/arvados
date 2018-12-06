// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0
import * as _ from "lodash";

export function getModifiedKeys(a: any, b: any) {
    const keys = _.union(_.keys(a), _.keys(b));
    return _.filter(keys, key => a[key] !== b[key]);
}

export function getModifiedKeysValues(a: any, b: any) {
    const keys = getModifiedKeys(a, b);
    const obj = {};
    keys.forEach(k => {
        obj[k] = a[k];
    });
    return obj;
}
