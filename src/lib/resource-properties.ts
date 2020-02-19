// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export const deleteProperty = (properties: any, key: string, value: string) => {
    if (Array.isArray(properties[key])) {
        properties[key] = properties[key].filter((v: string) => v !== value);
        if (properties[key].length === 1) {
            properties[key] = properties[key][0];
        } else if (properties[key].length === 0) {
            delete properties[key];
        }
    } else if (properties[key] === value) {
        delete properties[key];
    }
    return properties;
}

export const addProperty = (properties: any, key: string, value: string) => {
    if (properties[key]) {
        if (Array.isArray(properties[key])) {
            properties[key] = [...properties[key], value];
        } else {
            properties[key] = [properties[key], value];
        }
        // Remove potential duplicate and save as single value if needed
        properties[key] = Array.from(new Set(properties[key]));
        if (properties[key].length === 1) {
            properties[key] = properties[key][0];
        }
    } else {
        properties[key] = value;
    }
    return properties;
}