// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as _ from "lodash";
import { AxiosInstance } from "axios";
import { Resource } from "src/models/resource";
import { ApiActions } from "~/services/api/api-actions";
import { CommonService } from "~/services/common-service/common-service";

export enum CommonResourceServiceError {
    UNIQUE_VIOLATION = 'UniqueViolation',
    OWNERSHIP_CYCLE = 'OwnershipCycle',
    MODIFYING_CONTAINER_REQUEST_FINAL_STATE = 'ModifyingContainerRequestFinalState',
    NAME_HAS_ALREADY_BEEN_TAKEN = 'NameHasAlreadyBeenTaken',
    UNKNOWN = 'Unknown',
    NONE = 'None'
}

export class CommonResourceService<T extends Resource> extends CommonService<T> {

    constructor(serverApi: AxiosInstance, resourceType: string, actions: ApiActions) {
        super(serverApi, resourceType, actions);
    }
    
}

export const getCommonResourceServiceError = (errorResponse: any) => {
    if ('errors' in errorResponse && 'errorToken' in errorResponse) {
        const error = errorResponse.errors.join('');
        switch (true) {
            case /UniqueViolation/.test(error):
                return CommonResourceServiceError.UNIQUE_VIOLATION;
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


