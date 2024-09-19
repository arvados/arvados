// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

describe("AuthMiddleware", () => {
    let activeUser;
    let adminUser;

    before(function () {
        // Only set up common users once. These aren't set up as aliases because
        // aliases are cleaned up after every test. Also it doesn't make sense
        // to set the same users on beforeEach() over and over again, so we
        // separate a little from Cypress' 'Best Practices' here.
        cy.getUser("admin", "Admin", "User", true, true)
            .as("adminUser")
            .then(function () {
                adminUser = this.adminUser;
            });
        cy.getUser("user", "Active", "User", false, true)
            .as("activeUser")
            .then(function () {
                activeUser = this.activeUser;
            });
    });

    it("handles LOGOUT action", () => {
        cy.loginAs(activeUser);
        cy.waitForDom();
        // verify that the token is stored in localStorage
        cy.window().then(win => {
            expect(win.localStorage.getItem('apiToken')).to.equal(activeUser.token);
        });

            // logout
            cy.get('[aria-label="Account Management"]').click();
            cy.get('[data-cy=logout-menuitem]').click();

            cy.window().then(win => {
                // verify that logout has been successful
                cy.contains("Please log in.").should("exist"); 
                expect(win.localStorage.getItem('apiToken')).to.be.null;
            });
    });
});
