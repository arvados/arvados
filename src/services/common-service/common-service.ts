// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as _ from "lodash";
import { AxiosInstance, AxiosPromise } from "axios";
import * as uuid from "uuid/v4";
import { ApiActions } from "~/services/api/api-actions";

interface Errors {
    errors: string[];
    errorToken: string;
}

export interface ListArguments {
    limit?: number;
    offset?: number;
    filters?: string;
    order?: string;
    select?: string[];
    distinct?: boolean;
    count?: string;
}

export interface ListResults<T> {
    clusterId?: string;
    kind: string;
    offset: number;
    limit: number;
    items: T[];
    itemsAvailable: number;
}

export class CommonService<T> {
    protected serverApi: AxiosInstance;
    protected resourceType: string;
    protected actions: ApiActions;

    constructor(serverApi: AxiosInstance, resourceType: string, actions: ApiActions) {
        this.serverApi = serverApi;
        this.resourceType = '/' + resourceType + '/';
        this.actions = actions;
    }

    static mapResponseKeys = (response: { data: any }) =>
        CommonService.mapKeys(_.camelCase)(response.data)

    static mapKeys = (mapFn: (key: string) => string) =>
        (value: any): any => {
            switch (true) {
                case _.isPlainObject(value):
                    return Object
                        .keys(value)
                        .map(key => [key, mapFn(key)])
                        .reduce((newValue, [key, newKey]) => ({
                            ...newValue,
                            [newKey]: CommonService.mapKeys(mapFn)(value[key])
                        }), {});
                case _.isArray(value):
                    return value.map(CommonService.mapKeys(mapFn));
                default:
                    return value;
            }
        }

    static defaultResponse<R>(promise: AxiosPromise<R>, actions: ApiActions, mapKeys = true): Promise<R> {
        const reqId = uuid();
        actions.progressFn(reqId, true);
        return promise
            .then(data => {
                actions.progressFn(reqId, false);
                return data;
            })
            .then((response: { data: any }) => {
                return mapKeys ? CommonService.mapResponseKeys(response) : response.data;
            })
            .catch(({ response }) => {
                actions.progressFn(reqId, false);
                const errors = CommonService.mapResponseKeys(response) as Errors;
                actions.errorFn(reqId, errors);
                throw errors;
            });
    }

    create(data?: Partial<T>) {
        return CommonService.defaultResponse(
            this.serverApi
                .post<T>(this.resourceType, data && CommonService.mapKeys(_.snakeCase)(data)),
            this.actions
        );
    }

    delete(uuid: string): Promise<T> {
        return CommonService.defaultResponse(
            this.serverApi
                .delete(this.resourceType + uuid),
            this.actions
        );
    }

    get(uuid: string) {
        return CommonService.defaultResponse(
            this.serverApi
                .get<T>(this.resourceType + uuid),
            this.actions
        );
    }

    list(args: ListArguments = {}): Promise<ListResults<T>> {
        const { filters, order, ...other } = args;
        const params = {
            ...other,
            filters: filters ? `[${filters}]` : undefined,
            order: order ? order : undefined
        };
        return CommonService.defaultResponse(
            this.serverApi
                .get(this.resourceType, {
                    params: CommonService.mapKeys(_.snakeCase)(params)
                }),
            this.actions
        );
    }

    update(uuid: string, data: Partial<T>) {
        return CommonService.defaultResponse(
            this.serverApi
                .put<T>(this.resourceType + uuid, data && CommonService.mapKeys(_.snakeCase)(data)),
            this.actions
        );
    }
}
