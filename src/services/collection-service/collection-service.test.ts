// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import axios, { AxiosInstance } from 'axios';
import MockAdapter from 'axios-mock-adapter';
import { snakeCase } from 'lodash';
import { CollectionResource, defaultCollectionSelectedFields } from 'models/collection';
import { AuthService } from '../auth-service/auth-service';
import { CollectionService, emptyCollectionUuid } from './collection-service';

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
            mkdir: jest.fn(),
        } as any;
        authService = {} as AuthService;
        actions = {
            progressFn: jest.fn(),
            errorFn: jest.fn(),
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
            serverApi.put = jest.fn(() => Promise.resolve({ data: {} }));
            const filePaths: string[] = [];
            const collectionUUID = 'zzzzz-tpzed-5o5tg0l9a57gxxx';

            // when
            await collectionService.deleteFiles(collectionUUID, filePaths);

            // then
            expect(serverApi.put).toHaveBeenCalledTimes(1);
            expect(serverApi.put).toHaveBeenCalledWith(
                `/collections/${collectionUUID}`, {
                    collection: {
                        preserve_version: true
                    },
                    replace_files: {},
                }
            );
        });

        it('should remove only root files', async () => {
            // given
            serverApi.put = jest.fn(() => Promise.resolve({ data: {} }));
            const filePaths: string[] = ['/root/1', '/root/1/100', '/root/1/100/test.txt', '/root/2', '/root/2/200', '/root/3/300/test.txt'];
            const collectionUUID = 'zzzzz-tpzed-5o5tg0l9a57gxxx';

            // when
            await collectionService.deleteFiles(collectionUUID, filePaths);

            // then
            expect(serverApi.put).toHaveBeenCalledTimes(1);
            expect(serverApi.put).toHaveBeenCalledWith(
                `/collections/${collectionUUID}`, {
                    collection: {
                        preserve_version: true
                    },
                    replace_files: {
                        '/root/3/300/test.txt': '',
                        '/root/2': '',
                        '/root/1': '',
                    },
                }
            );
        });

        it('should remove files with uuid prefix', async () => {
            // given
            serverApi.put = jest.fn(() => Promise.resolve({ data: {} }));
            const filePaths: string[] = ['/root/1'];
            const collectionUUID = 'zzzzz-tpzed-5o5tg0l9a57gxxx';

            // when
            await collectionService.deleteFiles(collectionUUID, filePaths);

            // then
            expect(serverApi.put).toHaveBeenCalledTimes(1);
            expect(serverApi.put).toHaveBeenCalledWith(
                `/collections/${collectionUUID}`, {
                    collection: {
                        preserve_version: true
                    },
                    replace_files: {
                        '/root/1': '',
                    },
                }
            );
        });

        it('should batch remove files', async () => {
            serverApi.put = jest.fn(() => Promise.resolve({ data: {} }));
            // given
            const filePaths: string[] = ['/root/1', '/secondFile', 'barefile.txt'];
            const collectionUUID = 'zzzzz-4zz18-5o5tg0l9a57gxxx';

            // when
            await collectionService.deleteFiles(collectionUUID, filePaths);

            // then
            expect(serverApi.put).toHaveBeenCalledTimes(1);
            expect(serverApi.put).toHaveBeenCalledWith(
                `/collections/${collectionUUID}`, {
                    collection: {
                        preserve_version: true
                    },
                    replace_files: {
                        '/root/1': '',
                        '/secondFile': '',
                        '/barefile.txt': '',
                    },
                }
            );
        });
    });

    describe('renameFile', () => {
        it('should rename file', async () => {
            serverApi.put = jest.fn(() => Promise.resolve({ data: {} }));
            const collectionUuid = 'zzzzz-4zz18-ywq0rvhwwhkjnfq';
            const collectionPdh = '8cd9ce1dfa21c635b620b1bfee7aaa08+180';
            const oldPath = '/old/path';
            const newPath = '/new/filename';

            await collectionService.renameFile(collectionUuid, collectionPdh, oldPath, newPath);

            expect(serverApi.put).toHaveBeenCalledTimes(1);
            expect(serverApi.put).toHaveBeenCalledWith(
                `/collections/${collectionUuid}`, {
                    collection: {
                        preserve_version: true
                    },
                    replace_files: {
                        [newPath]: `${collectionPdh}${oldPath}`,
                        [oldPath]: '',
                    },
                }
            );
        });
    });

    describe('copyFiles', () => {
        it('should batch copy files', async () => {
            serverApi.put = jest.fn(() => Promise.resolve({ data: {} }));
            const filePaths: string[] = ['/root/1', '/secondFile', 'barefile.txt'];
            const sourcePdh = '8cd9ce1dfa21c635b620b1bfee7aaa08+180';

            const destinationUuid = 'zzzzz-4zz18-ywq0rvhwwhkjnfq';
            const destinationPath = '/destinationPath';

            // when
            await collectionService.copyFiles(sourcePdh, filePaths, destinationUuid, destinationPath);

            // then
            expect(serverApi.put).toHaveBeenCalledTimes(1);
            expect(serverApi.put).toHaveBeenCalledWith(
                `/collections/${destinationUuid}`, {
                    collection: {
                        preserve_version: true
                    },
                    replace_files: {
                        [`${destinationPath}/1`]: `${sourcePdh}/root/1`,
                        [`${destinationPath}/secondFile`]: `${sourcePdh}/secondFile`,
                        [`${destinationPath}/barefile.txt`]: `${sourcePdh}/barefile.txt`,
                    },
                }
            );
        });

        it('should copy files from rooth', async () => {
            // Test copying from root paths
            serverApi.put = jest.fn(() => Promise.resolve({ data: {} }));
            const filePaths: string[] = ['/'];
            const sourcePdh = '8cd9ce1dfa21c635b620b1bfee7aaa08+180';

            const destinationUuid = 'zzzzz-4zz18-ywq0rvhwwhkjnfq';
            const destinationPath = '/destinationPath';

            await collectionService.copyFiles(sourcePdh, filePaths, destinationUuid, destinationPath);

            expect(serverApi.put).toHaveBeenCalledTimes(1);
            expect(serverApi.put).toHaveBeenCalledWith(
                `/collections/${destinationUuid}`, {
                    collection: {
                        preserve_version: true
                    },
                    replace_files: {
                        [`${destinationPath}`]: `${sourcePdh}/`,
                    },
                }
            );
        });

        it('should copy files to root path', async () => {
            // Test copying to root paths
            serverApi.put = jest.fn(() => Promise.resolve({ data: {} }));
            const filePaths: string[] = ['/'];
            const sourcePdh = '8cd9ce1dfa21c635b620b1bfee7aaa08+180';

            const destinationUuid = 'zzzzz-4zz18-ywq0rvhwwhkjnfq';
            const destinationPath = '/';

            await collectionService.copyFiles(sourcePdh, filePaths, destinationUuid, destinationPath);

            expect(serverApi.put).toHaveBeenCalledTimes(1);
            expect(serverApi.put).toHaveBeenCalledWith(
                `/collections/${destinationUuid}`, {
                    collection: {
                        preserve_version: true
                    },
                    replace_files: {
                        "/": `${sourcePdh}/`,
                    },
                }
            );
        });
    });

    describe('moveFiles', () => {
        it('should batch move files', async () => {
            serverApi.put = jest.fn(() => Promise.resolve({ data: {} }));
            // given
            const filePaths: string[] = ['/rootFile', '/secondFile', '/subpath/subfile', 'barefile.txt'];
            const srcCollectionUUID = 'zzzzz-4zz18-5o5tg0l9a57gxxx';
            const srcCollectionPdh = '8cd9ce1dfa21c635b620b1bfee7aaa08+180';

            const destinationUuid = 'zzzzz-4zz18-ywq0rvhwwhkjnfq';
            const destinationPath = '/destinationPath';

            // when
            await collectionService.moveFiles(srcCollectionUUID, srcCollectionPdh, filePaths, destinationUuid, destinationPath);

            // then
            expect(serverApi.put).toHaveBeenCalledTimes(2);
            // Verify copy
            expect(serverApi.put).toHaveBeenCalledWith(
                `/collections/${destinationUuid}`, {
                    collection: {
                        preserve_version: true
                    },
                    replace_files: {
                        [`${destinationPath}/rootFile`]: `${srcCollectionPdh}/rootFile`,
                        [`${destinationPath}/secondFile`]: `${srcCollectionPdh}/secondFile`,
                        [`${destinationPath}/subfile`]: `${srcCollectionPdh}/subpath/subfile`,
                        [`${destinationPath}/barefile.txt`]: `${srcCollectionPdh}/barefile.txt`,
                    },
                }
            );
            // Verify delete
            expect(serverApi.put).toHaveBeenCalledWith(
                `/collections/${srcCollectionUUID}`, {
                    collection: {
                        preserve_version: true
                    },
                    replace_files: {
                        "/rootFile": "",
                        "/secondFile": "",
                        "/subpath/subfile": "",
                        "/barefile.txt": "",
                    },
                }
            );
        });

        it('should batch move files within collection', async () => {
            serverApi.put = jest.fn(() => Promise.resolve({ data: {} }));
            // given
            const filePaths: string[] = ['/one', '/two', '/subpath/subfile', 'barefile.txt'];
            const srcCollectionUUID = 'zzzzz-4zz18-5o5tg0l9a57gxxx';
            const srcCollectionPdh = '8cd9ce1dfa21c635b620b1bfee7aaa08+180';

            const destinationPath = '/destinationPath';

            // when
            await collectionService.moveFiles(srcCollectionUUID, srcCollectionPdh, filePaths, srcCollectionUUID, destinationPath);

            // then
            expect(serverApi.put).toHaveBeenCalledTimes(1);
            // Verify copy
            expect(serverApi.put).toHaveBeenCalledWith(
                `/collections/${srcCollectionUUID}`, {
                    collection: {
                        preserve_version: true
                    },
                    replace_files: {
                        [`${destinationPath}/one`]: `${srcCollectionPdh}/one`,
                        ['/one']: '',
                        [`${destinationPath}/two`]: `${srcCollectionPdh}/two`,
                        ['/two']: '',
                        [`${destinationPath}/subfile`]: `${srcCollectionPdh}/subpath/subfile`,
                        ['/subpath/subfile']: '',
                        [`${destinationPath}/barefile.txt`]: `${srcCollectionPdh}/barefile.txt`,
                        ['/barefile.txt']: '',
                    },
                }
            );
        });

        it('should abort batch move when copy fails', async () => {
            // Simulate failure to copy
            serverApi.put = jest.fn(() => Promise.reject({
                data: {},
                response: {
                    "errors": ["error getting snapshot of \"rootFile\" from \"8cd9ce1dfa21c635b620b1bfee7aaa08+180\": file does not exist"]
                }
            }));
            // given
            const filePaths: string[] = ['/rootFile', '/secondFile', '/subpath/subfile', 'barefile.txt'];
            const srcCollectionUUID = 'zzzzz-4zz18-5o5tg0l9a57gxxx';
            const srcCollectionPdh = '8cd9ce1dfa21c635b620b1bfee7aaa08+180';

            const destinationUuid = 'zzzzz-4zz18-ywq0rvhwwhkjnfq';
            const destinationPath = '/destinationPath';

            // when
            try {
                await collectionService.moveFiles(srcCollectionUUID, srcCollectionPdh, filePaths, destinationUuid, destinationPath);
            } catch {}

            // then
            expect(serverApi.put).toHaveBeenCalledTimes(1);
            // Verify copy
            expect(serverApi.put).toHaveBeenCalledWith(
                `/collections/${destinationUuid}`, {
                    collection: {
                        preserve_version: true
                    },
                    replace_files: {
                        [`${destinationPath}/rootFile`]: `${srcCollectionPdh}/rootFile`,
                        [`${destinationPath}/secondFile`]: `${srcCollectionPdh}/secondFile`,
                        [`${destinationPath}/subfile`]: `${srcCollectionPdh}/subpath/subfile`,
                        [`${destinationPath}/barefile.txt`]: `${srcCollectionPdh}/barefile.txt`,
                    },
                }
            );
        });
    });

    describe('createDirectory', () => {
        it('creates empty directory', async () => {
            // given
            const directoryNames = [
                {in: 'newDir', out: 'newDir'},
                {in: '/fooDir', out: 'fooDir'},
                {in: '/anotherPath/', out: 'anotherPath'},
                {in: 'trailingSlash/', out: 'trailingSlash'},
            ];
            const collectionUuid = 'zzzzz-tpzed-5o5tg0l9a57gxxx';

            for (var i = 0; i < directoryNames.length; i++) {
                serverApi.put = jest.fn(() => Promise.resolve({ data: {} }));
                // when
                await collectionService.createDirectory(collectionUuid, directoryNames[i].in);
                // then
                expect(serverApi.put).toHaveBeenCalledTimes(1);
                expect(serverApi.put).toHaveBeenCalledWith(
                    `/collections/${collectionUuid}`, {
                        collection: {
                            preserve_version: true
                        },
                        replace_files: {
                            ["/" + directoryNames[i].out]: emptyCollectionUuid,
                        },
                    }
                );
            }
        });
    });

});
