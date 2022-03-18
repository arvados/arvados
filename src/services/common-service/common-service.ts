// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { camelCase, isPlainObject, isArray, snakeCase } from "lodash";
import { AxiosInstance, AxiosPromise, AxiosRequestConfig } from "axios";
import uuid from "uuid/v4";
import { ApiActions } from "services/api/api-actions";
import QueryString from "query-string";
import { Session } from "models/session";

interface Errors {
    status: number;
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
    includeOldVersions?: boolean;
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
    protected readOnlyFields: string[];

    constructor(serverApi: AxiosInstance, resourceType: string, actions: ApiActions, readOnlyFields: string[] = []) {
        this.serverApi = serverApi;
        this.resourceType = resourceType;
        this.actions = actions;
        this.readOnlyFields = readOnlyFields;
    }

    static mapResponseKeys = (response: { data: any }) =>
        CommonService.mapKeys(camelCase)(response.data)

    static mapKeys = (mapFn: (key: string) => string) =>
        (value: any): any => {
            switch (true) {
                case isPlainObject(value):
                    return Object
                        .keys(value)
                        .map(key => [key, mapFn(key)])
                        .reduce((newValue, [key, newKey]) => ({
                            ...newValue,
                            [newKey]: (key === 'items') ? CommonService.mapKeys(mapFn)(value[key]) : value[key]
                        }), {});
                case isArray(value):
                    return value.map(CommonService.mapKeys(mapFn));
                default:
                    return value;
            }
        }

    protected validateUuid(uuid: string) {
        if (uuid === "") {
            throw new Error('UUID cannot be empty string');
        }
    }

    static defaultResponse<R>(promise: AxiosPromise<R>, actions: ApiActions, mapKeys = true, showErrors = true): Promise<R> {
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
                errors.status = response.status;
                actions.errorFn(reqId, errors, showErrors);
                throw errors;
            });
    }

    create(data?: Partial<T>, showErrors?: boolean) {
        return CommonService.defaultResponse(
            this.serverApi
                .post<T>(`/${this.resourceType}`, data && CommonService.mapKeys(snakeCase)(data)),
            this.actions,
            true, // mapKeys
            showErrors
        );
    }

    delete(uuid: string): Promise<T> {
        this.validateUuid(uuid);
        return CommonService.defaultResponse(
            this.serverApi
                .delete(`/${this.resourceType}/${uuid}`),
            this.actions
        );
    }

    get(uuid: string, showErrors?: boolean, select?: string[], session?: Session) {
        this.validateUuid(uuid);

        const cfg: AxiosRequestConfig = {};
        if (session) {
            cfg.baseURL = session.baseUrl;
            cfg.headers = { 'Authorization': 'Bearer ' + session.token };
        }

        return CommonService.defaultResponse(
            this.serverApi
                .get<T>(`/${this.resourceType}/${uuid}`, session ? cfg : undefined),
            this.actions,
            true, // mapKeys
            showErrors
        );
    }

    list(args: ListArguments = {}, showErrors?: boolean): Promise<ListResults<T>> {
        const { filters, select, ...other } = args;
        const params = {
            ...CommonService.mapKeys(snakeCase)(other),
            filters: filters ? `[${filters}]` : undefined,
            select: select
                ? `[${select.map(snakeCase).map(s => `"${s}"`).join(', ')}]`
                : undefined
        };

        if (QueryString.stringify(params).length <= 1500) {
            return CommonService.defaultResponse(
                this.serverApi.get(`/${this.resourceType}`, { params }),
                this.actions,
                showErrors
            );
        } else {
            // Using the POST special case to avoid URI length 414 errors.
            const formData = new FormData();
            formData.append("_method", "GET");
            Object.keys(params).forEach(key => {
                if (params[key] !== undefined) {
                    formData.append(key, params[key]);
                }
            });
            return CommonService.defaultResponse(
                this.serverApi.post(`/${this.resourceType}`, formData, {
                    params: {
                        _method: 'GET'
                    }
                }),
                this.actions,
                showErrors
            );
        }
    }

    update(uuid: string, data: Partial<T>) {
        this.validateUuid(uuid);
        return CommonService.defaultResponse(
            this.serverApi
                .put<T>(`/${this.resourceType}/${uuid}`, data && CommonService.mapKeys(snakeCase)(data)),
            this.actions
        );
    }
}
