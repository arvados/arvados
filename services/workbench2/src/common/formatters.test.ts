// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { formatFileSize, formatUploadSpeed, formatCost, formatCWLResourceSize } from "./formatters";

describe('formatFileSize', () => {
    it('should pick the largest unit', () => {
        const base = 1024;
        const testCases = [
            {input: 0, output: '0 B'},
            {input: 1, output: '1 B'},
            {input: 1023, output: '1023 B'},
            {input: base, output: '1.0 KiB'},
            {input: 1.1 * base, output: '1.1 KiB'},
            {input: 1.5 * base, output: '1.5 KiB'},
            {input: base ** 2, output: '1.0 MiB'},
            {input: 1.5 * (base ** 2), output: '1.5 MiB'},
            {input: base ** 3, output: '1.0 GiB'},
            {input: base ** 4, output: '1.0 TiB'},
        ];

        for (const { input, output } of testCases) {
            expect(formatFileSize(input)).toBe(output);
        }
    });

    it('should handle accidental empty string or undefined input', () => {
        expect(formatFileSize('')).toBe('-');
        expect(formatFileSize(undefined)).toBe('-');
    });

    it('should handle accidental non-empty string input', () => {
        expect(formatFileSize('foo')).toBe('0 B');
    });
});

describe('formatCWLResourceSize', () => {
    it('should format bytes as MiB', () => {
        const base = 1024 ** 2;

        const testCases = [
            {input: 0, output: '0 MiB'},
            {input: 1, output: '0 MiB'},
            {input: base - 1, output: '1 MiB'},
            {input: 2 * base, output: '2 MiB'},
            {input: 1024 * base, output: '1024 MiB'},
            {input: 10000 * base, output: '10000 MiB'},
        ];

        for (const { input, output } of testCases) {
            expect(formatCWLResourceSize(input)).toBe(output);
        }
    });
});

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
        expect(formatCost(0.0)).toBe('$0');
        expect(formatCost(0.125)).toBe('$0.125');
        expect(formatCost(0.1254)).toBe('$0.125');
        expect(formatCost(0.1255)).toBe('$0.126');
    });

    it('should round up any smaller value to 0.001', () => {
        expect(formatCost(0.0)).toBe('$0');
        expect(formatCost(0.001)).toBe('$0.001');
        expect(formatCost(0.0001)).toBe('$0.001');
        expect(formatCost(0.00001)).toBe('$0.001');
    });
});
