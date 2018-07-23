// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export const formatDate = (isoDate: string) => {
    const date = new Date(isoDate);
    const text = date.toLocaleString();
    return text === 'Invalid Date' ? "" : text;
};

export const formatFileSize = (size?: number) => {
    if (typeof size === "number") {
        for (const { base, unit } of FILE_SIZES) {
            if (size >= base) {
                return `${(size / base).toFixed()} ${unit}`;
            }
        }
    }
    return "";
};

const FILE_SIZES = [
    {
        base: 1000000000000,
        unit: "TB"
    },
    {
        base: 1000000000,
        unit: "GB"
    },
    {
        base: 1000000,
        unit: "MB"
    },
    {
        base: 1000,
        unit: "KB"
    },
    {
        base: 1,
        unit: "B"
    }
];