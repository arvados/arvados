// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

// ***********************************************
// This example commands.js shows you how to
// create various custom commands and overwrite
// existing commands.
//
// For more comprehensive examples of custom
// commands please read more here:
// https://on.cypress.io/custom-commands
// ***********************************************
//
//
// -- This is a parent command --
// Cypress.Commands.add("login", (email, password) => { ... })
//
//
// -- This is a child command --
// Cypress.Commands.add("drag", { prevSubject: 'element'}, (subject, options) => { ... })
//
//
// -- This is a dual command --
// Cypress.Commands.add("dismiss", { prevSubject: 'optional'}, (subject, options) => { ... })
//
//
// -- This will overwrite an existing command --
// Cypress.Commands.overwrite("visit", (originalFn, url, options) => { ... })

const YAML = require('yamljs');
const arvadosFixturesDir = Cypress.env('fixtures');
const controllerURL = Cypress.env('controller_url');
const systemToken = Cypress.env('system_token');

Cypress.Commands.add(
    "arvadosFixture", (name) => {
        return cy.readFile(arvadosFixturesDir+'/'+name+'.yml').then(
            function (str) {
                return YAML.parse(str);
            }
        )
    }
)

Cypress.Commands.add(
    "resetDB", () => {
        cy.request('POST', `${controllerURL}/database/reset?api_token=${systemToken}`);
    }
)