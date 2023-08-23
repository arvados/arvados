// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { getResourceWithEditableStatus } from "./resources";
import { ResourceKind } from "models/resource";

const groupFixtures = {
    user_uuid: 'zzzzz-tpzed-0123456789ab789',
    user_resource_uuid: 'zzzzz-tpzed-0123456789abcde',
    unknown_user_resource_uuid: 'zzzzz-tpzed-0123456789ab987',
    editable_collection_resource_uuid: 'zzzzz-4zz18-0123456789ab456',
    not_editable_collection_resource_uuid: 'zzzzz-4zz18-0123456789ab654',
    editable_project_resource_uuid: 'zzzzz-j7d0g-0123456789ab123',
    not_editable_project_resource_uuid: 'zzzzz-j7d0g-0123456789ab321',
};

describe('resources', () => {
    describe('getResourceWithEditableStatus', () => {
        const resourcesState = {
            [groupFixtures.editable_project_resource_uuid]: {
                uuid: groupFixtures.editable_project_resource_uuid,
                ownerUuid: groupFixtures.user_uuid,
                createdAt: 'string',
                modifiedByClientUuid: 'string',
                modifiedByUserUuid: 'string',
                modifiedAt: 'string',
                href: 'string',
                kind: ResourceKind.PROJECT,
                writableBy: [groupFixtures.user_uuid],
                etag: 'string',
            },
            [groupFixtures.editable_collection_resource_uuid]: {
                uuid: groupFixtures.editable_collection_resource_uuid,
                ownerUuid: groupFixtures.editable_project_resource_uuid,
                createdAt: 'string',
                modifiedByClientUuid: 'string',
                modifiedByUserUuid: 'string',
                modifiedAt: 'string',
                href: 'string',
                kind: ResourceKind.COLLECTION,
                etag: 'string',
            },
            [groupFixtures.not_editable_project_resource_uuid]: {
                uuid: groupFixtures.not_editable_project_resource_uuid,
                ownerUuid: groupFixtures.unknown_user_resource_uuid,
                createdAt: 'string',
                modifiedByClientUuid: 'string',
                modifiedByUserUuid: 'string',
                modifiedAt: 'string',
                href: 'string',
                kind: ResourceKind.PROJECT,
                writableBy: [groupFixtures.unknown_user_resource_uuid],
                etag: 'string',
            },
            [groupFixtures.not_editable_collection_resource_uuid]: {
                uuid: groupFixtures.not_editable_collection_resource_uuid,
                ownerUuid: groupFixtures.not_editable_project_resource_uuid,
                createdAt: 'string',
                modifiedByClientUuid: 'string',
                modifiedByUserUuid: 'string',
                modifiedAt: 'string',
                href: 'string',
                kind: ResourceKind.COLLECTION,
                etag: 'string',
            },
            [groupFixtures.user_resource_uuid]: {
                uuid: groupFixtures.user_resource_uuid,
                ownerUuid: groupFixtures.user_resource_uuid,
                createdAt: 'string',
                modifiedByClientUuid: 'string',
                modifiedByUserUuid: 'string',
                modifiedAt: 'string',
                href: 'string',
                kind: ResourceKind.USER,
                etag: 'string',
            }
        };

        it('should return editable user resource (resource UUID is equal to user UUID)', () => {
            // given
            const id = groupFixtures.user_resource_uuid;
            const userUuid = groupFixtures.user_resource_uuid;

            // when
            const result = getResourceWithEditableStatus(id, userUuid)(resourcesState);

            // then
            expect(result!.isEditable).toBeTruthy();
        });

        it('should return editable project resource', () => {
            // given
            const id = groupFixtures.editable_project_resource_uuid;
            const userUuid = groupFixtures.user_uuid;

            // when
            const result = getResourceWithEditableStatus(id, userUuid)(resourcesState);

            // then
            expect(result!.isEditable).toBeTruthy();
        });

        it('should return editable collection resource', () => {
            // given
            const id = groupFixtures.editable_collection_resource_uuid;
            const userUuid = groupFixtures.user_uuid;

            // when
            const result = getResourceWithEditableStatus(id, userUuid)(resourcesState);

            // then
            expect(result!.isEditable).toBeTruthy();
        });

        it('should return not editable project resource', () => {
            // given
            const id = groupFixtures.not_editable_project_resource_uuid;
            const userUuid = groupFixtures.user_uuid;

            // when
            const result = getResourceWithEditableStatus(id, userUuid)(resourcesState);

            // then
            expect(result!.isEditable).toBeFalsy();
        });

        it('should return not editable collection resource', () => {
            // given
            const id = groupFixtures.not_editable_collection_resource_uuid;
            const userUuid = groupFixtures.user_uuid;

            // when
            const result = getResourceWithEditableStatus(id, userUuid)(resourcesState);

            // then
            expect(result!.isEditable).toBeFalsy();
        });
    });
});