// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize as originalUnionize, SingleValueRec } from 'unionize';

export * from 'unionize';

type TagRecord<Record> = { [T in keyof Record]: T };

export function unionize<Record extends SingleValueRec>(record: Record) {
    const tags = {} as TagRecord<Record>;
    for (const tag in record) {
        tags[tag] = tag;
    }
    return {...originalUnionize(record, {
        tag: 'type',
        value: 'payload'
    }), tags};
}
