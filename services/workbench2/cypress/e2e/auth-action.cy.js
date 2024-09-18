// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

// import { getNewExtraToken, initAuth } from "../../src/store/auth/auth-action";
import { API_TOKEN_KEY } from 'services/auth-service/auth-service';

// import 'jest-localstorage-mock';
// import { ServiceRepository, createServices } from "services/services";
// import { configureStore, RootStore } from "../../src/store/store"; //causes tuppy failure
// import { createBrowserHistory } from "history";
// import { mockConfig } from 'common/config';
// import { ApiActions } from "services/api/api-actions";
import { ACCOUNT_LINK_STATUS_KEY } from 'services/link-account-service/link-account-service';
import Axios, { AxiosInstance } from 'axios';
// import MockAdapter from "axios-mock-adapter";
// import { ImportMock } from 'ts-mock-imports';
// import * as servicesModule from "services/services";
// import * as authActionSessionModule from "../../src/store/auth/auth-action-session";
// import { SessionStatus } from "models/session";
import { getRemoteHostConfig } from '../../src/store/auth/auth-action-session';

describe('auth-actions', () => {
    // let axiosInst;
    // let axiosMock;

    // let store;
    // let services;
    // const config = {};
    // const actions = {
    //     progressFn: (id, working) => { },
    //     errorFn: (id, message) => { }
    // };
    // let importMocks;

    const localClusterTokenExpiration = '2020-01-01T00:00:00.000Z';
    const loginClusterTokenExpiration = '2140-01-01T00:00:00.000Z';

    let activeUser;
    let adminUser;

    before(function () {
        // Only set up common users once. These aren't set up as aliases because
        // aliases are cleaned up after every test. Also it doesn't make sense
        // to set the same users on beforeEach() over and over again, so we
        // separate a little from Cypress' 'Best Practices' here.
        cy.getUser('admin', 'Admin', 'User', true, true)
            .as('adminUser')
            .then(function () {
                adminUser = this.adminUser;
            });
        cy.getUser('user', 'Active', 'User', false, true)
            .as('activeUser')
            .then(function () {
                activeUser = this.activeUser;
            });
    });

    it('creates an extra token', () => {
        let firstToken;
        let firstExtraToken;

        cy.loginAs(activeUser);
        cy.waitForDom();
        cy.waitForLocalStorage('arvadosStore').then((storedStore) => {
            const store = JSON.parse(storedStore);

            //check that no extra token was requested
            expect(store.auth.apiToken).to.not.be.undefined;
            expect(store.auth.extraApiToken).to.be.undefined;
            firstToken = store.auth.apiToken;
        });

        //ask for an extra token
        cy.get('[aria-label="Account Management"]').click();
        cy.contains('Get API token').click();
        cy.contains('GET NEW TOKEN').click();
        cy.waitForLocalStorage('arvadosStore').then((storedStore) => {
            const store = JSON.parse(storedStore);

            // check that cached token is used
            expect(store.auth.apiToken).to.equal(firstToken);
            cy.waitForLocalStorageUpdate('arvadosStore');

            //check that an extra token was requested
            expect(store.auth.extraApiToken).to.not.be.undefined;
            firstExtraToken = store.auth.extraApiToken;
        });
        //check that another request generates a new token
        cy.contains('GET NEW TOKEN').click();
        cy.waitForLocalStorageUpdate('arvadosStore');
        cy.waitForLocalStorage('arvadosStore').then((storedStore) => {
            const store = JSON.parse(storedStore);

            expect(store.auth.apiToken).to.not.be.undefined;
            expect(store.auth.extraApiToken).to.not.be.undefined;
            expect(store.auth.extraApiToken).to.not.equal(firstExtraToken);
        });
    });

    it('requests remote token and token expiration', () => {
        cy.loginAs(adminUser);
        cy.waitForLocalStorage('arvadosStore').then((storedStore) => {
            const store = JSON.parse(storedStore);

            // verify that the token is cached
            expect(store.auth.apiToken).to.not.be.undefined;
            expect(localClusterTokenExpiration).to.not.equal(loginClusterTokenExpiration);

            const now = new Date();
            const expiration = new Date(store.auth.apiTokenExpiration);
            const expectedExpiration = new Date(now.getTime() + 24 * 60 * 60 * 1000 + 2000);
            const timeDiff = Math.abs(expectedExpiration.getMilliseconds() - expiration.getMilliseconds());

            // verify that the token expiration is ~24 hours from now (with a 2 second buffer)
            expect(timeDiff).to.be.lessThan(2000);
        });
    });

    //TODO: finish this test, maybe convert back to component test?

    // it('should initialise state with user and api token from local storage', () => {
    //     let apiToken;
    //     cy.loginAs(activeUser);

    //     cy.waitForLocalStorage('apiToken').then((storedToken) => {
    //         apiToken = storedToken;

    //         // logout
    //         cy.get('[aria-label="Account Management"]').click();
    //         cy.get('[data-cy=logout-menuitem]').click();

    //         // verify logout
    //         cy.window().then((win) => {
    //             cy.contains('Please log in.').should('exist');
    //             expect(win.localStorage.getItem('apiToken')).to.be.null;
    //         });
    //     });

    //     cy.visit('/');
    //     cy.waitForLocalStorage('arvadosStore').then((storedStore) => {
    //         const store = JSON.parse(storedStore);
    //         console.log(store);
    //         const auth = store.auth;
    //         console.log(JSON.stringify(auth.user));
    //         expect(auth.user).to.deep.equal({
    //             email: 'user@example.local',
    //             firstName: 'Active',
    //             lastName: 'User',
    //             uuid: 'zzzzz-tpzed-wlme7goukc5495r',
    //             ownerUuid: 'zzzzz-tpzed-000000000000000',
    //             isAdmin: false,
    //             isActive: true,
    //             username: 'user',
    //             canWrite: true,
    //             canManage: true,
    //             prefs: { profile: {} },
    //         });
    //     });
    // });

    // TODO: Add remaining action tests
    /*
       it('should fire external url to login', () => {
       const initialState = undefined;
       window.location.assign = jest.fn();
       reducer(initialState, authActions.LOGIN());
       expect(window.location.assign).toBeCalledWith(
       `/login?return_to=${window.location.protocol}//${window.location.host}/token`
       );
       });

       it('should fire external url to logout', () => {
       const initialState = undefined;
       window.location.assign = jest.fn();
       reducer(initialState, authActions.LOGOUT());
       expect(window.location.assign).toBeCalledWith(
       `/logout?return_to=${location.protocol}//${location.host}`
       );
       });
     */
});
