// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

// ***********************************************************
// This example support/component.ts is processed and
// loaded automatically before your test files.
//
// This is a great place to put global configuration and
// behavior that modifies Cypress.
//
// You can change the location of this file or turn off
// automatically serving support files with the
// 'supportFile' configuration option.
//
// You can read more here:
// https://on.cypress.io/configuration
// ***********************************************************

// Import commands.js using ES2015 syntax:
import './commands'

// Alternatively you can use CommonJS syntax:
// require('./commands')

import { mount } from 'cypress/react'

// Augment the Cypress namespace to include type definitions for
// your custom command.
// Alternatively, can be defined in cypress/support/component.d.ts
// with a <reference path="./component" /> at the top of your spec.
// declare global {
//   namespace Cypress {
//     interface Chainable {
//       mount: typeof mount
//     }
//   }
// }

Cypress.Commands.add('mount', mount)

/*
    The following is a workaraound for Arvados Issue #22483 which is known and persists in Cypress v14+:
    https://github.com/cypress-io/cypress/issues/28644
    The entire if statement can be removed once the bug is fixed by Cypress.
*/
if (window.Cypress) {
    // Prevent chunk loading errors from failing tests
    const originalOnError = window.onerror;
    window.onerror = (msg, source, lineno, colno, err) => {
        if (err && err.message && err.message.includes('Loading chunk')) {
            console.warn('Chunk loading error intercepted:', err);
            return false;
        }
        return originalOnError?.(msg, source, lineno, colno, err);
    };

    window.addEventListener('unhandledrejection', (event) => {
        if (event.reason && event.reason.message && event.reason.message.includes('Loading chunk')) {
            event.preventDefault();
            console.warn('Chunk loading rejection intercepted:', event.reason);
        }
    });

    window.addEventListener('error', (event) => {
        if (event.error?.message?.includes('Loading chunk')) {
            event.preventDefault();
            cy.log('Chunk loading error detected - reloading page');
            window.location.reload();
        }
    });
}

// Example use:
// cy.mount(<MyComponent />)