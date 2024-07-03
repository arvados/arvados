// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

// This mocks react-copy-to-clipboard's dependency module to avoid warnings
// from jest when running tests. As we're not testing copy-to-clipboard, it's
// safe to just mock it.
// https://github.com/nkbt/react-copy-to-clipboard/issues/106#issuecomment-605227151
// jest.mock('copy-to-clipboard', () => {
//   return jest.fn();
// });

import React from 'react';
import { TokenDialogComponent } from './token-dialog';
import { ThemeProvider } from "@mui/material";
import { CustomTheme } from 'common/custom-theme';
import { combineReducers, createStore } from "redux";
import { Provider } from "react-redux";

describe('<CurrentTokenDialog />', () => {
  let props;
  let store;

  beforeEach(() => {
    props = {
      classes: {},
      token: 'xxxtokenxxx',
      apiHost: 'example.com',
      open: true,
      dispatch: cy.spy().as('dispatch'),
    };

    const initialAuthState = {
      localCluster: "zzzzz",
      remoteHostsConfig: {},
      sessions: {},
    };

    store = createStore(combineReducers({
      auth: (state = initialAuthState, action) => state,
    }));
  });

  describe('Get API Token dialog', () => {
    beforeEach(() => {
      cy.mount(
        <Provider store={store}>
          <ThemeProvider theme={CustomTheme}>
            <TokenDialogComponent {...props} />
          </ThemeProvider>
        </Provider>);
    });

    it('should include API host and token', () => {
      cy.get('pre').contains('export ARVADOS_API_HOST=example.com');
      cy.get('pre').contains('export ARVADOS_API_TOKEN=xxxtokenxxx');
    });

    it('should show the token expiration if present', () => {
      expect(props.tokenExpiration).to.be.undefined;
      cy.get('[data-cy=details-attribute-value]').contains('This token does not have an expiration date');

      const someDate = '2140-01-01T00:00:00.000Z'
      props.tokenExpiration = new Date(someDate);
      cy.mount(
        <Provider store={store}>
          <ThemeProvider theme={CustomTheme}>
            <TokenDialogComponent {...props} />
          </ThemeProvider>
        </Provider>);
      cy.get('[data-cy=details-attribute-value]').contains(props.tokenExpiration.toLocaleString());
    });

    it('should show a create new token button when allowed', () => {
      expect(!!props.canCreateNewTokens).to.equal(false);
      cy.contains('GET NEW TOKEN').should('not.exist');

      props.canCreateNewTokens = true;
      cy.mount(
        <Provider store={store}>
          <ThemeProvider theme={CustomTheme}>
            <TokenDialogComponent {...props} />
          </ThemeProvider>
        </Provider>);
      cy.contains('GET NEW TOKEN').should('exist');
    });
  });

  describe('Copy link to clipboard button', () => {
    beforeEach(() => {
      cy.mount(
        <Provider store={store}>
          <ThemeProvider theme={CustomTheme}>
            <TokenDialogComponent {...props} />
          </ThemeProvider>
        </Provider>);
    });

    it('should copy API TOKEN to the clipboard', () => {
      cy.get('button').contains('Copy').click();
      cy.get('@dispatch').should('be.calledWith', {
        payload: {
          hideDuration: 2000,
          kind: 1,
          message: 'Shell code block copied',
        },
        type: 'OPEN_SNACKBAR',
      });
    });
  });
});
