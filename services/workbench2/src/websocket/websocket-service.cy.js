// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { WebSocketService } from "./websocket-service";

describe('WebSocketService', () => {

    let mockAuthService;
    let webSocketStub;

    beforeEach(() => {
        webSocketStub = (url) => {
            // Not testing the open delay so we start it as OPEN instead of CONNECTING
            let readyState = WebSocket.OPEN;
            const eventListeners = {};

            const fakeWebSocket = {
                url,
                readyState,
                send: cy.stub().as('send'),
                close: cy.stub().callsFake(() => {
                    readyState = WebSocket.CLOSED;
                }),
                addEventListener: (event, callback) => {
                    if (!eventListeners[event]) {
                        eventListeners[event] = [];
                    }
                    eventListeners[event].push(callback);
                },
            };

            // Use settimeout to allow open callback to be set after WS is
            // constructed so that the callback is set when fired
            // Setting OPEN would be correct here but we aren't testing the
            // connection delay
            setTimeout(() => {
                if (eventListeners['open']) {
                    eventListeners['open'].forEach(callback => callback());
                }
            }, 0);

            return fakeWebSocket;
        }

        // Stub the global WebSocket
        cy.stub(window, 'WebSocket', url => webSocketStub(url));

        // Mock auth service
        mockAuthService = {
            getApiToken: cy.stub().returns('mock-token'),
        };
    });

    afterEach(() => {
        // Clear out singleton instance in between tests
        WebSocketService['instance'] = undefined;
    });

    it('should operate as a singleton and allow externally checking connection status', () => {
        const webSocketService = WebSocketService.getInstance();
        // Verify isActive is false
        expect(webSocketService.isActive()).to.be.false;

        // Connect the WebSocket
        webSocketService.connect('wss://mockurl', mockAuthService);

        // Check that connection is established
        expect(webSocketService.isActive()).to.be.true;

        // Verify singleton behavior
        const anotherInstance = WebSocketService.getInstance();
        expect(anotherInstance).to.equal(webSocketService); // Should be the same instance
        expect(anotherInstance.isActive()).to.be.true; // Should also reflect the active connection
    });

    it('should fire open callback after connecting', () => {
        const webSocketService = WebSocketService.getInstance();
        // Verify isActive is false
        expect(webSocketService.isActive()).to.be.false;

        // Connect the WebSocket
        webSocketService.connect('wss://mockurl', mockAuthService);

        // Check that connection is established
        expect(webSocketService.isActive()).to.be.true;

        // Check that the service sent a subscribe request after open
        cy.get('@send').should('have.been.calledWith', '{"method":"subscribe"}');
    });
});
