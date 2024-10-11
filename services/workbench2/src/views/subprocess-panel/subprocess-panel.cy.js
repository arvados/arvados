// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Provider } from 'react-redux';
import configureMockStore from 'redux-mock-store';
import thunk from 'redux-thunk';
import { SubprocessPanel } from './subprocess-panel';
import { ThemeProvider } from '@mui/material';
import { CustomTheme } from 'common/custom-theme';
import { ResourceName, ResourceUuid } from 'views-components/data-explorer/renderers';

const middlewares = [thunk];
const mockStore = configureMockStore(middlewares);

describe('SubprocessPanel', () => {
    let activeUser;
    let defaultStore;

    before(function () {
        cy.getUser('active', 'Active', 'User', false, true)
            .as('activeUser')
            .then(function () {
                activeUser = this.activeUser;
            });
    });

    beforeEach(function () {
        defaultStore = {
            resources: {
                subprocess1: { name: 'subprocess1', uuid: 'subprocess1' },
                subprocess2: { name: 'subprocess2', uuid: 'subprocess2' },
            },
            dataExplorer: {
                subprocessPanel: {
                    items: ['subprocess1', 'subprocess2'],
                    fetchMode: 0,
                    columns: [
                        {
                            name: 'Name',
                            selected: true,
                            configurable: true,
                            filters: [],
                            sort: null,
                            render: (uuid) => <ResourceName uuid={uuid} />,
                        },
                        {
                            name: 'uuid',
                            selected: true,
                            configurable: true,
                            filters: [],
                            sort: null,
                            render: (uuid) => <ResourceUuid uuid={uuid} />,
                        }
                    ],
                    itemsAvailable: 0,
                    loadingItemsAvailable: false,
                    page: 0,
                    rowsPerPage: 50,
                    rowsPerPageOptions: [10, 20, 50, 100, 200, 500],
                    searchValue: '',
                    requestState: 0,
                    countRequestState: 0,
                    isNotFound: false,
                },
            },
            multiselect: {
                checkedList: {},
            },
            router: {},
            progressIndicator: [],
            properties: {},
            searchBar: {},
            detailsPanel: {},

            favorites: {},
            publicFavorites: {},
        };
    });

    it('subprocess panel renders correctly', () => {
        const store = mockStore(defaultStore);
        cy.mount(
            <Provider store={store}>
                <ThemeProvider theme={CustomTheme}>
                    <SubprocessPanel />
                </ThemeProvider>
            </Provider>
        );

        // this test is only to verify that the component is rendered and does not crash
        cy.get('[data-cy=multiselect-checkbox-subprocess1]').should('be.visible');
        cy.get('[data-cy=multiselect-checkbox-subprocess2]').should('be.visible');

        cy.get('[data-cy=multiselect-checkbox-subprocess1]').invoke('prop', 'checked', true).should('be.checked');
        cy.get('[data-cy=multiselect-checkbox-subprocess2]').invoke('prop', 'checked', true).should('be.checked');
    });
});
