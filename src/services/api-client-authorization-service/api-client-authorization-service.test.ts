// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import axios, { AxiosInstance } from "axios";
import { ApiClientAuthorizationService } from "./api-client-authorization-service";


describe('ApiClientAuthorizationService', () => {
    let apiClientAuthorizationService: ApiClientAuthorizationService;
    let serverApi: AxiosInstance;
    let actions;

    beforeEach(() => {
        serverApi = axios.create();
        actions = {
            progressFn: jest.fn(),
        } as any;
        apiClientAuthorizationService = new ApiClientAuthorizationService(serverApi, actions);
    });

    describe('createCollectionSharingToken', () => {
        it('should return error on invalid collection uuid', () => {
            expect(() => apiClientAuthorizationService.createCollectionSharingToken("foo")).toThrowError("UUID foo is not a collection");
        });

        it('should make a create request with proper scopes and no expiration date', async () => {
            serverApi.post = jest.fn(() => Promise.resolve(
                { data: { uuid: 'zzzzz-4zz18-0123456789abcde' } }
            ));
            const uuid = 'zzzzz-4zz18-0123456789abcde'
            await apiClientAuthorizationService.createCollectionSharingToken(uuid);
            expect(serverApi.post).toHaveBeenCalledWith(
                '/api_client_authorizations', {
                    scopes: [
                        `GET /arvados/v1/collections/${uuid}`,
                        `GET /arvados/v1/collections/${uuid}/`,
                        `GET /arvados/v1/keep_services/accessible`,
                    ]
                }
            );
        });

        it('should make a create request with proper scopes and expiration date', async () => {
            serverApi.post = jest.fn(() => Promise.resolve(
                { data: { uuid: 'zzzzz-4zz18-0123456789abcde' } }
            ));
            const uuid = 'zzzzz-4zz18-0123456789abcde'
            const expDate = new Date(2022, 8, 28, 12, 0, 0);
            await apiClientAuthorizationService.createCollectionSharingToken(uuid, expDate);
            expect(serverApi.post).toHaveBeenCalledWith(
                '/api_client_authorizations', {
                    scopes: [
                        `GET /arvados/v1/collections/${uuid}`,
                        `GET /arvados/v1/collections/${uuid}/`,
                        `GET /arvados/v1/keep_services/accessible`,
                    ],
                    expires_at: expDate.toUTCString()
                }
            );
        });
    });

    describe('listCollectionSharingToken', () => {
        it('should return error on invalid collection uuid', () => {
            expect(() => apiClientAuthorizationService.listCollectionSharingTokens("foo")).toThrowError("UUID foo is not a collection");
        });

        it('should make a list request with proper scopes', async () => {
            serverApi.get = jest.fn(() => Promise.resolve(
                { data: { items: [{}] } }
            ));
            const uuid = 'zzzzz-4zz18-0123456789abcde'
            await apiClientAuthorizationService.listCollectionSharingTokens(uuid);
            expect(serverApi.get).toHaveBeenCalledWith(
                `/api_client_authorizations`, {params: {
                    filters: JSON.stringify([["scopes","=",[
                        `GET /arvados/v1/collections/${uuid}`,
                        `GET /arvados/v1/collections/${uuid}/`,
                        'GET /arvados/v1/keep_services/accessible',
                    ]]]),
                    select: undefined,
                }}
            );
        });
    });
});