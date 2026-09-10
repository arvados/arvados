// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

function testResourceAPIDetails(token, resourceType, testResource) {
    cy.doToolbarAction("API Details");

    // 'API Response' is default activated tab.
    cy.get(".MuiDialog-container .MuiTabs-root .Mui-selected")
        .should("contain", "API Response");

    // Obtain API record of resource.
    cy.getResource(token, resourceType, testResource.uuid)
        .then((resource) => {
            // In the content area, the <pre> element contains a valid JSON
            // document that is a subset of the full API record.
            cy.get(".MuiDialogContent-root pre")
                .invoke("text")
                .then(JSON.parse)
                .should((displayedData) => {
                    expect(resource).to.deep.include(displayedData);
                });
        });
}

describe("'API Details' dialog showing valid 'API Response'", () => {
    let adminUser;
    let activeUser;

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
        cy.getUser("collectionuser1", "Collection", "User", false, true)
            .as("activeUser")
            .then(function () {
                activeUser = this.activeUser;
            });
    });

    it("works for Collection", () => {
        cy.loginAs(activeUser);
        cy.createCollection(adminUser.token, {
            name: `Test collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: activeUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n",
            properties: {
                foo: "bar",
                baz: ["quux", true, false, {}],
                spam: {
                    ham: null
                }
            }
        }).then((testCollection) => {
            cy.goToPath(`/collections/${testCollection.uuid}`);
            testResourceAPIDetails(activeUser.token, "collections", testCollection);
        });
    });

    it("works for User", () => {
        cy.loginAs(activeUser);
        // Go to home project.
        cy.goToPath(`/projects/${activeUser.user.uuid}`);

        testResourceAPIDetails(activeUser.token, "users", activeUser.user);
    });

    it("works for Project", () => {
        cy.loginAs(activeUser);
        cy.createProject({
            owningUser: activeUser,
            projectName: "api-details-test"
        }).then((testProject) => {
            cy.goToPath(`/projects/${testProject.uuid}`);
            testResourceAPIDetails(activeUser.token, "groups", testProject);
        });
    });
});
