// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import axios, { AxiosInstance } from 'axios';
import MockAdapter from 'axios-mock-adapter';
import { snakeCase } from 'lodash';
import { CollectionResource, defaultCollectionSelectedFields } from 'models/collection';
import { AuthService } from '../auth-service/auth-service';
import { CollectionService } from './collection-service';

describe('collection-service', () => {
    let collectionService: CollectionService;
    let serverApi: AxiosInstance;
    let axiosMock: MockAdapter;
    let webdavClient: any;
    let authService;
    let actions;

    beforeEach(() => {
        serverApi = axios.create();
        axiosMock = new MockAdapter(serverApi);
        webdavClient = {
            delete: jest.fn(),
            upload: jest.fn(),
        } as any;
        authService = {} as AuthService;
        actions = {
            progressFn: jest.fn(),
        } as any;
        collectionService = new CollectionService(serverApi, webdavClient, authService, actions);
        collectionService.update = jest.fn();
    });

    describe('get', () => {
        it('should make a request with default selected fields', async () => {
            serverApi.get = jest.fn(() => Promise.resolve(
                { data: { items: [{}] } }
            ));
            const uuid = 'zzzzz-4zz18-0123456789abcde'
            await collectionService.get(uuid);
            expect(serverApi.get).toHaveBeenCalledWith(
                `/collections/${uuid}`, {
                    params: {
                        select: JSON.stringify(defaultCollectionSelectedFields.map(snakeCase)),
                    },
                }
            );
        });

        it('should be able to request specific fields', async () => {
            serverApi.get = jest.fn(() => Promise.resolve(
                { data: { items: [{}] } }
            ));
            const uuid = 'zzzzz-4zz18-0123456789abcde'
            await collectionService.get(uuid, undefined, ['manifestText']);
            expect(serverApi.get).toHaveBeenCalledWith(
                `/collections/${uuid}`, {
                    params: {
                        select: `["manifest_text"]`
                    },
                }
            );
        });
    });

    describe('update', () => {
        it('should call put selecting updated fields + others', async () => {
            serverApi.put = jest.fn(() => Promise.resolve({ data: {} }));
            const data: Partial<CollectionResource> = {
                name: 'foo',
            };
            const expected = {
                collection: {
                    ...data,
                    preserve_version: true,
                },
                select: ['uuid', 'name', 'version', 'modified_at'],
            }
            collectionService = new CollectionService(serverApi, webdavClient, authService, actions);
            await collectionService.update('uuid', data);
            expect(serverApi.put).toHaveBeenCalledWith('/collections/uuid', expected);
        });
    });

    describe('uploadFiles', () => {
        it('should skip if no files to upload files', async () => {
            // given
            const files: File[] = [];
            const collectionUUID = '';

            // when
            await collectionService.uploadFiles(collectionUUID, files);

            // then
            expect(webdavClient.upload).not.toHaveBeenCalled();
        });

        it('should upload files', async () => {
            // given
            const files: File[] = [{name: 'test-file1'} as File];
            const collectionUUID = 'zzzzz-4zz18-0123456789abcde';

            // when
            await collectionService.uploadFiles(collectionUUID, files);

            // then
            expect(webdavClient.upload).toHaveBeenCalledTimes(1);
            expect(webdavClient.upload.mock.calls[0][0]).toEqual("c=zzzzz-4zz18-0123456789abcde/test-file1");
        });

        it('should upload files with custom uplaod target', async () => {
            // given
            const files: File[] = [{name: 'test-file1'} as File];
            const collectionUUID = 'zzzzz-4zz18-0123456789abcde';
            const customTarget = 'zzzzz-4zz18-0123456789adddd/test-path/'

            // when
            await collectionService.uploadFiles(collectionUUID, files, undefined, customTarget);

            // then
            expect(webdavClient.upload).toHaveBeenCalledTimes(1);
            expect(webdavClient.upload.mock.calls[0][0]).toEqual("c=zzzzz-4zz18-0123456789adddd/test-path/test-file1");
        });
    });

    describe('deleteFiles', () => {
        it('should remove no files', async () => {
            // given
            const filePaths: string[] = [];
            const collectionUUID = '';

            // when
            await collectionService.deleteFiles(collectionUUID, filePaths);

            // then
            expect(webdavClient.delete).not.toHaveBeenCalled();
        });

        it('should remove only root files', async () => {
            // given
            const filePaths: string[] = ['/root/1', '/root/1/100', '/root/1/100/test.txt', '/root/2', '/root/2/200', '/root/3/300/test.txt'];
            const collectionUUID = '';

            // when
            await collectionService.deleteFiles(collectionUUID, filePaths);

            // then
            expect(webdavClient.delete).toHaveBeenCalledTimes(3);
            expect(webdavClient.delete).toHaveBeenCalledWith("c=/root/3/300/test.txt");
            expect(webdavClient.delete).toHaveBeenCalledWith("c=/root/2");
            expect(webdavClient.delete).toHaveBeenCalledWith("c=/root/1");
        });

        it('should remove files with uuid prefix', async () => {
            // given
            const filePaths: string[] = ['/root/1'];
            const collectionUUID = 'zzzzz-tpzed-5o5tg0l9a57gxxx';

            // when
            await collectionService.deleteFiles(collectionUUID, filePaths);

            // then
            expect(webdavClient.delete).toHaveBeenCalledTimes(1);
            expect(webdavClient.delete).toHaveBeenCalledWith("c=zzzzz-tpzed-5o5tg0l9a57gxxx/root/1");
        });
    });
});
