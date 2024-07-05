// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { BannerComponent } from './banner';
import servicesProvider from 'common/service-provider';
import { Provider } from "react-redux";
import { ThemeProvider } from "@mui/material";
import { CustomTheme } from 'common/custom-theme';
import { configureStore } from "store/store";
import { createBrowserHistory } from "history";
import { createServices } from "services/services";

describe('<BannerComponent />', () => {

    let props;

    beforeEach(() => {
        props = {
            isOpen: true,
            bannerUUID: undefined,
            keepWebInlineServiceUrl: '',
            openBanner: cy.stub(),
            closeBanner: cy.stub(),
            classes: {},
        }
    });

    const services = createServices("/arvados/v1");
    const store = configureStore(createBrowserHistory(), services);

    it('renders without crashing', () => {
        // when
        cy.mount(
            <Provider store={store}>
              <ThemeProvider theme={CustomTheme}>
                <BannerComponent {...props} />
              </ThemeProvider>
            </Provider>);

        // then
        cy.get('button').should('exist');
    });

    it('calls collectionService', () => {
        // given
        props.isOpen = true;
        props.bannerUUID = '123';

        cy.spy(servicesProvider, 'getServices').as('getServices');
        cy.spy(servicesProvider.getServices().collectionService, 'files').as('files');
        cy.spy(servicesProvider.getServices().collectionService, 'getFileContents').as('getFileContents');

        // when
        cy.mount(
            <Provider store={store}>
              <ThemeProvider theme={CustomTheme}>
                <BannerComponent {...props} />
              </ThemeProvider>
            </Provider>);

        // then
        cy.get('@getServices').should('be.called');
        cy.get('@files').should('be.called');
        cy.get('@getFileContents').should('be.called');
        cy.get('html').should('contain', 'Test banner message');
    });
});

