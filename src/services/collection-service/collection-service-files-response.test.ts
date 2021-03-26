// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { CollectionFile } from '~/models/collection-file';
import { getFileFullPath, extractFilesData } from './collection-service-files-response';

describe('collection-service-files-response', () => {

    describe('extractFilesData', () => {
        it('should extract', () => {
            // given
            const xmlString = '<D:multistatus xmlns:D="DAV:"><D:response><D:href>/c=xxxxx-zzzzz-vvvvvvvvvvvvvvv/</D:href><D:propstat><D:prop><D:resourcetype><D:collection xmlns:D="DAV:"/></D:resourcetype><D:getlastmodified>Wed, 24 Feb 2021 22:16:19 GMT</D:getlastmodified><D:supportedlock><D:lockentry xmlns:D="DAV:"><D:lockscope><D:exclusive/></D:lockscope><D:locktype><D:write/></D:locktype></D:lockentry></D:supportedlock><D:displayname></D:displayname></D:prop><D:status>HTTP/1.1 200 OK</D:status></D:propstat></D:response><D:response><D:href>/c=zzzzz-xxxxx-vvvvvvvvvvvvvvv/2</D:href><D:propstat><D:prop><D:getcontentlength>1582976</D:getcontentlength><D:getetag>"1666cee048aa7f98182780"</D:getetag><D:resourcetype></D:resourcetype><D:displayname>2</D:displayname><D:getlastmodified>Wed, 24 Feb 2021 22:16:19 GMT</D:getlastmodified><D:getcontenttype>text/plain; charset=utf-8</D:getcontenttype><D:supportedlock><D:lockentry xmlns:D="DAV:"><D:lockscope><D:exclusive/></D:lockscope><D:locktype><D:write/></D:locktype></D:lockentry></D:supportedlock></D:prop><D:status>HTTP/1.1 200 OK</D:status></D:propstat></D:response><D:response><D:href>/c=zzzzz-xxxxx-vvvvvvvvvvvvvvv/table%201%202%203</D:href><D:propstat><D:prop><D:resourcetype></D:resourcetype><D:getcontentlength>133352</D:getcontentlength><D:getetag>"1666cee048aa7f98208e8"</D:getetag><D:displayname>table 1 2 3</D:displayname><D:getlastmodified>Wed, 24 Feb 2021 22:16:19 GMT</D:getlastmodified><D:getcontenttype>text/plain; charset=utf-8</D:getcontenttype><D:supportedlock><D:lockentry xmlns:D="DAV:"><D:lockscope><D:exclusive/></D:lockscope><D:locktype><D:write/></D:locktype></D:lockentry></D:supportedlock></D:prop><D:status>HTTP/1.1 200 OK</D:status></D:propstat></D:response></D:multistatus>';
            const parser = new DOMParser();
            const xmlDoc = parser.parseFromString(xmlString, "text/xml");

            // when
            const result = extractFilesData(xmlDoc);

            // then
            expect(result).toEqual([{ id: "zzzzz-xxxxx-vvvvvvvvvvvvvvv/2", name: "2", path: "", size: 1582976, type: "file", url: "/c=zzzzz-xxxxx-vvvvvvvvvvvvvvv/2" }, { id: "zzzzz-xxxxx-vvvvvvvvvvvvvvv/table 1 2 3", name: "table 1 2 3", path: "", size: 133352, type: "file", url: "/c=zzzzz-xxxxx-vvvvvvvvvvvvvvv/table 1 2 3" }]);
        });

        it('should extract ecoded data and do not encode already encoded props', () => {
            // given
            const xmlString = '<?xml version="1.0" encoding="UTF-8"?><D:multistatus xmlns:D="DAV:"><D:response><D:href>/c=zzzzz-xxxxx-vvvvvvvvvvvvvvv/</D:href><D:propstat><D:prop><D:resourcetype><D:collection xmlns:D="DAV:"/></D:resourcetype><D:getlastmodified>Fri, 26 Mar 2021 11:45:50 GMT</D:getlastmodified><D:supportedlock><D:lockentry xmlns:D="DAV:"><D:lockscope><D:exclusive/></D:lockscope><D:locktype><D:write/></D:locktype></D:lockentry></D:supportedlock><D:displayname></D:displayname></D:prop><D:status>HTTP/1.1 200 OK</D:status></D:propstat></D:response><D:response><D:href>/c=zzzzz-xxxxx-vvvvvvvvvvvvvvv/table%25&amp;%3F%2A2</D:href><D:propstat><D:prop><D:resourcetype></D:resourcetype><D:getcontentlength>3</D:getcontentlength><D:getlastmodified>Fri, 26 Mar 2021 11:45:50 GMT</D:getlastmodified><D:getetag>"166fe1e1a403fb683"</D:getetag><D:getcontenttype>text/plain; charset=utf-8</D:getcontenttype><D:supportedlock><D:lockentry xmlns:D="DAV:"><D:lockscope><D:exclusive/></D:lockscope><D:locktype><D:write/></D:locktype></D:lockentry></D:supportedlock><D:displayname>table%&amp;?*2</D:displayname></D:prop><D:status>HTTP/1.1 200 OK</D:status></D:propstat></D:response></D:multistatus>';
            const parser = new DOMParser();
            const xmlDoc = parser.parseFromString(xmlString, "text/xml");

            // when
            const result = extractFilesData(xmlDoc);

            // then
            expect(result).toEqual([{ id: "zzzzz-xxxxx-vvvvvvvvvvvvvvv/table%&?*2", name: "table%&?*2", path: "", size: 3, type: "file", url: "/c=zzzzz-xxxxx-vvvvvvvvvvvvvvv/table%&?*2" }]);
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