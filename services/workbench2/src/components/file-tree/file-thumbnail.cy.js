// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { FileThumbnail } from "./file-thumbnail";
import { CollectionFileType } from '../../models/collection-file';
import { Provider } from "react-redux";
import { combineReducers, createStore } from "redux";
import { ThemeProvider } from "@mui/material";
import { CustomTheme } from "common/custom-theme";

let store;

describe("<FileThumbnail />", () => {
    let file;

    beforeEach(() => {
        const initialAuthState = {
            config: {
                keepWebServiceUrl: 'http://example.com/',
                keepWebInlineServiceUrl: 'http://*.collections.example.com/',
            }
        }
        store = createStore(combineReducers({
            auth: (state= initialAuthState, action) => state,
        }));

        file = {
            name: 'test-image.jpg',
            type: CollectionFileType.FILE,
            url: 'http://example.com/c=zzzzz-4zz18-0123456789abcde/t=v2/zzzzz-gj3su-0123456789abcde/xxxxxxtokenxxxxx/test-image.jpg',
            size: 300
        };
    });

    it("renders file thumbnail with proper src", () => {
        cy.mount(
            <Provider store={store}>
              <ThemeProvider theme={CustomTheme}>
                <FileThumbnail file={file} />
              </ThemeProvider>
            </Provider>);
        cy.get('img').should('have.attr', 'src', 'http://zzzzz-4zz18-0123456789abcde.collections.example.com/test-image.jpg?api_token=v2/zzzzz-gj3su-0123456789abcde/xxxxxxtokenxxxxx');
    });
});
