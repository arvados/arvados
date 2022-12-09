// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { AxiosInstance } from "axios";
import { snakeCase } from "lodash";
import { Resource } from "models/resource";
import { ApiActions } from "services/api/api-actions";
import { CommonService } from "services/common-service/common-service";

export enum CommonResourceServiceError {
    UNIQUE_NAME_VIOLATION = 'UniqueNameViolation',
    OWNERSHIP_CYCLE = 'OwnershipCycle',
    MODIFYING_CONTAINER_REQUEST_FINAL_STATE = 'ModifyingContainerRequestFinalState',
    NAME_HAS_ALREADY_BEEN_TAKEN = 'NameHasAlreadyBeenTaken',
    UNKNOWN = 'Unknown',
    NONE = 'None'
}

export class CommonResourceService<T extends Resource> extends CommonService<T> {
    constructor(serverApi: AxiosInstance, resourceType: string, actions: ApiActions, readOnlyFields: string[] = []) {
        super(serverApi, resourceType, actions, readOnlyFields.concat([
            'uuid',
            'etag',
            'kind'
        ]));
    }

    create(data?: Partial<T>) {
        let payload: any;
        if (data !== undefined) {
            this.readOnlyFields.forEach( field => delete data[field] );
            payload = {
                [this.resourceType.slice(0, -1)]: CommonService.mapKeys(snakeCase)(data),
            };
        }
        return super.create(payload);
    }

    update(uuid: string, data: Partial<T>, showErrors?: boolean, select?: string[]) {
        let payload: any;
        if (data !== undefined) {
            this.readOnlyFields.forEach( field => delete data[field] );
            payload = {
                [this.resourceType.slice(0, -1)]: CommonService.mapKeys(snakeCase)(data),
            };
            if (select !== undefined && select.length > 0) {
                payload.select = ['uuid', ...select.map(field => snakeCase(field))];
            };
        }
        return super.update(uuid, payload, showErrors);
    }
}

export const getCommonResourceServiceError = (errorResponse: any) => {
    if ('errors' in errorResponse) {
        const error = errorResponse.errors.join('');
        switch (true) {
            case /UniqueViolation/.test(error):
                return CommonResourceServiceError.UNIQUE_NAME_VIOLATION;
            case /ownership cycle/.test(error):
                return CommonResourceServiceError.OWNERSHIP_CYCLE;
            case /Mounts cannot be modified in state 'Final'/.test(error):
                return CommonResourceServiceError.MODIFYING_CONTAINER_REQUEST_FINAL_STATE;
            case /Name has already been taken/.test(error):
                return CommonResourceServiceError.NAME_HAS_ALREADY_BEEN_TAKEN;
            default:
                return CommonResourceServiceError.UNKNOWN;
        }
    }
    return CommonResourceServiceError.NONE;
};


