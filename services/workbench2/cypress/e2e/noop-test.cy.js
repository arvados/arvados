// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

describe.only("NoOp Timing Tests", function () {
    let activeUser;
    let adminUser;

    before(function () {
        cy.getUser("admin", "Admin", "User", true, true)
            .as("adminUser")
            .then(function () {
                adminUser = this.adminUser;
            });
        cy.getUser("activeuser", "Active", "User", false, true)
            .as("activeUser")
            .then(function () {
                activeUser = this.activeUser;
            });
    });

    it('uses loginAs() once as normal user', () => {
        cy.loginAs(activeUser);
    });

    it('uses loginAs() once as admin user', () => {
        cy.loginAs(adminUser);
    });

    it('uses loginAs() as both users', () => {
        cy.loginAs(activeUser);
        cy.loginAs(adminUser);
    });

    it('switches between users 10 times', () => {
        cy.loginAs(activeUser);
        cy.loginAs(adminUser);
        cy.loginAs(activeUser);
        cy.loginAs(adminUser);
        cy.loginAs(activeUser);
        cy.loginAs(adminUser);
        cy.loginAs(activeUser);
        cy.loginAs(adminUser);
        cy.loginAs(activeUser);
        cy.loginAs(adminUser);
    });
});
