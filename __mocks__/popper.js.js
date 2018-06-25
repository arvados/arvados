// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export default class Popper {
    static placements = [
        'auto',
        'auto-end',
        'auto-start',
        'bottom',
        'bottom-end',
        'bottom-start',
        'left',
        'left-end',
        'left-start',
        'right',
        'right-end',
        'right-start',
        'top',
        'top-end',
        'top-start'
    ];

    constructor() {
        return {
            destroy: jest.fn(),
            scheduleUpdate: jest.fn()
        };
    }
}