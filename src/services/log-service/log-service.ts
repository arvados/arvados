// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { AxiosInstance } from "axios";
import { LogEventType, LogResource } from 'models/log';
import { CommonResourceService } from "services/common-service/common-resource-service";
import { ApiActions } from "services/api/api-actions";
import { WebDAV } from "common/webdav";
import { extractFilesData } from "services/collection-service/collection-service-files-response";
import { CollectionFile } from "models/collection-file";
import { ContainerRequestResource } from "models/container-request";

export type LogFragment = {
    logType: LogEventType;
    contents: string[];
}

export class LogService extends CommonResourceService<LogResource> {
    constructor(serverApi: AxiosInstance, private apiWebdavClient: WebDAV, actions: ApiActions) {
        super(serverApi, "logs", actions);
    }

    async listLogFiles(containerRequest: ContainerRequestResource) {
        const request = await this.apiWebdavClient.propfind(`container_requests/${containerRequest.uuid}/log/${containerRequest.containerUuid}`);
        if (request.responseXML != null) {
            return extractFilesData(request.responseXML)
                .filter((file) => (
                    file.path === `/arvados/v1/container_requests/${containerRequest.uuid}/log/${containerRequest.containerUuid}`
                ));
        }
        return Promise.reject();
    }

    async getLogFileContents(containerRequest: ContainerRequestResource, fileRecord: CollectionFile, startByte: number, endByte: number): Promise<LogFragment> {
        const request = await this.apiWebdavClient.get(
            `container_requests/${containerRequest.uuid}/log/${containerRequest.containerUuid}/${fileRecord.name}`,
            {headers: {Range: `bytes=${startByte}-${endByte}`}}
        );
        const logFileType = logFileToLogType(fileRecord);

        if (request.responseText && logFileType) {
            return {
                logType: logFileType,
                contents: request.responseText.split(/\r?\n/),
            };
        } else {
            return Promise.reject();
        }
    }
}

export const logFileToLogType = (file: CollectionFile) => (file.name.replace(/\.(txt|json)$/, '') as LogEventType);
