// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { CollectionFile } from 'models/collection-file';
import { getFileFullPath, extractFilesData } from './collection-service-files-response';

describe('collection-service-files-response', () => {

    describe('extractFilesData', () => {
        it('should correctly decode URLs & file names', () => {
            const testCases = [
                // input URL, input display name, expected URL, expected name
                ['table%201%202%203', 'table 1 2 3', 'table%201%202%203', 'table 1 2 3'],
                ['table%25&amp;%3F%2A2', 'table%&amp;?*2', 'table%25&%3F%2A2', 'table%&?*2'],
                ["G%C3%BCnter%27s%20file.pdf", "Günter&#39;s file.pdf", "G%C3%BCnter%27s%20file.pdf", "Günter's file.pdf"],
                ['G%25C3%25BCnter%27s%2520file.pdf', 'G%C3%BCnter&#39;s%20file.pdf', "G%25C3%25BCnter%27s%2520file.pdf", "G%C3%BCnter's%20file.pdf"]
            ];

            testCases.forEach(([inputURL, inputDisplayName, expectedURL, expectedName]) => {
                // given
                const collUUID = 'xxxxx-zzzzz-vvvvvvvvvvvvvvv';
                const xmlData = `
                <?xml version="1.0" encoding="UTF-8"?>
                <D:multistatus xmlns:D="DAV:">
                    <D:response>
                        <D:href>/c=xxxxx-zzzzz-vvvvvvvvvvvvvvv/</D:href>
                        <D:propstat>
                            <D:prop>
                                <D:resourcetype>
                                    <D:collection xmlns:D="DAV:"/>
                                </D:resourcetype>
                                <D:supportedlock>
                                    <D:lockentry xmlns:D="DAV:">
                                        <D:lockscope>
                                            <D:exclusive/>
                                        </D:lockscope>
                                        <D:locktype>
                                            <D:write/>
                                        </D:locktype>
                                    </D:lockentry>
                                </D:supportedlock>
                                <D:displayname></D:displayname>
                                <D:getlastmodified>Fri, 26 Mar 2021 14:44:08 GMT</D:getlastmodified>
                            </D:prop>
                            <D:status>HTTP/1.1 200 OK</D:status>
                        </D:propstat>
                    </D:response>
                    <D:response>
                        <D:href>/c=${collUUID}/${inputURL}</D:href>
                        <D:propstat>
                            <D:prop>
                                <D:resourcetype></D:resourcetype>
                                <D:getcontenttype>application/pdf</D:getcontenttype>
                                <D:supportedlock>
                                    <D:lockentry xmlns:D="DAV:">
                                        <D:lockscope>
                                            <D:exclusive/>
                                        </D:lockscope>
                                        <D:locktype>
                                            <D:write/>
                                        </D:locktype>
                                    </D:lockentry>
                                </D:supportedlock>
                                <D:displayname>${inputDisplayName}</D:displayname>
                                <D:getcontentlength>3</D:getcontentlength>
                                <D:getlastmodified>Fri, 26 Mar 2021 14:44:08 GMT</D:getlastmodified>
                                <D:getetag>"166feb9c9110c008325a59"</D:getetag>
                            </D:prop>
                            <D:status>HTTP/1.1 200 OK</D:status>
                        </D:propstat>
                    </D:response>
                </D:multistatus>
                `;
                const parser = new DOMParser();
                const xmlDoc = parser.parseFromString(xmlData, "text/xml");

                // when
                const result = extractFilesData(xmlDoc);

                // then
                expect(result).toEqual([{ id: `${collUUID}/${expectedName}`, name: expectedName, path: "", size: 3, type: "file", url: `/c=${collUUID}/${expectedURL}` }]);
            });
        });
    });

    describe('getFileFullPath', () => {
        it('should encode weird names', async () => {
            // given
            const file = {
                name: '#test',
                path: 'http://localhost',
            } as CollectionFile;

            // when
            const result = getFileFullPath(file);

            // then
            expect(result).toBe('http://localhost/#test');
        });

    });
});