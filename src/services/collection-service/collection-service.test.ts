// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import axios, { AxiosInstance } from 'axios';
import MockAdapter from 'axios-mock-adapter';
import { CollectionResource } from 'models/collection';
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
        } as any;
        authService = {} as AuthService;
        actions = {
            progressFn: jest.fn(),
        } as any;
        collectionService = new CollectionService(serverApi, webdavClient, authService, actions);
        collectionService.update = jest.fn();
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