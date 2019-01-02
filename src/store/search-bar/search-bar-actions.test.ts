// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { getAdvancedDataFromQuery, getQueryFromAdvancedData, parseSearchQuery } from "~/store/search-bar/search-bar-actions";
import { ResourceKind } from "~/models/resource";

describe('search-bar-actions', () => {
    describe('parseSearchQuery', () => {
        it('should correctly parse query #1', () => {
            const q = 'val0 is:trashed val1';
            const r = parseSearchQuery(q);
            expect(r.hasKeywords).toBeTruthy();
            expect(r.values).toEqual(['val0', 'val1']);
            expect(r.properties).toEqual({
                is: ['trashed']
            });
        });

        it('should correctly parse query #2 (value with keyword should be ignored)', () => {
            const q = 'val0 is:from:trashed val1';
            const r = parseSearchQuery(q);
            expect(r.hasKeywords).toBeTruthy();
            expect(r.values).toEqual(['val0', 'val1']);
            expect(r.properties).toEqual({
                from: ['trashed']
            });
        });

        it('should correctly parse query #3 (many keywords)', () => {
            const q = 'val0 is:trashed val2 from:2017-04-01 val1';
            const r = parseSearchQuery(q);
            expect(r.hasKeywords).toBeTruthy();
            expect(r.values).toEqual(['val0', 'val2', 'val1']);
            expect(r.properties).toEqual({
                is: ['trashed'],
                from: ['2017-04-01']
            });
        });

        it('should correctly parse query #4 (no duplicated values)', () => {
            const q = 'val0 is:trashed val2 val2 val0';
            const r = parseSearchQuery(q);
            expect(r.hasKeywords).toBeTruthy();
            expect(r.values).toEqual(['val0', 'val2']);
            expect(r.properties).toEqual({
                is: ['trashed']
            });
        });

        it('should correctly parse query #5 (properties)', () => {
            const q = 'val0 has:filesize:100mb val2 val2 val0';
            const r = parseSearchQuery(q);
            expect(r.hasKeywords).toBeTruthy();
            expect(r.values).toEqual(['val0', 'val2']);
            expect(r.properties).toEqual({
                'has': ['filesize:100mb']
            });
        });

        it('should correctly parse query #6 (multiple properties, multiple is)', () => {
            const q = 'val0 has:filesize:100mb val2 has:user:daniel is:starred val2 val0 is:trashed';
            const r = parseSearchQuery(q);
            expect(r.hasKeywords).toBeTruthy();
            expect(r.values).toEqual(['val0', 'val2']);
            expect(r.properties).toEqual({
                'has': ['filesize:100mb', 'user:daniel'],
                'is': ['starred', 'trashed']
            });
        });
    });

    describe('getAdvancedDataFromQuery', () => {
        it('should correctly build advanced data record from query #1', () => {
            const r = getAdvancedDataFromQuery('val0 has:filesize:100mb val2 has:user:daniel is:starred val2 val0 is:trashed');
            expect(r).toEqual({
                searchValue: 'val0 val2',
                type: undefined,
                cluster: undefined,
                projectUuid: undefined,
                inTrash: true,
                dateFrom: undefined,
                dateTo: undefined,
                properties: [{
                    key: 'filesize',
                    value: '100mb'
                }, {
                    key: 'user',
                    value: 'daniel'
                }],
                saveQuery: false,
                queryName: ''
            });
        });

        it('should correctly build advanced data record from query #2', () => {
            const r = getAdvancedDataFromQuery('document from:2017-08-01 pdf has:filesize:101mb is:trashed type:arvados#collection cluster:c97qx');
            expect(r).toEqual({
                searchValue: 'document pdf',
                type: ResourceKind.COLLECTION,
                cluster: 'c97qx',
                projectUuid: undefined,
                inTrash: true,
                dateFrom: '2017-08-01',
                dateTo: undefined,
                properties: [{
                    key: 'filesize',
                    value: '101mb'
                }],
                saveQuery: false,
                queryName: ''
            });
        });
    });

    describe('getQueryFromAdvancedData', () => {
        it('should build query from advanced data', () => {
            const q = getQueryFromAdvancedData({
                searchValue: 'document pdf',
                type: ResourceKind.COLLECTION,
                cluster: 'c97qx',
                projectUuid: undefined,
                inTrash: true,
                dateFrom: '2017-08-01',
                dateTo: '',
                properties: [{
                    key: 'filesize',
                    value: '101mb'
                }],
                saveQuery: false,
                queryName: ''
            });
            expect(q).toBe('document pdf type:arvados#collection cluster:c97qx is:trashed from:2017-08-01 has:filesize:101mb');
        });
    });
});
