// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { PropertyValue } from "~/models/search-bar";
import { Vocabulary, getTagKeyLabel, getTagValueLabel } from "~/models/vocabulary";

export const formatDate = (isoDate?: string | null, utc: boolean = false) => {
    if (isoDate) {
        const date = new Date(isoDate);
        let text: string;
        if (utc) {
            text = date.toUTCString();
        }
        else {
            text = date.toLocaleString();
        }
        return text === 'Invalid Date' ? "(none)" : text;
    }
    return "(none)";
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

export const formatTime = (time: number, seconds?: boolean) => {
    const minutes = Math.floor(time / (1000 * 60) % 60).toFixed(0);
    const hours = Math.floor(time / (1000 * 60 * 60)).toFixed(0);

    if (seconds) {
        const seconds = Math.floor(time / (1000) % 60).toFixed(0);
        return hours + "h " + minutes + "m " + seconds + "s";
    }

    return hours + "h " + minutes + "m";
};

export const getTimeDiff = (endTime: string, startTime: string) => {
    return new Date(endTime).getTime() - new Date(startTime).getTime();
};

export const formatProgress = (loaded: number, total: number) => {
    const progress = loaded >= 0 && total > 0 ? loaded * 100 / total : 0;
    return `${progress.toFixed(2)}%`;
};

export function formatUploadSpeed(prevLoaded: number, loaded: number, prevTime: number, currentTime: number) {
    const speed = loaded > prevLoaded && currentTime > prevTime
        ? (loaded - prevLoaded) / (currentTime - prevTime)
        : 0;
    return `${(speed / 1000).toFixed(2)} KB/s`;
}

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

export const formatPropertyValue = (pv: PropertyValue, vocabulary?: Vocabulary) => {
    if (vocabulary && pv.keyID && pv.valueID) {
        return `${getTagKeyLabel(pv.keyID, vocabulary)}: ${getTagValueLabel(pv.keyID, pv.valueID!, vocabulary)}`;
    }
    if (pv.key) {
        return pv.value
            ? `${pv.key}: ${pv.value}`
            : pv.key;
    }
    return "";
};
