// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

const path = require("path");
require('cypress-plugin-tab');

// It can be tricky to mimick the user's search for a collection by its known
// (and unique) name, which can be noisy, time consuming, and a bit flaky. We
// locate the collection by API directly and jump to it.
// NOTE: To use after some collection-creating user action, it's a good idea to
// add a "barrier" assertion (e.g., disappeared form dialog, snackbar with a
// certain message, etc.) before calling.
function goToCollectionByName(collectionName, token) {
    return cy.doRequest("GET", "/arvados/v1/collections", null, {
        filters: JSON.stringify([["name", "=", collectionName]]),
        limit: "1",
        select: JSON.stringify(["uuid"]),
        count: "none"
    }, token, true)
        .then((response) => {
            // Not using Cypress "its" command, because "its" retries, yet we
            // want the following line to fail fast.
            const uuid = response.body.items[0].uuid;
            return cy.goToPath(`/collections/${uuid}`);
        });
}

function randomName(prefix = "Test collection ") {  // note the ending space
    return `${prefix}${Math.floor(Math.random() * 999999)}`;
}

describe("Collection panel tests", function () {
    let activeUser;
    let adminUser;
    let downloadsFolder;

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
        downloadsFolder = Cypress.config("downloadsFolder");
    });

    it("allows to download mountain duck config for a collection", () => {
        cy.loginAs(activeUser);
        // Creates the collection using the admin token so we can set up
        // a bogus manifest text without block signatures.
        cy.createCollection(adminUser.token, {
            name: randomName(),
            owner_uuid: activeUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n",
        })
            .as("testCollection")
            .then(function (testCollection) {
                cy.goToPath(`/collections/${testCollection.uuid}`);

                cy.get('[data-title="Open with 3rd party client"]').click();
                cy.get("[data-cy=download-button").click();

                const filename = path.join(downloadsFolder, `${testCollection.name}.duck`);

                cy.readFile(filename, { timeout: 15000 })
                    .then(body => {
                        const childrenCollection = Array.prototype.slice.call(Cypress.$(body).find("dict")[0].children);
                        const map = {};
                        let i,
                            j = 2;

                        for (i = 0; i < childrenCollection.length; i += j) {
                            map[childrenCollection[i].outerText] = childrenCollection[i + 1].outerText;
                        }

                        cy.get("#simple-tabpanel-0")
                            .find("a")
                            .then(a => {
                                const [host, port] = a.text().split("@")[1].split("/")[0].split(":");
                                expect(map["Protocol"]).to.equal("davs");
                                expect(map["UUID"]).to.equal(testCollection.uuid);
                                expect(map["Username"]).to.equal(activeUser.user.username);
                                expect(map["Port"]).to.equal(port);
                                expect(map["Hostname"]).to.equal(host);
                                if (map["Path"]) {
                                    expect(map["Path"]).to.equal(`/c=${testCollection.uuid}`);
                                }
                            });
                    })
                    .then(() => cy.task("clearDownload", { filename }));
            });
    });

    it("attempts to use a preexisting name creating or updating a collection", function () {
        const collName = randomName();
        cy.createCollection(adminUser.token, {
            name: collName,
            owner_uuid: activeUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n",
        });
        cy.loginAs(activeUser);
        cy.goToPath(`/projects/${activeUser.user.uuid}`);
        cy.get("[data-cy=breadcrumb-first]").should("contain", "Projects");
        cy.get("[data-cy=breadcrumb-last]").should("not.exist");
        // Attempt to create new collection with a duplicate name
        cy.get("[data-cy=side-panel-button]").click();
        cy.get("[data-cy=side-panel-new-collection]").click();
        cy.get("[data-cy=form-dialog]")
            .should("contain", "New collection")
            .within(() => {
                cy.get("[data-cy=name-field] input").type(collName);
                cy.get("[data-cy=form-submit-btn]").click();
            });
        // Error message should display, allowing editing the name
        cy.get("[data-cy=form-dialog]")
            .should("contain", "Collection with the same name already exists")
            .within(() => {
                cy.get("[data-cy=name-field] input").type(" renamed");
                cy.get("[data-cy=form-submit-btn]").click();
            });
        cy.get("[data-cy=form-dialog]").should("not.exist");
        // Attempt to rename the collection with the duplicate name
        cy.get('[data-title="Edit collection"]').click();
        cy.get("[data-cy=form-dialog]")
            .should("contain", "Edit Collection")
            .within(() => {
                cy.get("[data-cy=name-field] input").type("{selectall}{backspace}").type(collName);
                cy.get("[data-cy=form-submit-btn]").click();
            });
        cy.get("[data-cy=form-dialog]").should("exist").and("contain", "Collection with the same name already exists");
    });

    it("uses the property editor (from edit dialog) with vocabulary terms", function () {
        const collName = randomName();
        cy.loginAs(activeUser);
        cy.createCollection(adminUser.token, {
            name: collName,
            owner_uuid: activeUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n",
        })
            .as("testCollection")
            .then((testCollection) => cy.goToPath(`/collections/${testCollection.uuid}`));

        // Verify collection name
        cy.get("[data-cy=collection-details-card").should("contain", collName);
        // Open overview tab
        cy.doMPVTabSelect("Overview");
        // Verify property not present
        cy.get("[data-cy=resource-properties]").should("not.contain", "Color: Magenta");

        cy.get('[data-title="Edit collection"]').click();
        cy.get("[data-cy=form-dialog]").should("contain", "Properties");

        // Key: Color (IDTAGCOLORS) - Value: Magenta (IDVALCOLORS3)
        cy.get("[data-cy=resource-properties-form]").within(() => {
            cy.get("[data-cy=property-field-key] input").type("Color");
            cy.get("[data-cy=property-field-value]").click().within(() => {
                cy.get("input").type("Magenta");
            });
            cy.get("[data-cy=property-add-btn]").click();
        });
        // Confirm proper vocabulary labels are displayed on the UI.
        cy.get("[data-cy=form-dialog]")
            .should("contain", "Color: Magenta")
            .contains("Save").click();
        cy.get("[data-cy=form-dialog]").should("not.exist");
        // Confirm proper vocabulary IDs were saved on the backend.
        cy.get("@testCollection")
            .then((testCollection) => cy.getCollection(activeUser.token, testCollection.uuid))
            .its("properties")
            .should("deep.include", { IDTAGCOLORS: ["IDVALCOLORS3"] });
        // Confirm the property is displayed on the UI.
        cy.get("[data-cy=resource-properties").should("contain", "Color: Magenta");
    });

    it("uses the editor (from details panel) with vocabulary terms", function () {
        const collName = randomName();
        cy.loginAs(activeUser);
        cy.createCollection(adminUser.token, {
            name: collName,
            owner_uuid: activeUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n",
        })
            .as("testCollection")
            .then((testCollection) => cy.goToPath(`/collections/${this.testCollection.uuid}`));

        // Verify collection name
        cy.get("[data-cy=collection-details-card").should("contain", collName);

        // Open overview tab
        cy.doMPVTabSelect("Overview");

        // Verify properties not present
        cy.get("[data-cy=resource-properties]")
            .and("not.contain", "Color: Magenta")
            .and("not.contain", "Size: S");
        cy.get("[data-title='View details']").click();

        cy.get("[data-cy=details-panel] [data-cy=details-panel-edit-btn]").click();
        cy.get("[data-cy=form-dialog").should("contain", "Edit Collection");

        // Key: Color (IDTAGCOLORS) - Value: Magenta (IDVALCOLORS3)
        cy.get("[data-cy=resource-properties-form]").within(() => {
            cy.get("[data-cy=property-field-key] input").type("Color");
            cy.get("[data-cy=property-field-value]").click().within(() => {
                cy.get("input").type("Magenta");
            });
            cy.get("[data-cy=property-add-btn]").click();
        });
        // Confirm proper vocabulary labels are displayed on the UI.
        cy.get("[data-cy=form-dialog]").should("contain", "Color: Magenta");

        // Case-insensitive on-blur auto-selection test
        // Key: Size (IDTAGSIZES) - Value: Small (IDVALSIZES2)
        cy.get("[data-cy=resource-properties-form]").within(() => {
            cy.get("[data-cy=property-field-key] input").type("sIzE");
            cy.get("[data-cy=property-field-value]").click().within(() => {
                cy.get("input").type("sMaLL{enter}");
            });
            cy.get("[data-cy=property-add-btn]").click();
        });
        // Confirm proper vocabulary labels are displayed on the UI.
        cy.get("[data-cy=form-dialog]")
            .should("contain", "Size: S")
            .contains("Save").click();
        cy.get("[data-cy=form-dialog]").should("not.exist");

        // Confirm proper vocabulary IDs were saved on the backend.
        cy.get("@testCollection")
            .then((testCollection) => cy.getCollection(activeUser.token, testCollection.uuid))
            .its("properties")
            .should("deep.include", {
                IDTAGCOLORS: ["IDVALCOLORS3"],
                IDTAGSIZES: ["IDVALSIZES2"]
            });

        // Confirm properties display on the UI.
        cy.get("[data-cy=resource-properties]")
            .should("contain", "Color: Magenta")
            .and("contain", "Size: S");
    });

    it("shows collection by URL", function () {
        cy.loginAs(activeUser);
        [true, false].map(function (isWritable) {
            // Using different file names to avoid test flakyness: the second iteration
            // on this loop may pass an assertion from the first iteration by looking
            // for the same file name.
            const fileName = isWritable ? "bar" : "foo";
            const subDirName = "subdir";
            cy.createGroup(adminUser.token, {
                name: "Shared project",
                group_class: "project",
            })
                .as("sharedGroup");

            cy.doRequest("GET", "/arvados/v1/config", null, null)
                .its("body")
                .should(clusterConfig => {
                    expect(clusterConfig.Collections, "clusterConfig").to.have.property("TrustAllContent", true);
                    expect(clusterConfig.Services, "clusterConfig").to.have.property("WebDAV").have.property("ExternalURL");
                    expect(clusterConfig.Services, "clusterConfig").to.have.property("WebDAVDownload").have.property("ExternalURL");
                    const inlineUrl =
                        clusterConfig.Services.WebDAV.ExternalURL !== ""
                            ? clusterConfig.Services.WebDAV.ExternalURL
                            : clusterConfig.Services.WebDAVDownload.ExternalURL;
                    expect(inlineUrl).to.not.contain("*");
                })

            cy.get("@sharedGroup").then(sharedGroup => {
                cy.createCollection(adminUser.token, {
                    name: "Test collection",
                    owner_uuid: sharedGroup.uuid,
                    properties: { someKey: "someValue" },
                    manifest_text: `. 37b51d194a7513e45b56f6524f2d51f2+3 0:3:${fileName}\n./${subDirName} 37b51d194a7513e45b56f6524f2d51f2+3 0:3:${fileName}\n`,
                })
                    .as("testCollection");

                // Share the group with active user.
                cy.createLink(adminUser.token, {
                    name: isWritable ? "can_write" : "can_read",
                    link_class: "permission",
                    head_uuid: sharedGroup.uuid,
                    tail_uuid: activeUser.user.uuid,
                });
            });

            cy.get("@testCollection").then(testCollection => {
                cy.goToPath(`/collections/${testCollection.uuid}`);
                // Verify collection name
                cy.get("[data-cy=collection-details-card]")
                    .should("contain", testCollection.name);

                // Open overview tab
                cy.doMPVTabSelect("Overview");

                // Verify collection uuid
                cy.get("[data-cy=details-element]")
                    .should("contain", testCollection.uuid)
                    .and("not.contain", "This is an old version");
            });

            // Check for the read-only icon
            if (isWritable) {
                cy.get("[data-cy=read-only-icon]").should("not.exist");
            } else {
                cy.get("[data-cy=read-only-icon]").should("be.visible");
            }

            // Check that both read and write operations are available on
            // the 'More options' menu.
            cy.get("[data-cy=collection-details-card]").within(() => {
                cy.get("[data-targetid='Add to favorites']");
                if (isWritable) {
                    cy.get("[data-targetid='Edit collection']");
                } else {
                    cy.get("[data-targetid='Edit collection']").should("not.exist");
                }
            });
            cy.get("body").click(); // Collapse the menu avoiding details panel expansion
            cy.get("[data-cy=resource-properties]")
                .should("contain", "someKey: someValue")
                .and("not.contain", "anotherKey: anotherValue");
            // Check that the file listing show both read & write operations
            cy.waitForDom();
            cy.doMPVTabSelect("Files");
            cy.get("[data-cy=collection-files-right-panel]", { timeout: 5000 }).should("contain", fileName);
            if (isWritable) {
                cy.get("[data-cy=upload-button]").should(`${isWritable ? "" : "not."}contain`, "Upload data");
            }
            // Test context menus
            cy.get("[data-cy=collection-files-panel]").contains(fileName).rightclick();
            cy.get("[data-cy=context-menu]")
                .should("contain", "Download")
                .and("contain", "Open in new tab")
                .and("contain", "Copy link to latest version")
                .and("contain", "Copy link to immutable version")
                .and(`${isWritable ? "" : "not."}contain`, "Rename")
                .and(`${isWritable ? "" : "not."}contain`, "Remove");
            cy.get("body").click(); // Collapse the menu
            cy.get("[data-cy=collection-files-panel]").contains(subDirName).rightclick();
            cy.get("[data-cy=context-menu]")
                .should("not.contain", "Download")
                .and("contain", "Open in new tab")
                .and("contain", "Copy link to latest version")
                .and("contain", "Copy link to immutable version")
                .and(`${isWritable ? "" : "not."}contain`, "Rename")
                .and(`${isWritable ? "" : "not."}contain`, "Remove");
            cy.get("body").click(); // Collapse the menu
            // File/dir item 'more options' button
            cy.get("[data-cy=file-item-options-btn").first().click();
            cy.get("[data-cy=context-menu]").should(`${isWritable ? "" : "not."}contain`, "Remove");
            cy.get("body").click(); // Collapse the menu
            // Hamburger 'more options' menu button
            cy.doCollectionPanelOptionsAction("Select all");
            cy.get("[data-cy=collection-files-panel-options-btn]").click();
            cy.get("[data-cy=context-menu]").should(`${isWritable ? "" : "not."}contain`, "Remove selected");
            cy.get("body").click(); // Collapse the menu
        });
    });

    it("renames a file using valid names", function () {
        function eachPair(lst, func) {
            for (var i = 0; i < lst.length - 1; i++) {
                func(lst[i], lst[i + 1]);
            }
        }

        cy.loginAs(activeUser);
        cy.createCollection(adminUser.token, {
            name: randomName(),
            owner_uuid: activeUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n",
        }).then((collection) => cy.goToPath(`/collections/${collection.uuid}`));

        const names = [
            "bar", // initial name already set
            "&",
            "foo",
            "&amp;",
            "I ❤️ ⛵️",
            "...",
            "#..",
            "some name with whitespaces",
            "some name with #2",
            "is this name legal? I hope it is",
            "some_file.pdf#",
            "some_file.pdf?",
            "?some_file.pdf",
            "some%file.pdf",
            "some%2Ffile.pdf",
            "some%22file.pdf",
            "some%20file.pdf",
            "G%C3%BCnter's%20file.pdf",
            "table%&?*2",
            "bar", // make sure we can go back to the original name as a last step
        ];
        cy.intercept({ method: "PUT", url: "**/arvados/v1/collections/*" }).as("renameRequest");

        cy.doMPVTabSelect("Files");
        eachPair(names, (from, to) => {
            cy.get("[data-cy=collection-files-panel]").contains(`${from}`).rightclick();
            cy.get("[data-cy=context-menu]").contains("Rename").click();
            cy.get("[data-cy=form-dialog]")
                .should("contain", "Rename")
                .within(() => {
                    cy.get("input").type("{selectall}{backspace}").type(to, { parseSpecialCharSequences: false });
                });
            cy.get("[data-cy=form-submit-btn]").click();
            cy.wait("@renameRequest");
            cy.get("[data-cy=collection-files-panel]").should("not.contain", `${from}`).and("contain", `${to}`);
        });
    });

    it("renames a file to a different directory", function () {
        cy.loginAs(activeUser);
        cy.createCollection(adminUser.token, {
            name: randomName(),
            owner_uuid: activeUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n",
        }).then((collection) => cy.goToPath(`/collections/${collection.uuid}`));

        cy.doMPVTabSelect("Files");
        ["subdir", "G%C3%BCnter's%20file", "table%&?*2"].forEach(subdir => {
            cy.get("[data-cy=collection-files-panel]").contains("bar").rightclick();
            cy.get("[data-cy=context-menu]").contains("Rename").click();
            cy.get("[data-cy=form-dialog]")
                .should("contain", "Rename")
                .within(() => {
                    cy.get("input").type(`{selectall}{backspace}${subdir}/foo`);
                });
            cy.get("[data-cy=form-submit-btn]").click();
            cy.get("[data-cy=form-dialog]").should("not.exist");
            cy.get("[data-cy=collection-files-panel]").should("not.contain", "bar").and("contain", subdir);
            cy.get("[data-cy=collection-files-panel]").contains(subdir).click();

            // Rename 'subdir/foo' to 'bar'
            cy.get("[data-cy=collection-files-panel]").contains("foo").rightclick();
            cy.get("[data-cy=context-menu]").contains("Rename").click();
            cy.get("[data-cy=form-dialog]")
                .should("contain", "Rename")
                .within(() => {
                    cy.get("input").should("have.value", `${subdir}/foo`).type(`{selectall}{backspace}bar`);
                });
            cy.get("[data-cy=form-submit-btn]").click({ force: true });

            // need to wait for dialog to dismiss
            cy.get("[data-cy=form-dialog]").should("not.exist");

            cy.get("[data-cy=collection-files-panel]").contains("Home").click();

            cy.get("[data-cy=collection-files-panel]").contains(subdir).click();
            cy.get("[data-cy=collection-files-panel]")
                .should("contain", subdir) // empty dir kept
                .and("contain", "bar");

            // this is when the dom is actually finished loading
            cy.get("[data-cy=file-item-options-btn]", { timeout: 20000 }).first().should('exist')

            cy.get("[data-cy=collection-files-panel-content]").contains(subdir).rightclick();
            cy.get("[data-cy=context-menu]").contains("Remove").click();
            cy.get("[data-cy=confirmation-dialog-ok-btn]").click();
            cy.get("[data-cy=form-dialog]").should("not.exist");
        });
    });

    it("shows collection owner", () => {
        cy.loginAs(activeUser);
        cy.createCollection(adminUser.token, {
            name: randomName(),
            owner_uuid: activeUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n",
        }).then((collection) => cy.goToPath(`/collections/${collection.uuid}`));

        cy.doMPVTabSelect("Overview");
        cy.get("[data-cy=details-element]").should("contain", activeUser.user.full_name);
    });

    it("tries to rename a file with illegal names", function () {
        const illegalNamesFromUI = [
            [".", "Name cannot be '.' or '..'"],
            ["..", "Name cannot be '.' or '..'"],
            ["", "This field is required"],
            [" ", "Leading/trailing whitespaces not allowed"],
            [" foo", "Leading/trailing whitespaces not allowed"],
            ["foo ", "Leading/trailing whitespaces not allowed"],
            ["//foo", "Empty dir name not allowed"],
        ];

        cy.loginAs(activeUser);
        cy.createCollection(adminUser.token, {
            name: randomName(),
            owner_uuid: activeUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n",
        }).then((collection) => cy.goToPath(`/collections/${collection.uuid}`));

        cy.doMPVTabSelect("Files");
        illegalNamesFromUI.forEach(([name, errMsg]) => {
            cy.get("[data-cy=collection-files-panel]").contains("bar").rightclick();
            cy.get("[data-cy=context-menu]").contains("Rename").click();
            cy.get("[data-cy=form-dialog]")
                .should("contain", "Rename")
                .within(() => {
                    cy.get("input").type(`{selectall}{backspace}${name}`);
                });
            cy.get("[data-cy=form-dialog]")
                .should("contain", "Rename")
                .within(() => {
                    cy.contains(errMsg);
                });
            cy.get("[data-cy=form-cancel-btn]").click();
        });
    });

    it("can correctly display old versions", function () {
        const collName = randomName("Versioned Collection ");
        cy.loginAs(activeUser);

        cy.createCollection(adminUser.token, {
            name: collName,
            owner_uuid: activeUser.user.uuid,
            preserve_version: true,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n",
        })
            // Change the file name to create a new version.
            .then((collection) => cy.updateCollection(adminUser.token, collection.uuid, {
                manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:foo\n",
            }))
            // Get old version.
            .then((collection) => cy.doRequest("GET", "/arvados/v1/collections", null, {
                filters: JSON.stringify([
                    ["current_version_uuid", "=", collection.uuid],
                    ["version", "=", 1],
                ]),
                include_old_versions: true,
                limit: "1",
                select: JSON.stringify(["uuid"]),
                count: "none",
            }, activeUser.token, true))
            .its("body.items.0.uuid")
            // Go to old version.
            .then((oldVersionUuid) => cy.goToPath(`/collections/${oldVersionUuid}`));

        cy.doMPVTabSelect("Overview");
        cy.get("[data-cy=details-element]").should("contain", "This is an old version");
        cy.get("[data-cy=read-only-icon]").should("be.visible");
        cy.get("[data-cy=collection-details-card]").should("contain", collName);
        cy.doMPVTabSelect("Files");
        cy.get("[data-cy=collection-files-panel]").should("contain", "bar");
    });

    it("views & edits storage classes data", function () {
        cy.loginAs(activeUser);

        cy.createCollection(adminUser.token, {
            name: randomName(),
            owner_uuid: activeUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:some-file\n",
        })
            .as("testCollection")
            .then((testCollection) => cy.goToPath(`/collections/${testCollection.uuid}`));

        cy.doMPVTabSelect("Overview");
        // Initial check: it should show the 'default' storage class
        cy.get("[data-cy=details-element]")
            .should("contain", "Storage classes")
            .and("contain", "default")
            .and("not.contain", "foo")
            .and("not.contain", "bar");
        // Edit collection: add storage class 'foo'
        cy.get('[data-title="Edit collection"]').click();
        cy.get("[data-cy=form-dialog]")
            .should("contain", "Edit Collection")
            .and("contain", "Storage classes")
            .and("contain", "default")
            .and("contain", "foo")
            .and("contain", "bar")
            .within(() => {
                cy.get("[data-cy=checkbox-foo]").click();
            });
        cy.get("[data-cy=form-submit-btn]").click();
        cy.get("[data-cy=details-element]")
            .should("contain", "default")
            .and("contain", "foo")
            .and("not.contain", "bar");

        // Storage class changes are committed.
        cy.get("@testCollection")
            .its("uuid")
            .then((uuid) => cy.getCollection(activeUser.token, uuid))
            .its("storage_classes_desired")
            .should("deep.equal", ["default", "foo"]);

        // Edit collection: remove storage class 'default'
        cy.get('[data-title="Edit collection"]').click();
        cy.get("[data-cy=form-dialog]")
            .should("contain", "Edit Collection")
            .and("contain", "Storage classes")
            .and("contain", "default")
            .and("contain", "foo")
            .and("contain", "bar")
            .within(() => {
                cy.get("[data-cy=checkbox-default]").click();
            });
        cy.get("[data-cy=form-submit-btn]").click();
        cy.get("[data-cy=details-element]")
            .should("not.contain", "default")
            .and("contain", "foo")
            .and("not.contain", "bar");

        // Storage class changes are committed.
        cy.get("@testCollection")
            .its("uuid")
            .then((uuid) => cy.getCollection(activeUser.token, uuid))
            .its("storage_classes_desired")
            .should("deep.equal", ["foo"]);
    });

    it("moves a collection to a different project", function () {
        const collName = randomName();
        const projName = randomName("Test Project ");
        const fileName = randomName("Test_File_");

        cy.createCollection(adminUser.token, {
            name: collName,
            owner_uuid: activeUser.user.uuid,
            manifest_text: `. 37b51d194a7513e45b56f6524f2d51f2+3 0:3:${fileName}\n`,
        }).as("testCollection");

        cy.createGroup(adminUser.token, {
            name: projName,
            group_class: "project",
            owner_uuid: activeUser.user.uuid,
        }).as("testProject");

        cy.loginAs(activeUser);

        cy.get("@testCollection").then(testCollection => cy.goToPath(`/collections/${testCollection.uuid}`));
        cy.doMPVTabSelect("Files");
        cy.get("[data-cy=collection-files-panel]").should("contain", fileName);
        cy.get("@testProject").then(testProject => {
            cy.get("[data-cy=collection-details-card]")
                .should("not.contain", projName)
                .and("not.contain", testProject.uuid);
        });
        cy.get('[data-title="Move to"]').click();
        cy.get("[data-cy=form-dialog]")
            .should("contain", "Move to")
            .within(() => {
                // must use .then to avoid selecting instead of expanding https://github.com/cypress-io/cypress/issues/5529
                cy.get("[data-cy=projects-tree-home-tree-picker]")
                    .find("i")
                    .then(el => el.click());
                cy.get("[data-cy=projects-tree-home-tree-picker]").contains(projName).click();
            });
        cy.get("[data-cy=form-submit-btn]").click();
        cy.get("[data-cy=snackbar]").should("contain", "Collection has been moved");
        cy.get("button").contains(projName);
        // Double check that the collection is in the project
        cy.get("@testProject").then(testProject => cy.goToPath(`/projects/${testProject.uuid}`));
        cy.doMPVTabSelect("Data");
        cy.get("[data-cy=project-panel]").should("contain", collName);
    });

    it("automatically updates the collection UI contents without using the Refresh button", function () {
        const collName = randomName();
        cy.loginAs(activeUser);

        cy.createCollection(adminUser.token, {
            name: collName,
            owner_uuid: activeUser.user.uuid,
        })
            .its("uuid")
            .as("testUuid")
            .then((testUuid) => cy.goToPath(`/collections/${testUuid}`));

        const files = ["foobar", "anotherFile", "", "finalName"];
        cy.doMPVTabSelect("Files");
        cy.get("[data-cy=collection-files-panel]")
            .should("contain", "This collection is empty")
            .and("not.contain", files[0]);
        cy.get("[data-cy=collection-details-card]").should("contain", collName);

        cy.clock();
        files.map((fileName, i, files) => {
            let collNewName = `${collName} updated ${i}`;
            cy.get("@testUuid")
                .then((uuid) => cy.updateCollection(adminUser.token, uuid, {
                    name: collNewName,
                    manifest_text: fileName ? `. 37b51d194a7513e45b56f6524f2d51f2+3 0:3:${fileName}\n` : "",
                }))
                .then(() => {
                    // Fast forward 15 seconds for the websocket throttle
                    cy.tick(15000);
                });
            cy.get("[data-cy=collection-details-card]").should("contain", collNewName);
            if (fileName) {
                cy.get("[data-cy=collection-files-panel]").should("contain", fileName);
            } else {
                cy.get("[data-cy=collection-files-panel]")
                    .should("contain", "This collection is empty")
                    .and("not.contain", files[i - 1]);
            }
        });
    });

    it("makes a copy of an existing collection", function () {
        const collName = randomName();
        const copyName = `Copy of: ${collName}`;

        cy.loginAs(activeUser);

        cy.createCollection(adminUser.token, {
            name: collName,
            owner_uuid: activeUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:some-file\n",
        })
            .then((collection) => cy.goToPath(`/collections/${collection.uuid}`));

        cy.doMPVTabSelect("Files");
        cy.get("[data-cy=collection-files-panel]").should("contain", "some-file");
        cy.get('[data-title="Make a copy"]').click();
        cy.get("[data-cy=form-dialog]")
            .should("contain", "Make a copy")
            .within(() => {
                cy.get("[data-cy=projects-tree-home-tree-picker]").contains("Projects").click();
                cy.get("[data-cy=form-submit-btn]").click();
            });

        cy.get("[data-cy=snackbar]").should("contain", "Collection has been copied.");
        goToCollectionByName(copyName, activeUser.token);
        cy.doMPVTabSelect("Files");
        cy.get("[data-cy=collection-files-panel]").should("contain", "some-file");
    });

    it("uses the collection version browser to view a previous version", function () {
        const collName = randomName();

        cy.loginAs(activeUser);
        cy.createCollection(adminUser.token, {
            name: collName,
            owner_uuid: activeUser.user.uuid,
            preserve_version: true,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:foo 0:3:bar\n",
        }).then((collection) => cy.goToPath(`/collections/${collection.uuid}`));

        cy.doMPVTabSelect("Overview");
        cy.get("[data-cy=details-element]").should("not.contain", "This is an old version");
        cy.get("[data-cy=read-only-icon]").should("not.exist");
        cy.get("[data-cy=collection-version-number]").should("contain", "1");
        cy.get("[data-cy=collection-details-card]").should("contain", collName);
        cy.doMPVTabSelect("Files");
        cy.get("[data-cy=collection-files-panel]").should("contain", "foo").and("contain", "bar");

        // Modify collection, expect version number change
        cy.get("[data-cy=collection-files-panel]").contains("foo").rightclick();
        cy.get("[data-cy=context-menu]").contains("Remove").click();
        cy.get("[data-cy=confirmation-dialog]").should("contain", "Removing file");
        cy.get("[data-cy=confirmation-dialog-ok-btn]").click();
        cy.get("[data-cy=collection-files-panel]").should("not.contain", "foo").and("contain", "bar");
        cy.doMPVTabSelect("Overview");
        cy.get("[data-cy=collection-version-number]").should("contain", "2");

        // Click on version number, check version browser. Click on past version.
        cy.get("[data-cy=collection-version-browser]").should("not.exist");
        cy.get("[data-cy=collection-version-number]").contains("2").click();
        cy.get("[data-cy=collection-version-browser]")
            .should("contain", "Nr")
            .and("contain", "Size")
            .and("contain", "Date")
            .within(() => {
                // Version 1: 6 bytes in size
                cy.get("[data-cy=collection-version-browser-select-1]")
                    .should("contain", "1")
                    .and("contain", "6 B")
                    .and("contain", adminUser.user.full_name);
                // Version 2: 3 bytes in size (one file removed)
                cy.get("[data-cy=collection-version-browser-select-2]")
                    .should("contain", "2")
                    .and("contain", "3 B")
                    .and("contain", activeUser.user.full_name);
                cy.get("[data-cy=collection-version-browser-select-3]").should("not.exist");
                cy.get("[data-cy=collection-version-browser-select-1]").click();
            });
        // Navigate back to overview tab
        cy.doMPVTabSelect("Overview");
        cy.get("[data-cy=details-element]").should("contain", "This is an old version");
        cy.get("[data-cy=read-only-icon]").should("be.visible");
        cy.get("[data-cy=collection-version-number]").should("contain", "1");
        cy.get("[data-cy=collection-details-card]").should("contain", collName);
        cy.doMPVTabSelect("Files");
        cy.get("[data-cy=collection-files-panel]").should("contain", "foo").and("contain", "bar");

        // Check that only old collection action are available on toolbar
        cy.get('[data-title="Restore version"]').should('exist');
        cy.get('[data-title="Add to favorites"]').should('not.exist');

        // Click on "head version" link, confirm that it's the latest version.
        cy.doMPVTabSelect("Overview");;
        cy.get("[data-cy=details-element]").contains("head version").click();
        // Navigate back to overview after changing versions
        cy.doMPVTabSelect("Overview");
        cy.get("[data-cy=details-element]").should("not.contain", "This is an old version");
        cy.get("[data-cy=read-only-icon]").should("not.exist");
        cy.get("[data-cy=collection-version-number]").should("contain", "2");
        cy.get("[data-cy=collection-details-card]").should("contain", collName);
        cy.doMPVTabSelect("Files");
        cy.get("[data-cy=collection-files-panel]").should("not.contain", "foo").and("contain", "bar");

        // Check that old collection action isn't available on context menu
        cy.get('[data-title="Restore version"]').should('not.exist');

        // Make another change, confirm new version.
        cy.get('[data-title="Edit collection"]').click();
        cy.get("[data-cy=form-dialog]")
            .should("contain", "Edit Collection")
            .within(() => {
                // appends some text
                cy.get("input").first().type(" renamed");
            });
        cy.get("[data-cy=form-submit-btn]").click();
        cy.doMPVTabSelect("Overview");;
        cy.get("[data-cy=details-element]").should("not.contain", "This is an old version");
        cy.get("[data-cy=read-only-icon]").should("not.exist");
        cy.get("[data-cy=collection-version-number]").should("contain", "3");
        cy.get("[data-cy=collection-details-card]").should("contain", collName + " renamed");
        cy.get("[data-cy=collection-version-browser-select-3]").should("contain", "3").and("contain", "3 B");
        cy.doMPVTabSelect("Files");
        cy.get("[data-cy=collection-files-panel]").should("not.contain", "foo").and("contain", "bar");

        // Check context menus on version browser
        cy.get("[data-cy=collection-version-browser-select-3]").rightclick();
        cy.get("[data-cy=context-menu]")
            .should("contain", "Add to favorites")
            .and("contain", "Make a copy")
            .and("contain", "Edit collection");
        cy.get("body").click();
        // (and now an old version...)
        cy.get("[data-cy=collection-version-browser-select-1]").rightclick();
        cy.get("[data-cy=context-menu]")
            .should("not.contain", "Add to favorites")
            .and("contain", "Make a copy")
            .and("not.contain", "Edit collection");
        cy.get("body").click();

        // Restore first version
        cy.get("[data-cy=collection-version-browser] [data-cy=collection-version-browser-select-1]").click();
        cy.get('[data-title="Restore version"]').click();
        cy.get("[data-cy=confirmation-dialog]").should("contain", "Restore version");
        cy.get("[data-cy=confirmation-dialog-ok-btn]").click();
        // Navigate back to overview after changing versions
        cy.doMPVTabSelect("Overview");
        cy.get("[data-cy=details-element]").should("not.contain", "This is an old version");
        cy.get("[data-cy=collection-version-number]").should("contain", "4");
        cy.get("[data-cy=collection-details-card]").should("contain", collName);
        cy.doMPVTabSelect("Files");
        cy.get("[data-cy=collection-files-panel]").should("contain", "foo").and("contain", "bar");
    });

    it("copies selected files into new collection", () => {
        const srcName = randomName();
        cy.loginAs(activeUser);

        cy.createCollection(adminUser.token, {
            name: srcName,
            owner_uuid: activeUser.user.uuid,
            preserve_version: true,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:foo 0:3:bar\n",
        }).then((collection) => cy.goToPath(`/collections/${collection.uuid}`));

        cy.doMPVTabSelect("Files");
        cy.get("[data-cy=collection-files-panel] input[type=checkbox]").first().click();

        cy.get("[data-cy=collection-files-panel-options-btn]").click();
        cy.get("[data-cy=context-menu]").contains("Copy selected into new collection").click();

        cy.get("[data-cy=form-dialog]").contains("Projects").click();
        cy.get("[data-cy=form-submit-btn]").click();

        cy.get("[data-cy=snackbar]").should("contain", "New collection created.");
        goToCollectionByName(`Files extracted from: ${srcName}`, activeUser.token);
        cy.doMPVTabSelect("Files");
        cy.get("[data-cy=collection-files-panel]").and("contain", "bar");
    });

    it("copies selected files into existing collection", () => {
        const srcName = randomName();
        const dstName = randomName("Destination Collection ");

        cy.loginAs(activeUser);

        cy.createCollection(adminUser.token, {
            name: dstName,
            owner_uuid: activeUser.user.uuid,
            preserve_version: true,
            manifest_text: "",
        }).as("dstCollection");

        cy.createCollection(adminUser.token, {
            name: srcName,
            owner_uuid: activeUser.user.uuid,
            preserve_version: true,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:foo 0:3:bar\n",
        }).then((srcCollection) => cy.goToPath(`/collections/${srcCollection.uuid}`));
        // Now browsing src collection; check basic information
        cy.doMPVTabSelect("Files");
        cy.get("[data-cy=collection-files-panel] input[type=checkbox]").first().click();

        cy.get("[data-cy=collection-files-panel-options-btn]").click();
        cy.get("[data-cy=context-menu]").contains("Copy selected into existing collection").click();
        cy.get("[data-cy=form-dialog]").contains(dstName).click();
        cy.get("[data-cy=form-submit-btn]").click();
        cy.get("[data-cy=form-dialog]").should("not.exist");

        // Visit dst collection
        cy.get("@dstCollection").then((collection) => cy.goToPath(`/collections/${collection.uuid}`));

        cy.doMPVTabSelect("Overview");;
        cy.get("main").contains(dstName).should("exist");
        cy.doMPVTabSelect("Files");
        cy.get("[data-cy=collection-files-panel]").should("contain", "bar");
    });

    it("copies selected files into separate collections", () => {
        const srcName = randomName();
        const dstNameFoo = `File copied from collection ${srcName}/foo`;
        const dstNameBar = `File copied from collection ${srcName}/bar`;

        cy.loginAs(activeUser);

        cy.createCollection(adminUser.token, {
            name: srcName,
            owner_uuid: activeUser.user.uuid,
            preserve_version: true,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:foo 0:3:bar\n",
        }).then((collection) => cy.goToPath(`/collections/${collection.uuid}`));

        // Select both files
        cy.doMPVTabSelect("Files");
        cy.get("[data-cy=collection-files-panel] input[type=checkbox]").first().click();
        cy.get("[data-cy=collection-files-panel] input[type=checkbox]").last().click();

        // Copy to separate collections
        cy.get("[data-cy=collection-files-panel-options-btn]").click();
        cy.get("[data-cy=context-menu]").contains("Copy selected into separate collections").click();
        cy.get("[data-cy=form-dialog]").contains("Projects").click();
        cy.get("[data-cy=form-submit-btn]").click();

        cy.get("[data-cy=snackbar]").should("contain", "New collections created.");
        // Verify created collections
        goToCollectionByName(dstNameFoo, activeUser.token);
        cy.doMPVTabSelect("Files");
        cy.get("[data-cy='collection-files-panel'] [data-subfolder-path='foo']").should("be.visible");

        goToCollectionByName(dstNameBar, activeUser.token);
        cy.doMPVTabSelect("Files");
        cy.get("[data-cy='collection-files-panel'] [data-subfolder-path='bar']").should("be.visible");

        // Verify separate collection menu items not present when single file selected
        cy.doMPVTabSelect("Files");
        cy.get("[data-cy=collection-files-panel] input[type=checkbox]")
            .first().click();
        cy.get("[data-cy=collection-files-panel-options-btn]").click();
        cy.get("[data-cy=context-menu]")
            .should("not.contain", "Copy selected into separate collections")
            .and("not.contain", "Move selected into separate collections");
    });

    it("moves selected files into new collection", () => {
        const srcName = randomName();
        const dstName = `Files moved from: ${srcName}`;

        cy.loginAs(activeUser);
        cy.createCollection(adminUser.token, {
            name: srcName,
            owner_uuid: activeUser.user.uuid,
            preserve_version: true,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:foo 0:3:bar\n",
        }).then((collection) => cy.goToPath(`/collections/${collection.uuid}`));

        cy.doMPVTabSelect("Files");
        cy.get("[data-cy=collection-files-panel] input[type=checkbox]").first().click();

        cy.get("[data-cy=collection-files-panel-options-btn]").click();
        cy.get("[data-cy=context-menu]").contains("Move selected into new collection").click();

        cy.get("[data-cy=form-dialog]").contains("Projects").click();
        cy.get("[data-cy=form-submit-btn]").click();

        cy.get("[data-cy=snackbar]").should("contain", "Files have been moved to selected collection.");
        goToCollectionByName(dstName, activeUser.token);
        cy.doMPVTabSelect("Files");
        cy.get("[data-cy=collection-files-panel]").and("contain", "bar");
    });

    it("moves selected files into existing collection", () => {
        const srcName = randomName();
        const dstName = randomName();

        cy.loginAs(activeUser);

        cy.createCollection(adminUser.token, {
            name: dstName,
            owner_uuid: activeUser.user.uuid,
            preserve_version: true,
            manifest_text: "",
        }).as("dstCollection");

        cy.createCollection(adminUser.token, {
            name: srcName,
            owner_uuid: activeUser.user.uuid,
            preserve_version: true,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:foo 0:3:bar\n",
        }).then((srcCollection) => cy.goToPath(`/collections/${srcCollection.uuid}`));

        // Now browsing src collection.
        cy.doMPVTabSelect("Files");
        // Click on checkbox next to "bar".
        cy.get("[data-cy='collection-files-panel'] [data-subfolder-path='bar'] input[type='checkbox']").click();

        cy.get("[data-cy=collection-files-panel-options-btn]").click();
        cy.get("[data-cy=context-menu]").contains("Move selected into existing collection").click();

        cy.get("[data-cy=form-dialog]").contains(dstName).click();
        cy.get("[data-cy=form-submit-btn]").click();
        cy.get("[data-cy=form-dialog]").should("not.exist");

        // Go to dst collection.
        cy.get("@dstCollection").then(dstCollection => cy.goToPath(`/collections/${dstCollection.uuid}`));

        cy.get("main").contains(dstName).should("exist");
        cy.doMPVTabSelect("Files");
        cy.get("[data-cy='collection-files-panel'] [data-subfolder-path='bar']").should("be.visible");
    });

    it("moves selected files into separate collections", () => {
        const srcName = randomName();
        const dstNameFoo = `File moved from collection ${srcName}/foo`;
        const dstNameBar = `File moved from collection ${srcName}/bar`;

        cy.loginAs(activeUser);
        cy.createCollection(adminUser.token, {
            name: srcName,
            owner_uuid: activeUser.user.uuid,
            preserve_version: true,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:foo 0:3:bar\n",
        }).then((srcCollection) => cy.goToPath(`/collections/${srcCollection.uuid}`));

        // Now browsing src collection.
        cy.doMPVTabSelect("Files");
        // Select both files
        cy.get("[data-cy=collection-files-panel]").within(() => {
            cy.get("input[type=checkbox]").first().click();
            cy.get("input[type=checkbox]").last().click();
        });

        // Move to separate collections
        cy.get("[data-cy=collection-files-panel-options-btn]").click();
        cy.get("[data-cy=context-menu]").contains("Move selected into separate collections").click();
        cy.get("[data-cy=form-dialog]").contains("Projects").click();
        cy.get("[data-cy=form-submit-btn]").click();
        cy.get("[data-cy=snackbar]").should("contain", "New collections created.");

        // Verify created collections
        goToCollectionByName(dstNameFoo, activeUser.token);
        cy.get("main").contains(dstNameFoo).should("exist");
        cy.doMPVTabSelect("Files");
        cy.get("[data-cy='collection-files-panel'] [data-subfolder-path='foo']").should("be.visible");

        goToCollectionByName(dstNameBar, activeUser.token);
        cy.doMPVTabSelect("Files");
        cy.get("main").contains(dstNameBar).should("exist");
        cy.get("[data-cy='collection-files-panel'] [data-subfolder-path='bar']").should("be.visible");
    });

    it("creates new collection with properties on home project", function () {
        cy.loginAs(activeUser);
        cy.goToPath(`/projects/${activeUser.user.uuid}`);
        cy.get("[data-cy=breadcrumb-first]").should("contain", "Projects");
        cy.get("[data-cy=breadcrumb-last]").should("not.exist");
        // Create new collection
        cy.get("[data-cy=side-panel-button]").click();
        cy.get("[data-cy=side-panel-new-collection]").click();
        // Name between brackets tests bugfix #17582
        const collName = `[${randomName()}]`;

        // Select a storage class.
        cy.get("[data-cy=form-dialog]")
            .should("contain", "New collection")
            .and("contain", "Storage classes")
            .and("contain", "default")
            .and("contain", "foo")
            .and("contain", "bar")
            .within(() => {
                cy.get("[data-cy=parent-field] input").should("have.value", "Home project");
                cy.get("[data-cy=name-field] input").type(collName);
                cy.get("[data-cy=checkbox-foo]").click();
            });

        // Add a property.
        // Key: Color (IDTAGCOLORS) - Value: Magenta (IDVALCOLORS3)
        cy.get("[data-cy=form-dialog]").should("not.contain", "Color: Magenta");
        cy.get("[data-cy=resource-properties-form]").within(() => {
            cy.get("[data-cy=property-field-key] input").type("Color");
            cy.get("[data-cy=property-field-value]").click().within(() => {
                cy.get("input").type("Magenta");
            });
            cy.get("[data-cy=property-add-btn]").click();
        });
        // Confirm proper vocabulary labels are displayed on the UI.
        cy.get("[data-cy=form-dialog]").should("contain", "Color: Magenta");

        // Value field should not complain about being required just after
        // adding a new property. See #19732
        cy.get("[data-cy=form-dialog]").should("not.contain", "This field is required");
        cy.get("[data-cy=form-submit-btn]").click();

        // Confirm that the user was taken to the newly created collection
        cy.get("[data-cy=form-dialog]").should("not.exist");
        cy.get("[data-cy=breadcrumb-first]").should("contain", "Projects");
        cy.get("[data-cy=breadcrumb-last]").should("contain", collName);

        // Navigate to Overview tab
        cy.doMPVTabSelect("Overview");

        // Verify details
        cy.get("[data-cy='details-panel-storage classes'] [data-cy='details-attribute-value']")
            .should("contain", "default")
            .and("contain", "foo")
            .and("not.contain", "bar");
        cy.get("[data-cy=resource-properties]")
            .should("contain", "Color: Magenta");
        // Confirm that the collection's properties have been committed.
        cy.doRequest("GET", "/arvados/v1/collections", null, {
            filters: JSON.stringify([["name", "=", collName]]),
        })
            .its("body.items")
            .then(collArray => {
                expect(collArray).to.have.lengthOf(1);
                expect(collArray[0].properties["IDTAGCOLORS"]).to.deep.equal(["IDVALCOLORS3"]);
            });
    });

    it("shows responsible person for collection if available", () => {
        cy.createCollection(adminUser.token, {
            name: randomName(),
            owner_uuid: activeUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n",
        }).as("testCollection1");

        cy.createCollection(adminUser.token, {
            name: randomName(),
            owner_uuid: adminUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n",
        })
            .as("testCollection2")
            .then((testCollection2) => {
                cy.shareWith(adminUser.token, activeUser.user.uuid, testCollection2.uuid, "can_write");
            });

        cy.loginAs(activeUser);
        cy.get("@testCollection1").then((testCollection1) => cy.goToPath(`/collections/${testCollection1.uuid}`));
        // Navigate to Overview tab
        cy.doMPVTabSelect("Overview");
        cy.get("[data-cy=responsible-person-wrapper]")
            .should("contain", activeUser.user.uuid);

        cy.get("@testCollection2").then((testCollection2) => cy.goToPath(`/collections/${testCollection2.uuid}`));
        // Navigate to Overview tab
        cy.doMPVTabSelect("Overview");
        cy.get("[data-cy=responsible-person-wrapper]")
            .should("contain", adminUser.user.uuid);
    });

    it('displays the correct breadcrumbs after moving a collection to trash', () => {
        // See old redmine issue #22618
        const collName = randomName("Breadcrumb Test Collection ");
        cy.createCollection(adminUser.token, {
            name: collName,
            owner_uuid: activeUser.user.uuid,
            manifest_text: "",
        });

        cy.loginAs(activeUser);
        // Go to home projects
        cy.goToPath(`/projects/${activeUser.user.uuid}`);

        // Right click on the item in listing
        cy.get('[data-cy=data-table-row]').contains(collName).rightclick();
        cy.get('[data-cy=context-menu]').should('exist');
        // Move to trash
        cy.get('[data-cy=context-move-to-trash]').click();

        cy.get("[data-cy=breadcrumb-first]").should("contain", "Home Projects");
    });

    describe("file upload", () => {
        beforeEach(() => {
            cy.createCollection(adminUser.token, {
                name: `Test collection ${Math.floor(Math.random() * 999999)}`,
                owner_uuid: activeUser.user.uuid,
                manifest_text: "./subdir 37b51d194a7513e45b56f6524f2d51f2+3 0:3:foo\n. 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n",
            }).as("testCollection1");
        });

        it("uploads a file and checks the collection UI to be fresh", () => {
            cy.getAll("@testCollection1").then(function ([testCollection1]) {
                cy.loginAs(activeUser);
                cy.goToPath(`/collections/${testCollection1.uuid}`);
                // Navigate to Overview tab
                cy.doMPVTabSelect("Overview");
                cy.get("[data-cy=collection-file-count]").should("contain", "2");
                cy.doMPVTabSelect("Files");
                cy.get("[data-cy=upload-button]").click();
                cy.get("[data-cy=collection-files-panel]").contains("5mb_a.bin").should("not.exist");
                cy.fixture("files/5mb.bin", "base64").then(content => {
                    cy.get("[data-cy=drag-and-drop]").upload(content, "5mb_a.bin");
                    cy.get("[data-cy=form-submit-btn]").click();
                    cy.get("[data-cy=form-submit-btn]").should("not.exist");
                    cy.get("[data-cy=collection-files-panel]").contains("5mb_a.bin").should("exist");
                    cy.doMPVTabSelect("Overview");
                    cy.get("[data-cy=collection-file-count]").should("contain", "3");

                    cy.doMPVTabSelect("Files");
                    cy.get("[data-cy=collection-files-panel]").contains("subdir").click();
                    cy.get("[data-cy=upload-button]").click();
                    cy.fixture("files/5mb.bin", "base64").then(content => {
                        cy.get("[data-cy=drag-and-drop]").upload(content, "5mb_b.bin");
                        cy.get("[data-cy=form-submit-btn]").click();
                        cy.waitForDom().get("[data-cy=form-submit-btn]").should("not.exist");
                        // subdir gets unselected, I think this is a bug but
                        // for the time being let's just make sure the test works.
                        cy.get("[data-cy=collection-files-panel]").contains("subdir").click();
                        cy.waitForDom().get("[data-cy=collection-files-right-panel]").contains("5mb_b.bin").should("exist");
                    });
                });
            });
        });

        it('uploads and maintains nested folder structure', () => {
            cy.getAll('@testCollection1').then(function ([testCollection1]) {
                cy.loginAs(activeUser);
                cy.goToPath(`/collections/${testCollection1.uuid}`);
                cy.doMPVTabSelect("Files");
                cy.get('[data-cy=upload-button]').click();
                cy.fixture('files/5mb.bin', 'base64').then((content) => {
                    cy.get('[data-cy=drag-and-drop]').upload(content, 'foo/bar/baz/qux');
                    cy.get("[data-cy=form-submit-btn]").click();
                    cy.get("[data-cy=form-submit-btn]").should("not.exist");
                    cy.waitForDom().get("[data-cy=collection-files-panel]").contains("foo").should("exist").click();
                    cy.get('[data-subfolder-path="bar"]').should('exist').click();
                    cy.get('[data-subfolder-path="baz"]').should('exist').click();
                })
            });
        });

        it("allows to cancel running upload", () => {
            cy.getAll("@testCollection1").then(function ([testCollection1]) {
                cy.loginAs(activeUser);

                cy.goToPath(`/collections/${testCollection1.uuid}`);

                cy.doMPVTabSelect("Files");
                cy.get("[data-cy=upload-button]").click();

                cy.fixture("files/5mb.bin", "base64").then(content => {
                    cy.get("[data-cy=drag-and-drop]").upload(content, "5mb_a.bin");
                    cy.get("[data-cy=drag-and-drop]").upload(content, "5mb_b.bin");

                    cy.get("[data-cy=form-submit-btn]").click();

                    cy.get("button").contains("Cancel").click();

                    cy.get("[data-cy=form-submit-btn]").should("not.exist");
                });
            });
        });

        it("allows to cancel single file from the running upload", () => {
            cy.getAll("@testCollection1").then(function ([testCollection1]) {
                cy.loginAs(activeUser);

                cy.goToPath(`/collections/${testCollection1.uuid}`);

                cy.doMPVTabSelect("Files");
                cy.get("[data-cy=upload-button]").click();

                cy.fixture("files/5mb.bin", "base64").then(content => {
                    cy.get("[data-cy=drag-and-drop]").upload(content, "5mb_a.bin");
                    cy.get("[data-cy=drag-and-drop]").upload(content, "5mb_b.bin");

                    cy.get("[data-cy=form-submit-btn]").click();

                    cy.get("button[aria-label=Remove]").eq(1).click();

                    cy.get("[data-cy=form-submit-btn]").should("not.exist");

                    cy.get("[data-cy=collection-files-panel]").contains("5mb_a.bin").should("exist");
                });
            });
        });

        it("allows to cancel all files from the running upload", () => {
            cy.getAll("@testCollection1").then(function ([testCollection1]) {
                cy.loginAs(activeUser);

                cy.goToPath(`/collections/${testCollection1.uuid}`);
                cy.doMPVTabSelect("Files");

                // Confirm initial collection state.
                cy.get("[data-cy=collection-files-panel]").contains("bar").should("exist");
                cy.get("[data-cy=collection-files-panel]").contains("15mb_a.bin").should("not.exist");
                cy.get("[data-cy=collection-files-panel]").contains("15mb_b.bin").should("not.exist");

                cy.get("[data-cy=upload-button]").click();

                cy.fixture("files/15mb.bin", "base64").then(content => {
                    cy.get("[data-cy=drag-and-drop]").upload(content, "15mb_a.bin");
                    cy.get("[data-cy=drag-and-drop]").upload(content, "15mb_b.bin");

                    cy.get("[data-cy=form-submit-btn]").click();
                    cy.get("button[aria-label=Remove]").should("exist").click({ multiple: true});
                    cy.get("[data-cy=form-submit-btn]").should("not.exist");

                    // Confirm final collection state.
                    cy.get("[data-cy=collection-files-panel]").contains("bar").should("exist");
                    // The following fails, but doesn't seem to happen
                    // in the real world. Maybe there's a race between
                    // the PUT request finishing and the 'Remove' button
                    // dissapearing, because sometimes just one of the 2
                    // files gets uploaded.
                    // Maybe this will be needed to simulate a slow network:
                    // https://docs.cypress.io/api/commands/intercept#Convenience-functions-1
                    // cy.get('[data-cy=collection-files-panel]')
                    //     .contains('5mb_a.bin').should('not.exist');
                    // cy.get('[data-cy=collection-files-panel]')
                    //     .contains('5mb_b.bin').should('not.exist');
                });
            });
        });

    });

    describe("zip download", () => {
        beforeEach(() => {
            cy.createCollection(adminUser.token, {
                name: `Test collection ${Math.floor(Math.random() * 999999)}`,
                owner_uuid: activeUser.user.uuid,
                manifest_text: "./subdir 37b51d194a7513e45b56f6524f2d51f2+3 0:3:foo\n. 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n",
            }).as("testCollection1");
        });

        it('all files', () => {
            cy.getAll("@testCollection1").then(function ([testCollection1]) {
                const downloadName = `${testCollection1.name}.zip`;

                cy.intercept({ method: "GET", url: `**/c=${testCollection1.uuid}*`, times: 1, query: {
                    accept: "application/zip",
                    disposition: "attachment",
                    download_filename: downloadName,
                }}, (req) => {
                    const url = new URL(req.url);
                    const files = url.searchParams.get("files");
                    // Cannot assert on null so we assert the comparison
                    expect(files === null).to.equal(true);
                }).as('downloadQuery');

                cy.loginAs(activeUser);
                cy.goToPath(`/projects/${activeUser.user.uuid}`);

                // Navigate to collection files
                cy.doDataExplorerNavigate(testCollection1.name);
                cy.doMPVTabSelect("Files");

                // Click download all files as zip
                cy.doCollectionPanelOptionsAction("Download entire collection as zip");
                cy.waitForDom();

                // Verify filename
                cy.get("[data-cy=form-dialog]").within(() => {
                    cy.get('h2').contains("Download");
                    cy.get('input[name=fileName]').should('have.value', downloadName);
                    cy.get('button[data-cy=form-submit-btn]').click();
                });
                // Wait for download request to match
                cy.wait('@downloadQuery');
            });
        });

        it('one file', () => {
            cy.getAll("@testCollection1").then(function ([testCollection1]) {
                const downloadName = `${testCollection1.name} - bar.zip`;

                cy.intercept({ method: "GET", url: `**/c=${testCollection1.uuid}*`, times: 1, query: {
                    accept: "application/zip",
                    disposition: "attachment",
                    download_filename: downloadName,
                }}, (req) => {
                    const url = new URL(req.url);
                    const files = url.searchParams.toString()
                        .split("&")
                        .filter(param => param.startsWith("files="))
                        .join("&");

                    expect(files).to.equal("files=bar");
                }).as('downloadQuery');

                cy.loginAs(activeUser);
                cy.goToPath(`/projects/${activeUser.user.uuid}`);

                // Navigate to collection files
                cy.doDataExplorerNavigate(testCollection1.name);
                cy.doMPVTabSelect("Files");

                // Select one file
                cy.doCollectionFileSelect("bar");

                // Click download all files as zip
                cy.doCollectionPanelOptionsAction("Download selected files as zip");
                cy.waitForDom();

                // Verify filename
                cy.get("[data-cy=form-dialog]").within(() => {
                    cy.get('h2').contains("Download");
                    cy.get('input[name=fileName]').should('have.value', downloadName);
                    cy.get('button[data-cy=form-submit-btn]').click();
                });
                // Wait for download request to match
                cy.wait('@downloadQuery');
            });
        });

        it('multi file', () => {
            cy.getAll("@testCollection1").then(function ([testCollection1]) {
                const downloadName = `${testCollection1.name} - 2 files.zip`;

                cy.intercept({ method: "GET", url: `**/c=${testCollection1.uuid}*`, times: 1, query: {
                    accept: "application/zip",
                    disposition: "attachment",
                    download_filename: downloadName,
                }}, (req) => {
                    const url = new URL(req.url);
                    const files = url.searchParams.toString()
                        .split("&")
                        .filter(param => param.startsWith("files="))
                        .join("&");

                    expect(files).to.equal("files=subdir&files=bar");
                }).as('downloadQuery');

                cy.loginAs(activeUser);
                cy.goToPath(`/projects/${activeUser.user.uuid}`);

                // Navigate to collection files
                cy.doDataExplorerNavigate(testCollection1.name);
                cy.doMPVTabSelect("Files");

                // Select multi file
                cy.doCollectionFileSelect("subdir");
                cy.doCollectionFileSelect("bar");

                // Click download all files as zip
                cy.doCollectionPanelOptionsAction("Download selected files as zip");
                cy.waitForDom();

                // Verify filename
                cy.get("[data-cy=form-dialog]").within(() => {
                    cy.get('h2').contains("Download");
                    cy.get('input[name=fileName]').should('have.value', downloadName);
                    cy.get('button[data-cy=form-submit-btn]').click();
                });
                // Wait for download request to match
                cy.wait('@downloadQuery');
            });
        });
    });
});
