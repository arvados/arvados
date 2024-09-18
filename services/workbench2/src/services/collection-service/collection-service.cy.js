// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import axios from 'axios';
import { snakeCase } from 'lodash';
import { defaultCollectionSelectedFields } from 'models/collection';
import { CollectionService, emptyCollectionPdh } from './collection-service';

describe('collection-service', () => {
    let collectionService = {};
    let serverApi;
    let keepWebdavClient;
    let authService;
    let actions;

    beforeEach(() => {
        serverApi = axios.create();
        keepWebdavClient = {
            delete: cy.stub(),
            upload: cy.stub().as('upload'),
            mkdir: cy.stub(),
        };
        authService = {};
        actions = {
            progressFn: cy.stub(),
            errorFn: cy.stub(),
        };
        collectionService = new CollectionService(serverApi, keepWebdavClient, authService, actions);
        collectionService.update = cy.stub();
    });

    describe('get', () => {
        it('should make a request with default selected fields', async () => {
            serverApi.get = cy.stub().returns(Promise.resolve(
                { data: { items: [{}] } }
            )).as('get');
            const uuid = 'zzzzz-4zz18-0123456789abcde'
            await collectionService.get(uuid);
            cy.get('@get').should('be.calledWith', `/collections/${uuid}`, {
                params: {
                    select: JSON.stringify(defaultCollectionSelectedFields.map(snakeCase)),
                }
            }); 
        });

        it('should be able to request specific fields', async () => {
            serverApi.get = cy.stub().returns(Promise.resolve(
                { data: { items: [{}] } }
            )).as('get');
            const uuid = 'zzzzz-4zz18-0123456789abcde'
            await collectionService.get(uuid, undefined, ['manifestText']);
            cy.get('@get').should('be.calledWith', `/collections/${uuid}`, {
                params: {
                    select: JSON.stringify(['manifest_text']),
                }
            });
        });
    });

    describe('update', () => {
        it('should call put selecting updated fields + others', async () => {
            serverApi.put = cy.stub().returns(Promise.resolve({ data: {} })).as('put');
            const data = {
                name: 'foo',
            };
            const expected = {
                collection: {
                    ...data,
                    preserve_version: true,
                },
                select: ['uuid', 'name', 'version', 'modified_at'],
            }
            collectionService = new CollectionService(serverApi, keepWebdavClient, authService, actions);
            await collectionService.update('uuid', data);
            cy.get('@put').should('be.calledWith', '/collections/uuid', expected);
        });
    });

    describe('uploadFiles', () => {
        it('should skip if no files to upload files', async () => {
            // given
            const files = [];
            const collectionUUID = '';

            // when
            await collectionService.uploadFiles(collectionUUID, files);

            // then
            cy.get('@upload').should('not.have.been.called');
        });

        it('should upload files', async () => {
            // given
            const files = [{name: 'test-file1'}];
            const collectionUUID = 'zzzzz-4zz18-0123456789abcde';

            // when
            await collectionService.uploadFiles(collectionUUID, files);

            // then
            cy.get('@upload').should('have.been.calledOnce');
            cy.get('@upload').should('have.been.calledWith', "c=zzzzz-4zz18-0123456789abcde/test-file1");
        });

        it('should upload files with custom uplaod target', async () => {
            // given
            const files = [{name: 'test-file1'}];
            const collectionUUID = 'zzzzz-4zz18-0123456789abcde';
            const customTarget = 'zzzzz-4zz18-0123456789adddd/test-path/'

            // when
            await collectionService.uploadFiles(collectionUUID, files, undefined, customTarget);

            // then
            cy.get('@upload').should('have.been.calledOnce');
            cy.get('@upload').should('have.been.calledWith', "c=zzzzz-4zz18-0123456789adddd/test-path/test-file1");
        });
    });

    describe('deleteFiles', () => {
        it('should remove no files', async () => {
            // given
            serverApi.put = cy.stub().returns(Promise.resolve({ data: {} })).as('put');
            const filePaths = [];
            const collectionUUID = 'zzzzz-tpzed-5o5tg0l9a57gxxx';

            // when
            await collectionService.deleteFiles(collectionUUID, filePaths);

            // then
            cy.get('@put').should('have.been.calledOnce');
            cy.get('@put').should('have.been.calledWith', `/collections/${collectionUUID}`, {
                collection: {
                    preserve_version: true
                },
                replace_files: {},
            });
        });

        it('should remove only root files', async () => {
            // given
            serverApi.put = cy.stub().returns(Promise.resolve({ data: {} })).as('put');
            const filePaths = ['/root/1', '/root/1/100', '/root/1/100/test.txt', '/root/2', '/root/2/200', '/root/3/300/test.txt'];
            const collectionUUID = 'zzzzz-tpzed-5o5tg0l9a57gxxx';

            // when
            await collectionService.deleteFiles(collectionUUID, filePaths);

            // then
            cy.get('@put').should('have.been.calledOnce');
            cy.get('@put').should('have.been.calledWith', `/collections/${collectionUUID}`, {
                collection: {
                    preserve_version: true
                },
                replace_files: {
                    '/root/1': '',
                    '/root/2': '',
                    '/root/3/300/test.txt': '',
                },
            });
        });

        it('should batch remove files', async () => {
            serverApi.put = cy.stub().returns(Promise.resolve({ data: {} })).as('put');
            // given
            const filePaths = ['/root/1', '/secondFile', 'barefile.txt'];
            const collectionUUID = 'zzzzz-4zz18-5o5tg0l9a57gxxx';

            // when
            await collectionService.deleteFiles(collectionUUID, filePaths);

            // then
            cy.get('@put').should('have.been.calledOnce');
            cy.get('@put').should('have.been.calledWith', `/collections/${collectionUUID}`, {
                collection: {
                    preserve_version: true
                },
                replace_files: {
                    '/root/1': '',
                    '/secondFile': '',
                    '/barefile.txt': '',
                },
            });
        });
    });

    describe('renameFile', () => {
        it('should rename file', async () => {
            serverApi.put = cy.stub().returns(Promise.resolve({ data: {} })).as('put');
            const collectionUuid = 'zzzzz-4zz18-ywq0rvhwwhkjnfq';
            const collectionPdh = '8cd9ce1dfa21c635b620b1bfee7aaa08+180';
            const oldPath = '/old/path';
            const newPath = '/new/filename';

            await collectionService.renameFile(collectionUuid, collectionPdh, oldPath, newPath);

            cy.get('@put').should('have.been.calledOnce');
            cy.get('@put').should('have.been.calledWith', `/collections/${collectionUuid}`, {
                collection: {
                    preserve_version: true
                },
                replace_files: {
                    [newPath]: `${collectionPdh}${oldPath}`,
                    [oldPath]: '',
                },
            });
        });
    });

    describe('copyFiles', () => {
        it('should batch copy files', async () => {
            serverApi.put = cy.stub().returns(Promise.resolve({ data: {} })).as('put');
            const filePaths = ['/root/1', '/secondFile', 'barefile.txt'];
            const sourcePdh = '8cd9ce1dfa21c635b620b1bfee7aaa08+180';

            const destinationUuid = 'zzzzz-4zz18-ywq0rvhwwhkjnfq';
            const destinationPath = '/destinationPath';

            // when
            await collectionService.copyFiles(sourcePdh, filePaths, {uuid: destinationUuid}, destinationPath);

            // then
            cy.get('@put').should('have.been.calledOnce');
            cy.get('@put').should('have.been.calledWith', `/collections/${destinationUuid}`, {
                collection: {
                    preserve_version: true
                },
                replace_files: {
                    [`${destinationPath}/1`]: `${sourcePdh}/root/1`,
                    [`${destinationPath}/secondFile`]: `${sourcePdh}/secondFile`,
                    [`${destinationPath}/barefile.txt`]: `${sourcePdh}/barefile.txt`,
                },
            });            
        });

        it('should copy files from rooth', async () => {
            // Test copying from root paths
            serverApi.put = cy.stub().returns(Promise.resolve({ data: {} })).as('put');
            const filePaths = ['/'];
            const sourcePdh = '8cd9ce1dfa21c635b620b1bfee7aaa08+180';

            const destinationUuid = 'zzzzz-4zz18-ywq0rvhwwhkjnfq';
            const destinationPath = '/destinationPath';

            await collectionService.copyFiles(sourcePdh, filePaths, {uuid: destinationUuid}, destinationPath);

            cy.get('@put').should('have.been.calledOnce');
            cy.get('@put').should('have.been.calledWith', `/collections/${destinationUuid}`, {
                collection: {
                    preserve_version: true
                },
                replace_files: {
                    [`${destinationPath}`]: `${sourcePdh}/`,
                },
            });
        });

        it('should copy files to root path', async () => {
            // Test copying to root paths
            serverApi.put = cy.stub().returns(Promise.resolve({ data: {} })).as('put');
            const filePaths = ['/'];
            const sourcePdh = '8cd9ce1dfa21c635b620b1bfee7aaa08+180';

            const destinationUuid = 'zzzzz-4zz18-ywq0rvhwwhkjnfq';
            const destinationPath = '/';

            await collectionService.copyFiles(sourcePdh, filePaths, {uuid: destinationUuid}, destinationPath);

            cy.get('@put').should('have.been.calledOnce');
            cy.get('@put').should('have.been.calledWith', `/collections/${destinationUuid}`, {
                collection: {
                    preserve_version: true
                },
                replace_files: {
                    "/": `${sourcePdh}/`,
                },
            });
        });
    });

    describe('moveFiles', () => {
        it('should batch move files', async () => {
            serverApi.put = cy.stub().returns(Promise.resolve({ data: {} })).as('put');
            // given
            const filePaths = ['/rootFile', '/secondFile', '/subpath/subfile', 'barefile.txt'];
            const srcCollectionUUID = 'zzzzz-4zz18-5o5tg0l9a57gxxx';
            const srcCollectionPdh = '8cd9ce1dfa21c635b620b1bfee7aaa08+180';

            const destinationUuid = 'zzzzz-4zz18-ywq0rvhwwhkjnfq';
            const destinationPath = '/destinationPath';

            // when
            await collectionService.moveFiles(srcCollectionUUID, srcCollectionPdh, filePaths, {uuid: destinationUuid}, destinationPath);

            // then
            cy.get('@put').should('have.been.calledTwice');
            // Verify copy
            cy.get('@put').should('have.been.calledWith', `/collections/${destinationUuid}`, {
                collection: {
                    preserve_version: true
                },
                replace_files: {
                    [`${destinationPath}/rootFile`]: `${srcCollectionPdh}/rootFile`,
                    [`${destinationPath}/secondFile`]: `${srcCollectionPdh}/secondFile`,
                    [`${destinationPath}/subfile`]: `${srcCollectionPdh}/subpath/subfile`,
                    [`${destinationPath}/barefile.txt`]: `${srcCollectionPdh}/barefile.txt`,
                },
            });
            // Verify delete
            cy.get('@put').should('have.been.calledWith', `/collections/${srcCollectionUUID}`, {
                collection: {
                    preserve_version: true
                },
                replace_files: {
                    '/rootFile': '',
                    '/secondFile': '',
                    '/subpath/subfile': '',
                    '/barefile.txt': '',
                },
            });
        });

        it('should batch move files within collection', async () => {
            serverApi.put = cy.stub().returns(Promise.resolve({ data: {} })).as('put');
            // given
            const filePaths = ['/one', '/two', '/subpath/subfile', 'barefile.txt'];
            const srcCollectionUUID = 'zzzzz-4zz18-5o5tg0l9a57gxxx';
            const srcCollectionPdh = '8cd9ce1dfa21c635b620b1bfee7aaa08+180';

            const destinationPath = '/destinationPath';

            // when
            await collectionService.moveFiles(srcCollectionUUID, srcCollectionPdh, filePaths, {uuid: srcCollectionUUID}, destinationPath);

            // then
            cy.get('@put').should('have.been.calledOnce');
            // Verify copy
            cy.get('@put').should('have.been.calledWith', `/collections/${srcCollectionUUID}`, {
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
            });
        });

        it('should abort batch move when copy fails', async () => {
            // Simulate failure to copy
            // rejection error will show up in console, but it's expected
            serverApi.put = cy.stub().returns(Promise.reject({
                data: {},
                response: {
                    "errors": ["error getting snapshot of \"rootFile\" from \"8cd9ce1dfa21c635b620b1bfee7aaa08+180\": file does not exist"]
                }
            })).as('put');
            // given
            const filePaths = ['/rootFile', '/secondFile', '/subpath/subfile', 'barefile.txt'];
            const srcCollectionUUID = 'zzzzz-4zz18-5o5tg0l9a57gxxx';
            const srcCollectionPdh = '8cd9ce1dfa21c635b620b1bfee7aaa08+180';

            const destinationUuid = 'zzzzz-4zz18-ywq0rvhwwhkjnfq';
            const destinationPath = '/destinationPath';

            // when
            try {
                await collectionService.moveFiles(srcCollectionUUID, srcCollectionPdh, filePaths, {uuid: destinationUuid}, destinationPath);
            } catch {}

            // then
            cy.get('@put').should('have.been.calledOnce');
            // Verify copy
            cy.get('@put').should('have.been.calledWith', `/collections/${destinationUuid}`, {
                collection: {
                    preserve_version: true
                },
                replace_files: {
                    [`${destinationPath}/rootFile`]: `${srcCollectionPdh}/rootFile`,
                    [`${destinationPath}/secondFile`]: `${srcCollectionPdh}/secondFile`,
                    [`${destinationPath}/subfile`]: `${srcCollectionPdh}/subpath/subfile`,
                    [`${destinationPath}/barefile.txt`]: `${srcCollectionPdh}/barefile.txt`,
                },
            });
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
                serverApi.put = cy.stub().returns(Promise.resolve({ data: {} })).as('put');
                // when
                await collectionService.createDirectory(collectionUuid, directoryNames[i].in);
                // then
                cy.get('@put').should('have.been.calledOnce');
                cy.get('@put').should('have.been.calledWith', `/collections/${collectionUuid}`, {
                    collection: {
                        preserve_version: true
                    },
                    replace_files: {
                        ["/" + directoryNames[i].out]: emptyCollectionPdh,
                    },
                });
            }
        });
    });

});
