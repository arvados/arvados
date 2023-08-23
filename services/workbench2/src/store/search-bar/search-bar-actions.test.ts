// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { getAdvancedDataFromQuery, getQueryFromAdvancedData } from "store/search-bar/search-bar-actions";
import { ResourceKind } from "models/resource";

describe('search-bar-actions', () => {
    describe('getAdvancedDataFromQuery', () => {
        it('should correctly build advanced data record from query #1', () => {
            const r = getAdvancedDataFromQuery('val0 has:"file size":"100mb" val2 has:"user":"daniel" is:starred val2 val0');
            expect(r).toEqual({
                searchValue: 'val0 val2',
                type: undefined,
                cluster: undefined,
                projectUuid: undefined,
                inTrash: false,
                pastVersions: false,
                dateFrom: '',
                dateTo: '',
                properties: [{
                    key: 'file size',
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
            const r = getAdvancedDataFromQuery('document from:2017-08-01 pdf has:"filesize":"101mb" is:trashed type:arvados#collection cluster:c97qx is:pastVersion');
            expect(r).toEqual({
                searchValue: 'document pdf',
                type: ResourceKind.COLLECTION,
                cluster: 'c97qx',
                projectUuid: undefined,
                inTrash: true,
                pastVersions: true,
                dateFrom: '2017-08-01',
                dateTo: '',
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
                pastVersions: false,
                dateFrom: '2017-08-01',
                dateTo: '',
                properties: [
                    { key: 'file size', value: '101mb' },
                    { key: 'Species', value: 'Human' },
                    { key: 'Species', value: 'Canine' },
                ],
                saveQuery: false,
                queryName: ''
            });
            expect(q).toBe('document pdf type:arvados#collection cluster:c97qx is:trashed from:2017-08-01 has:"file size":"101mb" has:"Species":"Human" has:"Species":"Canine"');
        });

        it('should build query from advanced data #2', () => {
            const q = getQueryFromAdvancedData({
                searchValue: 'document pdf',
                type: ResourceKind.COLLECTION,
                cluster: 'c97qx',
                projectUuid: undefined,
                inTrash: false,
                pastVersions: true,
                dateFrom: '2017-08-01',
                dateTo: '',
                properties: [
                    { key: 'file size', value: '101mb' },
                    { key: 'Species', value: 'Human' },
                    { key: 'Species', value: 'Canine' },
                ],
                saveQuery: false,
                queryName: ''
            });
            expect(q).toBe('document pdf type:arvados#collection cluster:c97qx is:pastVersion from:2017-08-01 has:"file size":"101mb" has:"Species":"Human" has:"Species":"Canine"');
        });

        it('should add has:"key":"value" expression to query from same property key', () => {
            const searchValue = 'document pdf has:"file size":"101mb" has:"Species":"Canine"';
            const prevData = {
                searchValue,
                type: undefined,
                cluster: undefined,
                projectUuid: undefined,
                inTrash: false,
                pastVersions: false,
                dateFrom: '',
                dateTo: '',
                properties: [
                    { key: 'file size', value: '101mb' },
                    { key: 'Species', value: 'Canine' },
                ],
                saveQuery: false,
                queryName: ''
            };
            const currData = {
                ...prevData,
                properties: [
                    { key: 'file size', value: '101mb' },
                    { key: 'Species', value: 'Canine' },
                    { key: 'Species', value: 'Human' },
                ],
            };
            const q = getQueryFromAdvancedData(currData, prevData);
            expect(q).toBe('document pdf has:"file size":"101mb" has:"Species":"Canine" has:"Species":"Human"');
        });

        it('should add has:"keyID":"valueID" expression to query when necessary', () => {
            const searchValue = 'document pdf has:"file size":"101mb"';
            const prevData = {
                searchValue,
                type: undefined,
                cluster: undefined,
                projectUuid: undefined,
                inTrash: false,
                pastVersions: false,
                dateFrom: '',
                dateTo: '',
                properties: [
                    { key: 'file size', value: '101mb' },
                ],
                saveQuery: false,
                queryName: ''
            };
            const currData = {
                ...prevData,
                properties: [
                    { key: 'file size', value: '101mb' },
                    { key: 'Species', keyID: 'IDTAGSPECIES', value: 'Human', valueID: 'IDVALHUMAN'},
                ],
            };
            const q = getQueryFromAdvancedData(currData, prevData);
            expect(q).toBe('document pdf has:"file size":"101mb" has:"IDTAGSPECIES":"IDVALHUMAN"');
        });

        it('should remove has:"key":"value" expression from query', () => {
            const searchValue = 'document pdf has:"file size":"101mb" has:"Species":"Human" has:"Species":"Canine"';
            const prevData = {
                searchValue,
                type: undefined,
                cluster: undefined,
                projectUuid: undefined,
                inTrash: false,
                pastVersions: false,
                dateFrom: '',
                dateTo: '',
                properties: [
                    { key: 'file size', value: '101mb' },
                    { key: 'Species', value: 'Canine' },
                    { key: 'Species', value: 'Human' },
                ],
                saveQuery: false,
                queryName: ''
            };
            const currData = {
                ...prevData,
                properties: [
                    { key: 'file size', value: '101mb' },
                    { key: 'Species', value: 'Canine' },
                ],
            };
            const q = getQueryFromAdvancedData(currData, prevData);
            expect(q).toBe('document pdf has:"file size":"101mb" has:"Species":"Canine"');
        });
    });
});
