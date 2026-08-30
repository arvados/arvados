// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

describe("'API Details' dialog tests", () => {
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

    it("shows the expected 'API Response' for a collection", () => {
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
        })
            .then((testCollection) => {
                cy.loginAs(adminUser);
                cy.goToPath(`/collections/${testCollection.uuid}`);

                cy.doToolbarAction("API Details");

                // 'API Response' is default activated tab.
                cy.get(".MuiDialog-container .MuiTabs-root .Mui-selected").should("contain", "API RESPONSE");

                // Obtain API record of collection.
                cy.getCollection(adminUser.token, testCollection.uuid)
                    .then((fullCollection) => {
                        // In the content area, the <pre> element contains a
                        // valid JSON document that is a subset of the API
                        // record.
                        cy.get(".MuiDialogContent-root pre")
                            .invoke("text")
                            .then(JSON.parse)
                            .should((displayedData) => {
                                expect(fullCollection)
                                    .to.deep.include(displayedData);
                            });
                    });
            });
    });
});
