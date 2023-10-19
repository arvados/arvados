// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { LogService } from "./log-service";
import { ApiActions } from "services/api/api-actions";
import axios from "axios";
import { WebDAVRequestConfig } from "common/webdav";
import { LogEventType } from "models/log";

describe("LogService", () => {

    let apiWebdavClient: any;
    const axiosInstance = axios.create();
    const actions: ApiActions = {
        progressFn: (id: string, working: boolean) => {},
        errorFn: (id: string, message: string) => {}
    };

    beforeEach(() => {
        apiWebdavClient = {
            delete: jest.fn(),
            upload: jest.fn(),
            mkdir: jest.fn(),
            get: jest.fn(),
            propfind: jest.fn(),
        } as any;
    });

    it("lists log files using propfind on live logs api endpoint", async () => {
        const logService = new LogService(axiosInstance, apiWebdavClient, actions);

        // given
        const containerRequest = {uuid: 'zzzzz-xvhdp-000000000000000', containerUuid: 'zzzzz-dz642-000000000000000'};
        const xmlData = `<?xml version="1.0" encoding="UTF-8"?>
            <D:multistatus xmlns:D="DAV:">
                    <D:response>
                            <D:href>/arvados/v1/container_requests/${containerRequest.uuid}/log/${containerRequest.containerUuid}/</D:href>
                            <D:propstat>
                                    <D:prop>
                                            <D:resourcetype>
                                                    <D:collection xmlns:D="DAV:" />
                                            </D:resourcetype>
                                            <D:getlastmodified>Tue, 15 Aug 2023 12:54:37 GMT</D:getlastmodified>
                                            <D:displayname></D:displayname>
                                            <D:supportedlock>
                                                    <D:lockentry xmlns:D="DAV:">
                                                            <D:lockscope>
                                                                    <D:exclusive />
                                                            </D:lockscope>
                                                            <D:locktype>
                                                                    <D:write />
                                                            </D:locktype>
                                                    </D:lockentry>
                                            </D:supportedlock>
                                    </D:prop>
                                    <D:status>HTTP/1.1 200 OK</D:status>
                            </D:propstat>
                    </D:response>
                    <D:response>
                            <D:href>/arvados/v1/container_requests/${containerRequest.uuid}/log/${containerRequest.containerUuid}/stdout.txt</D:href>
                            <D:propstat>
                                    <D:prop>
                                            <D:displayname>stdout.txt</D:displayname>
                                            <D:getcontentlength>15</D:getcontentlength>
                                            <D:getcontenttype>text/plain; charset=utf-8</D:getcontenttype>
                                            <D:getetag>"177b8fb161ff9f58f"</D:getetag>
                                            <D:supportedlock>
                                                    <D:lockentry xmlns:D="DAV:">
                                                            <D:lockscope>
                                                                    <D:exclusive />
                                                            </D:lockscope>
                                                            <D:locktype>
                                                                    <D:write />
                                                            </D:locktype>
                                                    </D:lockentry>
                                            </D:supportedlock>
                                            <D:resourcetype></D:resourcetype>
                                            <D:getlastmodified>Tue, 15 Aug 2023 12:54:37 GMT</D:getlastmodified>
                                    </D:prop>
                                    <D:status>HTTP/1.1 200 OK</D:status>
                            </D:propstat>
                    </D:response>
                    <D:response>
                            <D:href>/arvados/v1/container_requests/${containerRequest.uuid}/wrongpath.txt</D:href>
                            <D:propstat>
                                    <D:prop>
                                            <D:displayname>wrongpath.txt</D:displayname>
                                            <D:getcontentlength>15</D:getcontentlength>
                                            <D:getcontenttype>text/plain; charset=utf-8</D:getcontenttype>
                                            <D:getetag>"177b8fb161ff9f58f"</D:getetag>
                                            <D:supportedlock>
                                                    <D:lockentry xmlns:D="DAV:">
                                                            <D:lockscope>
                                                                    <D:exclusive />
                                                            </D:lockscope>
                                                            <D:locktype>
                                                                    <D:write />
                                                            </D:locktype>
                                                    </D:lockentry>
                                            </D:supportedlock>
                                            <D:resourcetype></D:resourcetype>
                                            <D:getlastmodified>Tue, 15 Aug 2023 12:54:37 GMT</D:getlastmodified>
                                    </D:prop>
                                    <D:status>HTTP/1.1 200 OK</D:status>
                            </D:propstat>
                    </D:response>
            </D:multistatus>`;
        const xmlDoc = (new DOMParser()).parseFromString(xmlData, "text/xml");
        apiWebdavClient.propfind = jest.fn().mockReturnValue(Promise.resolve({responseXML: xmlDoc}));

        // when
        const logs = await logService.listLogFiles(containerRequest);

        // then
        expect(apiWebdavClient.propfind).toHaveBeenCalledWith(`container_requests/${containerRequest.uuid}/log/${containerRequest.containerUuid}`);
        expect(logs.length).toEqual(1);
        expect(logs[0]).toHaveProperty('name', 'stdout.txt');
        expect(logs[0]).toHaveProperty('type', 'file');
    });

    it("requests log file contents with correct range request", async () => {
        const logService = new LogService(axiosInstance, apiWebdavClient, actions);

        // given
        const containerRequest = {uuid: 'zzzzz-xvhdp-000000000000000', containerUuid: 'zzzzz-dz642-000000000000000'};
        const fileRecord = {name: `stdout.txt`};
        const fileContents = `Line 1\nLine 2\nLine 3`;
        apiWebdavClient.get = jest.fn().mockImplementation((path: string, options: WebDAVRequestConfig) => {
            const matches = /bytes=([0-9]+)-([0-9]+)/.exec(options.headers?.Range || '');
            if (matches?.length === 3) {
                return Promise.resolve({responseText: fileContents.substring(Number(matches[1]), Number(matches[2]) + 1)})
            }
            return Promise.reject();
        });

        // when
        let result = await logService.getLogFileContents(containerRequest, fileRecord, 0, 3);
        // then
        expect(apiWebdavClient.get).toHaveBeenCalledWith(
            `container_requests/${containerRequest.uuid}/log/${containerRequest.containerUuid}/${fileRecord.name}`,
            {headers: {Range: `bytes=0-3`}}
        );
        expect(result.logType).toEqual(LogEventType.STDOUT);
        expect(result.contents).toEqual(['Line']);

        // when
        result = await logService.getLogFileContents(containerRequest, fileRecord, 0, 10);
        // then
        expect(apiWebdavClient.get).toHaveBeenCalledWith(
            `container_requests/${containerRequest.uuid}/log/${containerRequest.containerUuid}/${fileRecord.name}`,
            {headers: {Range: `bytes=0-10`}}
        );
        expect(result.logType).toEqual(LogEventType.STDOUT);
        expect(result.contents).toEqual(['Line 1', 'Line']);

        // when
        result = await logService.getLogFileContents(containerRequest, fileRecord, 6, 14);
        // then
        expect(apiWebdavClient.get).toHaveBeenCalledWith(
            `container_requests/${containerRequest.uuid}/log/${containerRequest.containerUuid}/${fileRecord.name}`,
            {headers: {Range: `bytes=6-14`}}
        );
        expect(result.logType).toEqual(LogEventType.STDOUT);
        expect(result.contents).toEqual(['', 'Line 2', 'L']);
    });

});
