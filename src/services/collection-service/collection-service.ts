// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { CommonResourceService } from "../../common/api/common-resource-service";
import { CollectionResource } from "../../models/collection";
import axios, { AxiosInstance } from "axios";
import { KeepService } from "../keep-service/keep-service";
import { FilterBuilder } from "../../common/api/filter-builder";

export class CollectionService extends CommonResourceService<CollectionResource> {
    constructor(serverApi: AxiosInstance, private keepService: KeepService) {
        super(serverApi, "collections");
    }

    uploadFiles(files: File[]) {
        console.log("Uploading files", files);

        const fd = new FormData();
        fd.append("filters", `[["service_type","=","proxy"]]`);
        fd.append("_method", "GET");

        const filters = new FilterBuilder();
        filters.addEqual("service_type", "proxy");

        return this.keepService.list({ filters }).then(data => {
            console.log(data);

            const serviceHost = (data.items[0].serviceSslFlag ? "https://" : "http://") + data.items[0].serviceHost + ":" + data.items[0].servicePort;
            console.log("Servicehost", serviceHost);

            const fd = new FormData();
            files.forEach((f, idx) => fd.append(`file_${idx}`, f));

            axios.post(serviceHost, fd, {
                onUploadProgress: (e: ProgressEvent) => {
                    console.log(`${e.loaded} / ${e.total}`);
                }
            });
        });
    }
}
