// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { formatUploadSpeed } from "./formatters";

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