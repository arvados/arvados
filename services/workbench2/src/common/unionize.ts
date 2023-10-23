// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize as originalUnionize, SingleValueRec } from 'unionize';

export * from 'unionize';

export function unionize<Record extends SingleValueRec>(record: Record) {
    return originalUnionize(record, {
        tag: 'type',
        value: 'payload'
    });
}

