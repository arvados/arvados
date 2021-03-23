// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuKind } from '~/views-components/context-menu/context-menu';
import { resourceUuidToContextMenuKind } from './context-menu-actions';
import configureStore from 'redux-mock-store';
import thunk from 'redux-thunk';
import { PROJECT_PANEL_CURRENT_UUID } from '../project-panel/project-panel-action';
import { GroupClass } from '~/models/group';

describe('context-menu-actions', () => {
    describe('resourceUuidToContextMenuKind', () => {
        const middlewares = [thunk];
        const mockStore = configureStore(middlewares);
        const userUuid = 'zzzzz-tpzed-bbbbbbbbbbbbbbb';
        const otherUserUuid = 'zzzzz-tpzed-bbbbbbbbbbbbbbc';
        const headCollectionUuid = 'zzzzz-4zz18-aaaaaaaaaaaaaaa';
        const oldCollectionUuid = 'zzzzz-4zz18-aaaaaaaaaaaaaab';
        const projectUuid = 'zzzzz-j7d0g-ccccccccccccccc';
        const filterGroupUuid = 'zzzzz-j7d0g-ccccccccccccccd';
        const linkUuid = 'zzzzz-o0j2j-0123456789abcde';
        const containerRequestUuid = 'zzzzz-xvhdp-0123456789abcde';

        it('should return the correct menu kind', () => {
            const cases = [
                // resourceUuid, isAdminUser, isEditable, isTrashed, forceReadonly, expected
                [headCollectionUuid, false, true, true, false, ContextMenuKind.TRASHED_COLLECTION],
                [headCollectionUuid, false, true, false, false, ContextMenuKind.COLLECTION],
                [headCollectionUuid, false, true, false, true, ContextMenuKind.READONLY_COLLECTION],
                [headCollectionUuid, false, false, true, false, ContextMenuKind.READONLY_COLLECTION],
                [headCollectionUuid, false, false, false, false, ContextMenuKind.READONLY_COLLECTION],
                [headCollectionUuid, true, true, true, false, ContextMenuKind.TRASHED_COLLECTION],
                [headCollectionUuid, true, true, false, false, ContextMenuKind.COLLECTION_ADMIN],
                [headCollectionUuid, true, false, true, false, ContextMenuKind.TRASHED_COLLECTION],
                [headCollectionUuid, true, false, false, false, ContextMenuKind.COLLECTION_ADMIN],
                [headCollectionUuid, true, false, false, true, ContextMenuKind.READONLY_COLLECTION],

                [oldCollectionUuid, false, true, true, false, ContextMenuKind.OLD_VERSION_COLLECTION],
                [oldCollectionUuid, false, true, false, false, ContextMenuKind.OLD_VERSION_COLLECTION],
                [oldCollectionUuid, false, false, true, false, ContextMenuKind.OLD_VERSION_COLLECTION],
                [oldCollectionUuid, false, false, false, false, ContextMenuKind.OLD_VERSION_COLLECTION],
                [oldCollectionUuid, true, true, true, false, ContextMenuKind.OLD_VERSION_COLLECTION],
                [oldCollectionUuid, true, true, false, false, ContextMenuKind.OLD_VERSION_COLLECTION],
                [oldCollectionUuid, true, false, true, false, ContextMenuKind.OLD_VERSION_COLLECTION],
                [oldCollectionUuid, true, false, false, false, ContextMenuKind.OLD_VERSION_COLLECTION],

                // FIXME: WB2 doesn't currently have context menu for trashed projects
                // [projectUuid, false, true, true, false, ContextMenuKind.TRASHED_PROJECT],
                [projectUuid, false, true, false, false, ContextMenuKind.PROJECT],
                [projectUuid, false, true, false, true, ContextMenuKind.READONLY_PROJECT],
                [projectUuid, false, false, true, false, ContextMenuKind.READONLY_PROJECT],
                [projectUuid, false, false, false, false, ContextMenuKind.READONLY_PROJECT],
                // [projectUuid, true, true, true, false, ContextMenuKind.TRASHED_PROJECT],
                [projectUuid, true, true, false, false, ContextMenuKind.PROJECT_ADMIN],
                // [projectUuid, true, false, true, false, ContextMenuKind.TRASHED_PROJECT],
                [projectUuid, true, false, false, false, ContextMenuKind.PROJECT_ADMIN],
                [projectUuid, true, false, false, true, ContextMenuKind.READONLY_PROJECT],

                [linkUuid, false, true, true, false, ContextMenuKind.LINK],
                [linkUuid, false, true, false, false, ContextMenuKind.LINK],
                [linkUuid, false, false, true, false, ContextMenuKind.LINK],
                [linkUuid, false, false, false, false, ContextMenuKind.LINK],
                [linkUuid, true, true, true, false, ContextMenuKind.LINK],
                [linkUuid, true, true, false, false, ContextMenuKind.LINK],
                [linkUuid, true, false, true, false, ContextMenuKind.LINK],
                [linkUuid, true, false, false, false, ContextMenuKind.LINK],

                [userUuid, false, true, true, false, ContextMenuKind.ROOT_PROJECT],
                [userUuid, false, true, false, false, ContextMenuKind.ROOT_PROJECT],
                [userUuid, false, false, true, false, ContextMenuKind.ROOT_PROJECT],
                [userUuid, false, false, false, false, ContextMenuKind.ROOT_PROJECT],
                [userUuid, true, true, true, false, ContextMenuKind.ROOT_PROJECT],
                [userUuid, true, true, false, false, ContextMenuKind.ROOT_PROJECT],
                [userUuid, true, false, true, false, ContextMenuKind.ROOT_PROJECT],
                [userUuid, true, false, false, false, ContextMenuKind.ROOT_PROJECT],

                [containerRequestUuid, false, true, true, false, ContextMenuKind.PROCESS_RESOURCE],
                [containerRequestUuid, false, true, false, false, ContextMenuKind.PROCESS_RESOURCE],
                [containerRequestUuid, false, false, true, false, ContextMenuKind.PROCESS_RESOURCE],
                [containerRequestUuid, false, false, false, false, ContextMenuKind.PROCESS_RESOURCE],
                [containerRequestUuid, false, false, false, true, ContextMenuKind.READONLY_PROCESS_RESOURCE],
                [containerRequestUuid, true, true, true, false, ContextMenuKind.PROCESS_ADMIN],
                [containerRequestUuid, true, true, false, false, ContextMenuKind.PROCESS_ADMIN],
                [containerRequestUuid, true, false, true, false, ContextMenuKind.PROCESS_ADMIN],
                [containerRequestUuid, true, false, false, false, ContextMenuKind.PROCESS_ADMIN],
                [containerRequestUuid, true, false, false, true, ContextMenuKind.READONLY_PROCESS_RESOURCE],
            ]

            cases.forEach(([resourceUuid, isAdminUser, isEditable, isTrashed, forceReadonly, expected]) => {
                const initialState = {
                    properties: {
                        [PROJECT_PANEL_CURRENT_UUID]: projectUuid,
                    },
                    resources: {
                        [headCollectionUuid]: {
                            uuid: headCollectionUuid,
                            ownerUuid: projectUuid,
                            currentVersionUuid: headCollectionUuid,
                            isTrashed: isTrashed,
                        },
                        [oldCollectionUuid]: {
                            uuid: oldCollectionUuid,
                            currentVersionUuid: headCollectionUuid,
                            isTrashed: isTrashed,
                        },
                        [projectUuid]: {
                            uuid: projectUuid,
                            ownerUuid: isEditable ? userUuid : otherUserUuid,
                            writableBy: isEditable ? [userUuid] : [otherUserUuid],
                            groupClass: GroupClass.PROJECT,
                        },
                        [filterGroupUuid]: {
                            uuid: filterGroupUuid,
                            ownerUuid: isEditable ? userUuid : otherUserUuid,
                            writableBy: isEditable ? [userUuid] : [otherUserUuid],
                            groupClass: GroupClass.FILTER,
                        },
                        [linkUuid]: {
                            uuid: linkUuid,
                        },
                        [userUuid]: {
                            uuid: userUuid,
                        },
                        [containerRequestUuid]: {
                            uuid: containerRequestUuid,
                            ownerUuid: projectUuid,
                        },
                    },
                    auth: {
                        user: {
                            uuid: userUuid,
                            isAdmin: isAdminUser,
                        },
                    },
                };
                const store = mockStore(initialState);

                let menuKind: any;
                try {
                    menuKind = store.dispatch<any>(resourceUuidToContextMenuKind(resourceUuid as string, forceReadonly as boolean))
                    expect(menuKind).toBe(expected);
                } catch (err) {
                    throw new Error(`menuKind for resource ${JSON.stringify(initialState.resources[resourceUuid as string])} forceReadonly: ${forceReadonly} expected to be ${expected} but got ${menuKind}.`);
                }
            });
        });
    });
});
