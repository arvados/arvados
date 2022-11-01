// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { formatUploadSpeed, formatContainerCost } from "./formatters";

describe('formatUploadSpeed', () => {
    it('should show speed less than 1MB/s', () => {
        // given
        const speed = 900;

        // when
        const result = formatUploadSpeed(0, speed, 0, 1);

        // then
        expect(result).toBe('0.90 MB/s');
    });

    it('should show 5MB/s', () => {
        // given
        const speed = 5230;

        // when
        const result = formatUploadSpeed(0, speed, 0, 1);

        // then
        expect(result).toBe('5.23 MB/s');
    });
});

describe('formatContainerCost', () => {
    it('should correctly round to tenth of a cent', () => {
        expect(formatContainerCost(0.0)).toBe('$0');
        expect(formatContainerCost(0.125)).toBe('$0.125');
        expect(formatContainerCost(0.1254)).toBe('$0.125');
        expect(formatContainerCost(0.1255)).toBe('$0.126');
    });

    it('should round up any smaller value to 0.001', () => {
        expect(formatContainerCost(0.0)).toBe('$0');
        expect(formatContainerCost(0.001)).toBe('$0.001');
        expect(formatContainerCost(0.0001)).toBe('$0.001');
        expect(formatContainerCost(0.00001)).toBe('$0.001');
    });
});
