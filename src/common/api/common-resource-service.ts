// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as _ from "lodash";
import { FilterBuilder } from "./filter-builder";
import { OrderBuilder } from "./order-builder";
import { AxiosInstance } from "axios";
import { Resource } from "../../models/resource";

export interface ListArguments {
    limit?: number;
    offset?: number;
    filters?: FilterBuilder;
    order?: OrderBuilder;
    select?: string[];
    distinct?: boolean;
    count?: string;
}

export interface ListResults<T> {
    kind: string;
    offset: number;
    limit: number;
    items: T[];
    itemsAvailable: number;
}

export class CommonResourceService<T extends Resource> {

    static mapResponseKeys = (response: any): Promise<any> =>
        CommonResourceService.mapKeys(_.camelCase)(response.data)

    static mapKeys = (mapFn: (key: string) => string) =>
        (value: any): any => {
            switch (true) {
                case _.isPlainObject(value):
                    return Object
                        .keys(value)
                        .map(key => [key, mapFn(key)])
                        .reduce((newValue, [key, newKey]) => ({
                            ...newValue,
                            [newKey]: CommonResourceService.mapKeys(mapFn)(value[key])
                        }), {});
                case _.isArray(value):
                    return value.map(CommonResourceService.mapKeys(mapFn));
                default:
                    return value;
            }
        }

    protected serverApi: AxiosInstance;
    protected resourceType: string;

    constructor(serverApi: AxiosInstance, resourceType: string) {
        this.serverApi = serverApi;
        this.resourceType = '/' + resourceType + '/';
    }

    create(data: Partial<T>) {
        return this.serverApi
            .post<T>(this.resourceType, CommonResourceService.mapKeys(_.snakeCase)(data))
            .then(CommonResourceService.mapResponseKeys);
    }

    delete(uuid: string): Promise<T> {
        return this.serverApi
            .delete(this.resourceType + uuid)
            .then(CommonResourceService.mapResponseKeys);
    }

    get(uuid: string) {
        return this.serverApi
            .get<T>(this.resourceType + uuid)
            .then(CommonResourceService.mapResponseKeys);
    }

    list(args: ListArguments = {}): Promise<ListResults<T>> {
        const { filters, order, ...other } = args;
        const params = {
            ...other,
            filters: filters ? filters.serialize() : undefined,
            order: order ? order.getOrder() : undefined
        };
        return this.serverApi
            .get(this.resourceType, {
                params: CommonResourceService.mapKeys(_.snakeCase)(params)
            })
            .then(CommonResourceService.mapResponseKeys);
    }

    update(uuid: string) {
        throw new Error("Not implemented");
    }
}

