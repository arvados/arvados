// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import configureMockStore from 'redux-mock-store'
import { Provider } from 'react-redux';
import { CollectionFileViewerAction } from './collection-file-viewer-action';
import { ContextMenuKind } from '../menu-item-sort';
import { createTree, initTreeNode, setNode } from "models/tree";
import { getInlineFileUrl, sanitizeToken } from "./helpers";

const middlewares = [];
const mockStore = configureMockStore(middlewares);

describe('CollectionFileViewerAction', () => {
    let defaultStore;
    const fileUrl = "https://download.host:12345/c=abcde-4zz18-abcdefghijklmno/t=v2/token2/token3/cat.jpg";
    const insecureKeepInlineUrl = "https://download.host:12345/";
    const secureKeepInlineUrl = "https://*.collections.host:12345/";

    beforeEach(() => {
        let filesTree = createTree();
        let data = {id: "000", value: {"url": fileUrl}};
        filesTree = setNode(initTreeNode(data))(filesTree);

        defaultStore = {
            auth: {
                config: {
                    keepWebServiceUrl: "https://download.host:12345/",
                    keepWebInlineServiceUrl: insecureKeepInlineUrl,
                    clusterConfig: {
                        Collections: {
                          TrustAllContent: false
                        }
                    }
                }
            },
            contextMenu: {
                resource: {
                    uuid: "000",
                    menuKind: ContextMenuKind.COLLECTION_FILE_ITEM,
                }
            },
            collectionPanel: {
                item: {
                    uuid: ""
                }
            },
            collectionPanelFiles: filesTree
        };
    });

    it('should hide open in new tab when unsafe', () => {
        // given
        const store = mockStore(defaultStore);

        // when
        cy.mount(<Provider store={store}>
            <CollectionFileViewerAction />
        </Provider>);

        // ensure cy.mount has been successful
        cy.get('[data-cy-root').should('exist');

        // and
        cy.get('[data-cy=open-in-new-tab]').should('have.length', 0);
    });

    it('should show open in new tab when TrustAllContent=true', () => {
        // given
        let initialState = defaultStore;
        initialState.auth.config.clusterConfig.Collections.TrustAllContent = true;
        const store = mockStore(initialState);

        // when
        cy.mount(<Provider store={store}>
            <CollectionFileViewerAction />
        </Provider>);

        // then
        cy.get('[data-cy=open-in-new-tab]').should('exist');

        // and
        cy.get('[data-cy=open-in-new-tab]').should(
            'have.attr',
            'href',
            sanitizeToken(getInlineFileUrl(fileUrl, initialState.auth.config.keepWebServiceUrl, initialState.auth.config.keepWebInlineServiceUrl))
        );
    });

    it('should show open in new tab when inline url is secure', () => {
        // given
        let initialState = defaultStore;
        initialState.auth.config.keepWebInlineServiceUrl = secureKeepInlineUrl;
        const store = mockStore(initialState);

        // when
        cy.mount(<Provider store={store}>
            <CollectionFileViewerAction />
        </Provider>);

        // then
        cy.get('[data-cy=open-in-new-tab]').should('exist');

        // and
        sanitizeToken(getInlineFileUrl(fileUrl, initialState.auth.config.keepWebServiceUrl, initialState.auth.config.keepWebInlineServiceUrl))
    });
});
