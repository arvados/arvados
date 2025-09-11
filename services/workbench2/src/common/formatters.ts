// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { PropertyValue } from 'models/search-bar';
import {
    Vocabulary,
    getTagKeyLabel,
    getTagValueLabel,
} from 'models/vocabulary';
import moment from 'moment';

export const formatDateTime = (isoDate?: string | null, utc: boolean = false) => {
    if (isoDate) {
        const date = new Date(isoDate);
        let text: string;
        if (utc) {
            text = date.toUTCString();
        } else {
            text = date.toLocaleString();
        }
        return text === 'Invalid Date' ? '(none)' : text;
    }
    return '-';
};

export const formatDateOnly = (isoDate?: string | null, withTimeRemaining?: boolean) => {
    if (isoDate) {
        const date = new Date(isoDate);
        if (date) {
            const text = withTimeRemaining ? `${date.toLocaleDateString()} (${timeRemaining(isoDate)})` : date.toLocaleDateString();
            return text === 'Invalid Date' ? '(none)' : text;
        }
        return '-';
    }
    return '-';
};

export const timeRemaining = (targetDate: string | Date): string => {
    const now = moment();
    const end = moment(targetDate);

    if (end.isBefore(now)) return 'date is in the past';

    const years = end.diff(now, 'years');
    now.add(years, 'years');

    const months = end.diff(now, 'months');
    now.add(months, 'months');

    const days = end.diff(now, 'days');

    const parts: string[] = [];
    if (years > 0) parts.push(`in ${years} year${years > 1 ? 's' : ''}`);
    if (months > 0) parts.push(`${years > 0 ? '' : 'in '}${months} month${months > 1 ? 's' : ''}`);
    if (days > 0) parts.push(`${months > 0 ? '' : 'in '}${days} day${days > 1 ? 's' : ''}`);

    return parts.join(', ');
};

export const formatFileSize = (size?: number | string) => {
    if (typeof size === 'number') {
        if (size === 0) {
            return '0 B';
        }

        for (const { base, unit } of FILE_SIZES) {
            if (size >= base) {
                return `${(size / base).toFixed(base === 1 ? 0 : 1)} ${unit}`;
            }
        }
    }
    if ((typeof size === 'string' && size === '') || size === undefined) {
        return '-';
    }
    return '0 B';
};

export const formatCWLResourceSize = (size: number) => {
    return `${(size / CWL_SIZE.base).toFixed(0)} ${CWL_SIZE.unit}`;
};

export const formatTime = (time: number, seconds?: boolean) => {
    const minutes = Math.floor((time / (1000 * 60)) % 60).toFixed(0);
    const hours = Math.floor(time / (1000 * 60 * 60)).toFixed(0);

    if (seconds) {
        const seconds = Math.floor((time / 1000) % 60).toFixed(0);
        return hours + 'h ' + minutes + 'm ' + seconds + 's';
    }

    return hours + 'h ' + minutes + 'm';
};

export const getTimeDiff = (endTime: string, startTime: string) => {
    return new Date(endTime).getTime() - new Date(startTime).getTime();
};

export const formatProgress = (loaded: number, total: number) => {
    const progress = loaded >= 0 && total > 0 ? (loaded * 100) / total : 0;
    return `${progress.toFixed(2)}%`;
};

export function formatUploadSpeed(
    prevLoaded: number,
    loaded: number,
    prevTime: number,
    currentTime: number
) {
    const speed =
        loaded > prevLoaded && currentTime > prevTime
            ? (loaded - prevLoaded) / (currentTime - prevTime)
            : 0;

    return `${(speed / 1000).toFixed(2)} MB/s`;
}

const FILE_SIZES = [
    {
        base: 1024 ** 4,
        unit: 'TiB',
    },
    {
        base: 1024 ** 3,
        unit: 'GiB',
    },
    {
        base: 1024 ** 2,
        unit: 'MiB',
    },
    {
        base: 1024,
        unit: 'KiB',
    },
    {
        base: 1,
        unit: 'B',
    },
];

const CWL_SIZE = {
    base: 1024 ** 2,
    unit: 'MiB',
};

export const formatPropertyValue = (
    pv: PropertyValue,
    vocabulary?: Vocabulary
) => {
    if (vocabulary && pv.keyID && pv.valueID) {
        return `${getTagKeyLabel(pv.keyID, vocabulary)}: ${getTagValueLabel(
            pv.keyID,
            pv.valueID!,
            vocabulary
        )}`;
    }
    if (pv.key) {
        return pv.value ? `${pv.key}: ${pv.value}` : pv.key;
    }
    return '';
};

export const formatCost = (cost: number): string => {
    const decimalPlaces = 3;

    const factor = Math.pow(10, decimalPlaces);
    const rounded = Math.round(cost * factor) / factor;
    if (cost > 0 && rounded === 0) {
        // Display min value of 0.001
        return `$${1 / factor}`;
    } else {
        // Otherwise use rounded value to proper decimal places
        return `$${rounded}`;
    }
};
