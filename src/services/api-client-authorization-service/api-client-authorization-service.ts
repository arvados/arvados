// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { AxiosInstance } from "axios";
import { ApiActions } from 'services/api/api-actions';
import { ApiClientAuthorization } from 'models/api-client-authorization';
import { CommonService, ListResults } from 'services/common-service/common-service';
import { extractUuidObjectType, ResourceObjectType } from "models/resource";
import { FilterBuilder } from "services/api/filter-builder";

export class ApiClientAuthorizationService extends CommonService<ApiClientAuthorization> {
    constructor(serverApi: AxiosInstance, actions: ApiActions) {
        super(serverApi, "api_client_authorizations", actions);
    }

    createCollectionSharingToken(uuid: string): Promise<ApiClientAuthorization> {
        if (extractUuidObjectType(uuid) !== ResourceObjectType.COLLECTION) {
            throw new Error(`UUID ${uuid} is not a collection`);
        }
        return this.create({
            scopes: [
                `GET /arvados/v1/collections/${uuid}`,
                `GET /arvados/v1/collections/${uuid}/`,
                `GET /arvados/v1/keep_services/accessible`,
            ]
        });
    }

    listCollectionSharingTokens(uuid: string): Promise<ListResults<ApiClientAuthorization>> {
        if (extractUuidObjectType(uuid) !== ResourceObjectType.COLLECTION) {
            throw new Error(`UUID ${uuid} is not a collection`);
        }
        return this.list({
            filters: new FilterBuilder()
                .addEqual("scopes", [
                    `GET /arvados/v1/collections/${uuid}`,
                    `GET /arvados/v1/collections/${uuid}/`,
                    "GET /arvados/v1/keep_services/accessible"
                ]).getFilters()
        });
    }
}