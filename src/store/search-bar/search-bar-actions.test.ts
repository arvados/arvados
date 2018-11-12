// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { parseQuery } from "~/store/search-bar/search-bar-actions";

describe('search-bar-actions', () => {
    it('should correctly parse query #1', () => {
        const q = 'val0 is:trashed val1';
        const r = parseQuery(q);
        expect(r.hasKeywords).toBeTruthy();
        expect(r.values).toEqual(['val0', 'val1']);
        expect(r.properties).toEqual({
            is: 'trashed'
        });
    });

    it('should correctly parse query #2 (value with keyword should be ignored)', () => {
        const q = 'val0 is:from:trashed val1';
        const r = parseQuery(q);
        expect(r.hasKeywords).toBeTruthy();
        expect(r.values).toEqual(['val0', 'val1']);
        expect(r.properties).toEqual({
            from: 'trashed'
        });
    });

    it('should correctly parse query #3 (many keywords)', () => {
        const q = 'val0 is:trashed val2 from:2017-04-01 val1';
        const r = parseQuery(q);
        expect(r.hasKeywords).toBeTruthy();
        expect(r.values).toEqual(['val0', 'val2', 'val1']);
        expect(r.properties).toEqual({
            is: 'trashed',
            from: '2017-04-01'
        });
    });

    it('should correctly parse query #4 (no duplicated values)', () => {
        const q = 'val0 is:trashed val2 val2 val0';
        const r = parseQuery(q);
        expect(r.hasKeywords).toBeTruthy();
        expect(r.values).toEqual(['val0', 'val2']);
        expect(r.properties).toEqual({
            is: 'trashed'
        });
    });
});
