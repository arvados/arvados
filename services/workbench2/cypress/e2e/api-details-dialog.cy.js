// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

function doTestWithResource(token, resourceType, testResource) {
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

    before(function () {
        // Only set up common users once. These aren't set up as aliases because
        // aliases are cleaned up after every test. Also it doesn't make sense
        // to set the same users on beforeEach() over and over again, so we
        // separate a little from Cypress' 'Best Practices' here.
        cy.getUser('admin', 'Admin', 'User', true, true)
            .as("adminUser")
            .then(function () {
                adminUser = this.adminUser;
            });
    });

    it("works for Collection", () => {
        cy.createCollection(adminUser.token, {
            name: `Test collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: adminUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n",
            properties: {
                foo: "bar",
                baz: ["quux", true, false, {}],
                spam: {
                    ham: null
                }
            }
        }).then((testCollection) => {
            cy.loginAs(adminUser);
            cy.goToPath(`/collections/${testCollection.uuid}`);

            doTestWithResource(adminUser.token, "collections", testCollection);
        });
    });

    it("works for User", () => {
        cy.loginAs(adminUser);
        // Go to home project.
        cy.goToPath(`/projects/${adminUser.user.uuid}`);

        doTestWithResource(adminUser.token, "users", adminUser.user);
    });

    it("works for Project", () => {
        cy.createProject({
            owningUser: adminUser,
            projectName: "api-details-test"
        }).then((testProject) => {
            cy.loginAs(adminUser);
            cy.goToPath(`/projects/${testProject.uuid}`);

            doTestWithResource(adminUser.token, "groups", testProject);
        });
    });
});
