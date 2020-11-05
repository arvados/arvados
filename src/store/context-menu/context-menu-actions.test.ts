// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as resource from '~/models/resource';
import { ContextMenuKind } from '~/views-components/context-menu/context-menu';
import { resourceKindToContextMenuKind } from './context-menu-actions';

describe('context-menu-actions', () => {
    describe('resourceKindToContextMenuKind', () => {
        const uuid = '123';

        describe('ResourceKind.PROJECT', () => {
            beforeEach(() => {
                // setup
                jest.spyOn(resource, 'extractUuidKind')
                    .mockImplementation(() => resource.ResourceKind.PROJECT);
            });

            it('should return ContextMenuKind.PROJECT_ADMIN', () => {
                // given
                const isAdmin = true;

                // when
                const result = resourceKindToContextMenuKind(uuid, isAdmin);

                // then
                expect(result).toEqual(ContextMenuKind.PROJECT_ADMIN);
            });

            it('should return ContextMenuKind.PROJECT', () => {
                // given
                const isAdmin = false;
                const isEditable = true;

                // when
                const result = resourceKindToContextMenuKind(uuid, isAdmin, isEditable);

                // then
                expect(result).toEqual(ContextMenuKind.PROJECT);
            });

            it('should return ContextMenuKind.READONLY_PROJECT', () => {
                // given
                const isAdmin = false;
                const isEditable = false;

                // when
                const result = resourceKindToContextMenuKind(uuid, isAdmin, isEditable);

                // then
                expect(result).toEqual(ContextMenuKind.READONLY_PROJECT);
            });
        });

        describe('ResourceKind.COLLECTION', () => {
            beforeEach(() => {
                // setup
                jest.spyOn(resource, 'extractUuidKind')
                    .mockImplementation(() => resource.ResourceKind.COLLECTION);
            });

            it('should return ContextMenuKind.COLLECTION_ADMIN', () => {
                // given
                const isAdmin = true;

                // when
                const result = resourceKindToContextMenuKind(uuid, isAdmin);

                // then
                expect(result).toEqual(ContextMenuKind.COLLECTION_ADMIN);
            });

            it('should return ContextMenuKind.COLLECTION', () => {
                // given
                const isAdmin = false;
                const isEditable = true;

                // when
                const result = resourceKindToContextMenuKind(uuid, isAdmin, isEditable);

                // then
                expect(result).toEqual(ContextMenuKind.COLLECTION);
            });

            it('should return ContextMenuKind.READONLY_COLLECTION', () => {
                // given
                const isAdmin = false;
                const isEditable = false;

                // when
                const result = resourceKindToContextMenuKind(uuid, isAdmin, isEditable);

                // then
                expect(result).toEqual(ContextMenuKind.READONLY_COLLECTION);
            });
        });

        describe('ResourceKind.PROCESS', () => {
            beforeEach(() => {
                // setup
                jest.spyOn(resource, 'extractUuidKind')
                    .mockImplementation(() => resource.ResourceKind.PROCESS);
            });

            it('should return ContextMenuKind.PROCESS_ADMIN', () => {
                // given
                const isAdmin = true;

                // when
                const result = resourceKindToContextMenuKind(uuid, isAdmin);

                // then
                expect(result).toEqual(ContextMenuKind.PROCESS_ADMIN);
            });

            it('should return ContextMenuKind.PROCESS_RESOURCE', () => {
                // given
                const isAdmin = false;

                // when
                const result = resourceKindToContextMenuKind(uuid, isAdmin);

                // then
                expect(result).toEqual(ContextMenuKind.PROCESS_RESOURCE);
            });
        });

        describe('ResourceKind.USER', () => {
            beforeEach(() => {
                // setup
                jest.spyOn(resource, 'extractUuidKind')
                    .mockImplementation(() => resource.ResourceKind.USER);
            });

            it('should return ContextMenuKind.ROOT_PROJECT', () => {
                // when
                const result = resourceKindToContextMenuKind(uuid);

                // then
                expect(result).toEqual(ContextMenuKind.ROOT_PROJECT);
            });
        });

        describe('ResourceKind.LINK', () => {
            beforeEach(() => {
                // setup
                jest.spyOn(resource, 'extractUuidKind')
                    .mockImplementation(() => resource.ResourceKind.LINK);
            });

            it('should return ContextMenuKind.LINK', () => {
                // when
                const result = resourceKindToContextMenuKind(uuid);

                // then
                expect(result).toEqual(ContextMenuKind.LINK);
            });
        });
    });
});