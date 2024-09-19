// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { getUserDisplayName } from './user';

describe('User', () => {
    it('gets the user display name', () => {
        const testCases = [
            {
                caseName: 'Full data available',
                user: {
                    email: 'someuser@example.com', username: 'someuser',
                    firstName: 'Some', lastName: 'User',
                    uuid: 'zzzzz-tpzed-someusersuuid',
                    ownerUuid: 'zzzzz-tpzed-someusersowneruuid',
                    prefs: {}, isAdmin: false, isActive: true,
                    canWrite: false, canManage: false,
                },
                expect: 'Some User'
            },
            {
                caseName: 'Full data available (with email)',
                withEmail: true,
                user: {
                    email: 'someuser@example.com', username: 'someuser',
                    firstName: 'Some', lastName: 'User',
                    uuid: 'zzzzz-tpzed-someusersuuid',
                    ownerUuid: 'zzzzz-tpzed-someusersowneruuid',
                    prefs: {}, isAdmin: false, isActive: true,
                    canWrite: false, canManage: false
                },
                expect: 'Some User <someuser@example.com>'
            },
            {
                caseName: 'Missing first name',
                user: {
                    email: 'someuser@example.com', username: 'someuser',
                    firstName: '', lastName: 'User',
                    uuid: 'zzzzz-tpzed-someusersuuid',
                    ownerUuid: 'zzzzz-tpzed-someusersowneruuid',
                    prefs: {}, isAdmin: false, isActive: true,
                    canWrite: false, canManage: false,

                },
                expect: 'someuser@example.com'
            },
            {
                caseName: 'Missing last name',
                user: {
                    email: 'someuser@example.com', username: 'someuser',
                    firstName: 'Some', lastName: '',
                    uuid: 'zzzzz-tpzed-someusersuuid',
                    ownerUuid: 'zzzzz-tpzed-someusersowneruuid',
                    prefs: {}, isAdmin: false, isActive: true,
                    canWrite: false, canManage: false,
                },
                expect: 'someuser@example.com'
            },
            {
                caseName: 'Missing first & last names',
                user: {
                    email: 'someuser@example.com', username: 'someuser',
                    firstName: '', lastName: '',
                    uuid: 'zzzzz-tpzed-someusersuuid',
                    ownerUuid: 'zzzzz-tpzed-someusersowneruuid',
                    prefs: {}, isAdmin: false, isActive: true,
                    canWrite: false, canManage: false,
                },
                expect: 'someuser@example.com'
            },
            {
                caseName: 'Missing first & last names (with email)',
                withEmail: true,
                user: {
                    email: 'someuser@example.com', username: 'someuser',
                    firstName: '', lastName: '',
                    uuid: 'zzzzz-tpzed-someusersuuid',
                    ownerUuid: 'zzzzz-tpzed-someusersowneruuid',
                    prefs: {}, isAdmin: false, isActive: true,
                    canWrite: false, canManage: false,
                },
                expect: 'someuser@example.com'
            },
            {
                caseName: 'Missing first & last names, and email address',
                user: {
                    email: '', username: 'someuser',
                    firstName: '', lastName: '',
                    uuid: 'zzzzz-tpzed-someusersuuid',
                    ownerUuid: 'zzzzz-tpzed-someusersowneruuid',
                    prefs: {}, isAdmin: false, isActive: true,
                    canWrite: false, canManage: false,
                },
                expect: 'someuser'
            },
            {
                caseName: 'Missing first & last names, and email address (with email)',
                withEmail: true,
                user: {
                    email: '', username: 'someuser',
                    firstName: '', lastName: '',
                    uuid: 'zzzzz-tpzed-someusersuuid',
                    ownerUuid: 'zzzzz-tpzed-someusersowneruuid',
                    prefs: {}, isAdmin: false, isActive: true,
                    canWrite: false, canManage: false,
                },
                expect: 'someuser'
            },
            {
                caseName: 'Missing all data (should not happen)',
                user: {
                    email: '', username: '',
                    firstName: '', lastName: '',
                    uuid: 'zzzzz-tpzed-someusersuuid',
                    ownerUuid: 'zzzzz-tpzed-someusersowneruuid',
                    prefs: {}, isAdmin: false, isActive: true,
                    canWrite: false, canManage: false,
                },
                expect: 'zzzzz-tpzed-someusersuuid'
            },
        ];
        testCases.forEach(c => {
            const dispName = getUserDisplayName(c.user, c.withEmail);
            expect(dispName).to.equal(c.expect);
        })
    });
});
