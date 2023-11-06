// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

const path = require("path");

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

    beforeEach(function () {
        cy.clearCookies();
        cy.clearLocalStorage();
    });

    it("allows to download mountain duck config for a collection", () => {
        cy.createCollection(adminUser.token, {
            name: `Test collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: activeUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n",
        })
            .as("testCollection")
            .then(function (testCollection) {
                cy.loginAs(activeUser);
                cy.goToPath(`/collections/${testCollection.uuid}`);

                cy.get("[data-cy=collection-panel-options-btn]").click();
                cy.get("[data-cy=context-menu]").contains("Open with 3rd party client").click();
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
        const name = `Test collection ${Math.floor(Math.random() * 999999)}`;
        cy.createCollection(adminUser.token, {
            name: name,
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
                cy.get("[data-cy=name-field]").within(() => {
                    cy.get("input").type(name);
                });
                cy.get("[data-cy=form-submit-btn]").click();
            });
        // Error message should display, allowing editing the name
        cy.get("[data-cy=form-dialog]")
            .should("exist")
            .and("contain", "Collection with the same name already exists")
            .within(() => {
                cy.get("[data-cy=name-field]").within(() => {
                    cy.get("input").type(" renamed");
                });
                cy.get("[data-cy=form-submit-btn]").click();
            });
        cy.get("[data-cy=form-dialog]").should("not.exist");
        // Attempt to rename the collection with the duplicate name
        cy.get("[data-cy=collection-panel-options-btn]").click();
        cy.get("[data-cy=context-menu]").contains("Edit collection").click();
        cy.get("[data-cy=form-dialog]")
            .should("contain", "Edit Collection")
            .within(() => {
                cy.get("[data-cy=name-field]").within(() => {
                    cy.get("input").type("{selectall}{backspace}").type(name);
                });
                cy.get("[data-cy=form-submit-btn]").click();
            });
        cy.get("[data-cy=form-dialog]").should("exist").and("contain", "Collection with the same name already exists");
    });

    it("uses the property editor (from edit dialog) with vocabulary terms", function () {
        cy.createCollection(adminUser.token, {
            name: `Test collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: activeUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n",
        })
            .as("testCollection")
            .then(function () {
                cy.loginAs(activeUser);
                cy.goToPath(`/collections/${this.testCollection.uuid}`);

                cy.get("[data-cy=collection-info-panel").should("contain", this.testCollection.name).and("not.contain", "Color: Magenta");

                cy.get("[data-cy=collection-panel-options-btn]").click();
                cy.get("[data-cy=context-menu]").contains("Edit collection").click();
                cy.get("[data-cy=form-dialog]").should("contain", "Properties");

                // Key: Color (IDTAGCOLORS) - Value: Magenta (IDVALCOLORS3)
                cy.get("[data-cy=resource-properties-form]").within(() => {
                    cy.get("[data-cy=property-field-key]").within(() => {
                        cy.get("input").type("Color");
                    });
                    cy.get("[data-cy=property-field-value]").within(() => {
                        cy.get("input").type("Magenta");
                    });
                    cy.root().submit();
                });
                // Confirm proper vocabulary labels are displayed on the UI.
                cy.get("[data-cy=form-dialog]").should("contain", "Color: Magenta");
                cy.get("[data-cy=form-dialog]").contains("Save").click();
                cy.get("[data-cy=form-dialog]").should("not.exist");
                // Confirm proper vocabulary IDs were saved on the backend.
                cy.doRequest("GET", `/arvados/v1/collections/${this.testCollection.uuid}`)
                    .its("body")
                    .as("collection")
                    .then(function () {
                        expect(this.collection.properties.IDTAGCOLORS).to.equal("IDVALCOLORS3");
                    });
                // Confirm the property is displayed on the UI.
                cy.get("[data-cy=collection-info-panel").should("contain", this.testCollection.name).and("contain", "Color: Magenta");
            });
    });

    it("uses the editor (from details panel) with vocabulary terms", function () {
        cy.createCollection(adminUser.token, {
            name: `Test collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: activeUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n",
        })
            .as("testCollection")
            .then(function () {
                cy.loginAs(activeUser);
                cy.goToPath(`/collections/${this.testCollection.uuid}`);

                cy.get("[data-cy=collection-info-panel")
                    .should("contain", this.testCollection.name)
                    .and("not.contain", "Color: Magenta")
                    .and("not.contain", "Size: S");
                cy.get("[data-cy=additional-info-icon]").click();

                cy.get("[data-cy=details-panel]").within(() => {
                    cy.get("[data-cy=details-panel-edit-btn]").click();
                });
                cy.get("[data-cy=form-dialog").contains("Edit Collection");

                // Key: Color (IDTAGCOLORS) - Value: Magenta (IDVALCOLORS3)
                cy.get("[data-cy=resource-properties-form]").within(() => {
                    cy.get("[data-cy=property-field-key]").within(() => {
                        cy.get("input").type("Color");
                    });
                    cy.get("[data-cy=property-field-value]").within(() => {
                        cy.get("input").type("Magenta");
                    });
                    cy.root().submit();
                });
                // Confirm proper vocabulary labels are displayed on the UI.
                cy.get("[data-cy=form-dialog]").should("contain", "Color: Magenta");

                // Case-insensitive on-blur auto-selection test
                // Key: Size (IDTAGSIZES) - Value: Small (IDVALSIZES2)
                cy.get("[data-cy=resource-properties-form]").within(() => {
                    cy.get("[data-cy=property-field-key]").within(() => {
                        cy.get("input").type("sIzE");
                    });
                    cy.get("[data-cy=property-field-value]").within(() => {
                        cy.get("input").type("sMaLL");
                    });
                    // Cannot "type()" TAB on Cypress so let's click another field
                    // to trigger the onBlur event.
                    cy.get("[data-cy=property-field-key]").click();
                    cy.root().submit();
                });
                // Confirm proper vocabulary labels are displayed on the UI.
                cy.get("[data-cy=form-dialog]").should("contain", "Size: S");

                cy.get("[data-cy=form-dialog]").contains("Save").click();
                cy.get("[data-cy=form-dialog]").should("not.exist");

                // Confirm proper vocabulary IDs were saved on the backend.
                cy.doRequest("GET", `/arvados/v1/collections/${this.testCollection.uuid}`)
                    .its("body")
                    .as("collection")
                    .then(function () {
                        expect(this.collection.properties.IDTAGCOLORS).to.equal("IDVALCOLORS3");
                        expect(this.collection.properties.IDTAGSIZES).to.equal("IDVALSIZES2");
                    });

                // Confirm properties display on the UI.
                cy.get("[data-cy=collection-info-panel")
                    .should("contain", this.testCollection.name)
                    .and("contain", "Color: Magenta")
                    .and("contain", "Size: S");
            });
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
                .as("sharedGroup")
                .then(function () {
                    // Creates the collection using the admin token so we can set up
                    // a bogus manifest text without block signatures.
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
                        .createCollection(adminUser.token, {
                            name: "Test collection",
                            owner_uuid: this.sharedGroup.uuid,
                            properties: { someKey: "someValue" },
                            manifest_text: `. 37b51d194a7513e45b56f6524f2d51f2+3 0:3:${fileName}\n./${subDirName} 37b51d194a7513e45b56f6524f2d51f2+3 0:3:${fileName}\n`,
                        })
                        .as("testCollection")
                        .then(function () {
                            // Share the group with active user.
                            cy.createLink(adminUser.token, {
                                name: isWritable ? "can_write" : "can_read",
                                link_class: "permission",
                                head_uuid: this.sharedGroup.uuid,
                                tail_uuid: activeUser.user.uuid,
                            });
                            cy.goToPath(`/collections/${this.testCollection.uuid}`);

                            // Check that name & uuid are correct.
                            cy.get("[data-cy=collection-info-panel]")
                                .should("contain", this.testCollection.name)
                                .and("contain", this.testCollection.uuid)
                                .and("not.contain", "This is an old version");
                            // Check for the read-only icon
                            cy.get("[data-cy=read-only-icon]").should(`${isWritable ? "not." : ""}exist`);
                            // Check that both read and write operations are available on
                            // the 'More options' menu.
                            cy.get("[data-cy=collection-panel-options-btn]").click();
                            cy.get("[data-cy=context-menu]")
                                .should("contain", "Add to favorites")
                                .and(`${isWritable ? "" : "not."}contain`, "Edit collection");
                            cy.get("body").click(); // Collapse the menu avoiding details panel expansion
                            cy.get("[data-cy=collection-info-panel]")
                                .should("contain", "someKey: someValue")
                                .and("not.contain", "anotherKey: anotherValue");
                            // Check that the file listing show both read & write operations
                            cy.waitForDom()
                                .get("[data-cy=collection-files-panel]")
                                .within(() => {
                                    cy.get("[data-cy=collection-files-right-panel]", { timeout: 5000 }).should("contain", fileName);
                                    if (isWritable) {
                                        cy.get("[data-cy=upload-button]").should(`${isWritable ? "" : "not."}contain`, "Upload data");
                                    }
                                });
                            // Test context menus
                            cy.get("[data-cy=collection-files-panel]").contains(fileName).rightclick();
                            cy.get("[data-cy=context-menu]")
                                .should("contain", "Download")
                                .and("contain", "Open in new tab")
                                .and("contain", "Copy to clipboard")
                                .and(`${isWritable ? "" : "not."}contain`, "Rename")
                                .and(`${isWritable ? "" : "not."}contain`, "Remove");
                            cy.get("body").click(); // Collapse the menu
                            cy.get("[data-cy=collection-files-panel]").contains(subDirName).rightclick();
                            cy.get("[data-cy=context-menu]")
                                .should("not.contain", "Download")
                                .and("contain", "Open in new tab")
                                .and("contain", "Copy to clipboard")
                                .and(`${isWritable ? "" : "not."}contain`, "Rename")
                                .and(`${isWritable ? "" : "not."}contain`, "Remove");
                            cy.get("body").click(); // Collapse the menu
                            // File/dir item 'more options' button
                            cy.get("[data-cy=file-item-options-btn").first().click();
                            cy.get("[data-cy=context-menu]").should(`${isWritable ? "" : "not."}contain`, "Remove");
                            cy.get("body").click(); // Collapse the menu
                            // Hamburger 'more options' menu button
                            cy.get("[data-cy=collection-files-panel-options-btn]").click();
                            cy.get("[data-cy=context-menu]").should("contain", "Select all").click();
                            cy.get("[data-cy=collection-files-panel-options-btn]").click();
                            cy.get("[data-cy=context-menu]").should(`${isWritable ? "" : "not."}contain`, "Remove selected");
                            cy.get("body").click(); // Collapse the menu
                        });
                });
        });
    });

    it("renames a file using valid names", function () {
        function eachPair(lst, func) {
            for (var i = 0; i < lst.length - 1; i++) {
                func(lst[i], lst[i + 1]);
            }
        }
        // Creates the collection using the admin token so we can set up
        // a bogus manifest text without block signatures.
        cy.createCollection(adminUser.token, {
            name: `Test collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: activeUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n",
        })
            .as("testCollection")
            .then(function () {
                cy.loginAs(activeUser);
                cy.goToPath(`/collections/${this.testCollection.uuid}`);

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
                eachPair(names, (from, to) => {
                    cy.waitForDom().get("[data-cy=collection-files-panel]").contains(`${from}`).rightclick();
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
    });

    it("renames a file to a different directory", function () {
        // Creates the collection using the admin token so we can set up
        // a bogus manifest text without block signatures.
        cy.createCollection(adminUser.token, {
            name: `Test collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: activeUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n",
        })
            .as("testCollection")
            .then(function () {
                cy.loginAs(activeUser);
                cy.goToPath(`/collections/${this.testCollection.uuid}`);

                ["subdir", "G%C3%BCnter's%20file", "table%&?*2"].forEach(subdir => {
                    cy.waitForDom().get("[data-cy=collection-files-panel]").contains("bar").rightclick();
                    cy.get("[data-cy=context-menu]").contains("Rename").click();
                    cy.get("[data-cy=form-dialog]")
                        .should("contain", "Rename")
                        .within(() => {
                            cy.get("input").type(`{selectall}{backspace}${subdir}/foo`);
                        });
                    cy.get("[data-cy=form-submit-btn]").click();
                    cy.get("[data-cy=collection-files-panel]").should("not.contain", "bar").and("contain", subdir);
                    cy.get("[data-cy=collection-files-panel]").contains(subdir).click();

                    // Rename 'subdir/foo' to 'bar'
                    cy.wait(1000);
                    cy.get("[data-cy=collection-files-panel]").contains("foo").rightclick();
                    cy.get("[data-cy=context-menu]").contains("Rename").click();
                    cy.get("[data-cy=form-dialog]")
                        .should("contain", "Rename")
                        .within(() => {
                            cy.get("input").should("have.value", `${subdir}/foo`).type(`{selectall}{backspace}bar`);
                        });
                    cy.get("[data-cy=form-submit-btn]").click();

                    // need to wait for dialog to dismiss
                    cy.get("[data-cy=form-dialog]").should("not.exist");

                    cy.waitForDom().get("[data-cy=collection-files-panel]").contains("Home").click();

                    cy.wait(2000);
                    cy.get("[data-cy=collection-files-panel]")
                        .should("contain", subdir) // empty dir kept
                        .and("contain", "bar");

                    cy.get("[data-cy=collection-files-panel]").contains(subdir).rightclick();
                    cy.get("[data-cy=context-menu]").contains("Remove").click();
                    cy.get("[data-cy=confirmation-dialog-ok-btn]").click();
                    cy.get("[data-cy=form-dialog]").should("not.exist");
                });
            });
    });

    it("shows collection owner", () => {
        cy.createCollection(adminUser.token, {
            name: `Test collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: activeUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n",
        })
            .as("testCollection")
            .then(testCollection => {
                cy.loginAs(activeUser);
                cy.goToPath(`/collections/${testCollection.uuid}`);
                cy.wait(5000);
                cy.get("[data-cy=collection-info-panel]").contains(`Collection User`);
            });
    });

    it("tries to rename a file with illegal names", function () {
        // Creates the collection using the admin token so we can set up
        // a bogus manifest text without block signatures.
        cy.createCollection(adminUser.token, {
            name: `Test collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: activeUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n",
        })
            .as("testCollection")
            .then(function () {
                cy.loginAs(activeUser);
                cy.goToPath(`/collections/${this.testCollection.uuid}`);

                const illegalNamesFromUI = [
                    [".", "Name cannot be '.' or '..'"],
                    ["..", "Name cannot be '.' or '..'"],
                    ["", "This field is required"],
                    [" ", "Leading/trailing whitespaces not allowed"],
                    [" foo", "Leading/trailing whitespaces not allowed"],
                    ["foo ", "Leading/trailing whitespaces not allowed"],
                    ["//foo", "Empty dir name not allowed"],
                ];
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
                            cy.contains(`${errMsg}`);
                        });
                    cy.get("[data-cy=form-cancel-btn]").click();
                });
            });
    });

    it("can correctly display old versions", function () {
        const colName = `Versioned Collection ${Math.floor(Math.random() * 999999)}`;
        let colUuid = "";
        let oldVersionUuid = "";
        // Make sure no other collections with this name exist
        cy.doRequest("GET", "/arvados/v1/collections", null, {
            filters: `[["name", "=", "${colName}"]]`,
            include_old_versions: true,
        })
            .its("body.items")
            .as("collections")
            .then(function () {
                expect(this.collections).to.be.empty;
            });
        // Creates the collection using the admin token so we can set up
        // a bogus manifest text without block signatures.
        cy.createCollection(adminUser.token, {
            name: colName,
            owner_uuid: activeUser.user.uuid,
            preserve_version: true,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n",
        })
            .as("originalVersion")
            .then(function () {
                // Change the file name to create a new version.
                cy.updateCollection(adminUser.token, this.originalVersion.uuid, {
                    manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:foo\n",
                });
                colUuid = this.originalVersion.uuid;
            });
        // Confirm that there are 2 versions of the collection
        cy.doRequest("GET", "/arvados/v1/collections", null, {
            filters: `[["name", "=", "${colName}"]]`,
            include_old_versions: true,
        })
            .its("body.items")
            .as("collections")
            .then(function () {
                expect(this.collections).to.have.lengthOf(2);
                this.collections.map(function (aCollection) {
                    expect(aCollection.current_version_uuid).to.equal(colUuid);
                    if (aCollection.uuid !== aCollection.current_version_uuid) {
                        oldVersionUuid = aCollection.uuid;
                    }
                });
                // Check the old version displays as what it is.
                cy.loginAs(activeUser);
                cy.goToPath(`/collections/${oldVersionUuid}`);

                cy.get("[data-cy=collection-info-panel]").should("contain", "This is an old version");
                cy.get("[data-cy=read-only-icon]").should("exist");
                cy.get("[data-cy=collection-info-panel]").should("contain", colName);
                cy.get("[data-cy=collection-files-panel]").should("contain", "bar");
            });
    });

    it("views & edits storage classes data", function () {
        const colName = `Test Collection ${Math.floor(Math.random() * 999999)}`;
        cy.createCollection(adminUser.token, {
            name: colName,
            owner_uuid: activeUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:some-file\n",
        })
            .as("collection")
            .then(function () {
                expect(this.collection.storage_classes_desired).to.deep.equal(["default"]);

                cy.loginAs(activeUser);
                cy.goToPath(`/collections/${this.collection.uuid}`);

                // Initial check: it should show the 'default' storage class
                cy.get("[data-cy=collection-info-panel]")
                    .should("contain", "Storage classes")
                    .and("contain", "default")
                    .and("not.contain", "foo")
                    .and("not.contain", "bar");
                // Edit collection: add storage class 'foo'
                cy.get("[data-cy=collection-panel-options-btn]").click();
                cy.get("[data-cy=context-menu]").contains("Edit collection").click();
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
                cy.get("[data-cy=collection-info-panel]").should("contain", "default").and("contain", "foo").and("not.contain", "bar");
                cy.doRequest("GET", `/arvados/v1/collections/${this.collection.uuid}`)
                    .its("body")
                    .as("updatedCollection")
                    .then(function () {
                        expect(this.updatedCollection.storage_classes_desired).to.deep.equal(["default", "foo"]);
                    });
                // Edit collection: remove storage class 'default'
                cy.get("[data-cy=collection-panel-options-btn]").click();
                cy.get("[data-cy=context-menu]").contains("Edit collection").click();
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
                cy.get("[data-cy=collection-info-panel]").should("not.contain", "default").and("contain", "foo").and("not.contain", "bar");
                cy.doRequest("GET", `/arvados/v1/collections/${this.collection.uuid}`)
                    .its("body")
                    .as("updatedCollection")
                    .then(function () {
                        expect(this.updatedCollection.storage_classes_desired).to.deep.equal(["foo"]);
                    });
            });
    });

    it("moves a collection to a different project", function () {
        const collName = `Test Collection ${Math.floor(Math.random() * 999999)}`;
        const projName = `Test Project ${Math.floor(Math.random() * 999999)}`;
        const fileName = `Test_File_${Math.floor(Math.random() * 999999)}`;

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

        cy.getAll("@testCollection", "@testProject").then(function ([testCollection, testProject]) {
            cy.loginAs(activeUser);
            cy.goToPath(`/collections/${testCollection.uuid}`);
            cy.get("[data-cy=collection-files-panel]").should("contain", fileName);
            cy.get("[data-cy=collection-info-panel]").should("not.contain", projName).and("not.contain", testProject.uuid);
            cy.get("[data-cy=collection-panel-options-btn]").click();
            cy.get("[data-cy=context-menu]").contains("Move to").click();
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
            cy.get("[data-cy=snackbar]").contains("Collection has been moved");
            cy.get("[data-cy=collection-info-panel]").contains(projName).and("contain", testProject.uuid);
            // Double check that the collection is in the project
            cy.goToPath(`/projects/${testProject.uuid}`);
            cy.waitForDom().get("[data-cy=project-panel]").should("contain", collName);
        });
    });

    it("automatically updates the collection UI contents without using the Refresh button", function () {
        const collName = `Test Collection ${Math.floor(Math.random() * 999999)}`;

        cy.createCollection(adminUser.token, {
            name: collName,
            owner_uuid: activeUser.user.uuid,
        }).as("testCollection");

        cy.getAll("@testCollection").then(function ([testCollection]) {
            cy.loginAs(activeUser);

            const files = ["foobar", "anotherFile", "", "finalName"];

            cy.goToPath(`/collections/${testCollection.uuid}`);
            cy.get("[data-cy=collection-files-panel]").should("contain", "This collection is empty");
            cy.get("[data-cy=collection-files-panel]").should("not.contain", files[0]);
            cy.get("[data-cy=collection-info-panel]").should("contain", collName);

            files.map((fileName, i, files) => {
                cy.updateCollection(adminUser.token, testCollection.uuid, {
                    name: `${collName + " updated"}`,
                    manifest_text: fileName ? `. 37b51d194a7513e45b56f6524f2d51f2+3 0:3:${fileName}\n` : "",
                }).as("updatedCollection");
                cy.getAll("@updatedCollection").then(function ([updatedCollection]) {
                    expect(updatedCollection.name).to.equal(`${collName + " updated"}`);
                    cy.get("[data-cy=collection-info-panel]").should("contain", updatedCollection.name);
                    fileName
                        ? cy.get("[data-cy=collection-files-panel]").should("contain", fileName)
                        : cy.get("[data-cy=collection-files-panel]").should("not.contain", files[i - 1]);
                });
            });
        });
    });

    it("makes a copy of an existing collection", function () {
        const collName = `Test Collection ${Math.floor(Math.random() * 999999)}`;
        const copyName = `Copy of: ${collName}`;

        cy.createCollection(adminUser.token, {
            name: collName,
            owner_uuid: activeUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:some-file\n",
        })
            .as("collection")
            .then(function () {
                cy.loginAs(activeUser);
                cy.goToPath(`/collections/${this.collection.uuid}`);
                cy.get("[data-cy=collection-files-panel]").should("contain", "some-file");
                cy.get("[data-cy=collection-panel-options-btn]").click();
                cy.get("[data-cy=context-menu]").contains("Make a copy").click();
                cy.get("[data-cy=form-dialog]")
                    .should("contain", "Make a copy")
                    .within(() => {
                        cy.get("[data-cy=projects-tree-home-tree-picker]").contains("Projects").click();
                        cy.get("[data-cy=form-submit-btn]").click();
                    });
                cy.get("[data-cy=snackbar]").contains("Collection has been copied.");
                cy.get("[data-cy=snackbar-goto-action]").click();
                cy.get("[data-cy=project-panel]").contains(copyName).click();
                cy.get("[data-cy=collection-files-panel]").should("contain", "some-file");
            });
    });

    it("uses the collection version browser to view a previous version", function () {
        const colName = `Test Collection ${Math.floor(Math.random() * 999999)}`;

        // Creates the collection using the admin token so we can set up
        // a bogus manifest text without block signatures.
        cy.createCollection(adminUser.token, {
            name: colName,
            owner_uuid: activeUser.user.uuid,
            preserve_version: true,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:foo 0:3:bar\n",
        })
            .as("collection")
            .then(function () {
                // Visit collection, check basic information
                cy.loginAs(activeUser);
                cy.goToPath(`/collections/${this.collection.uuid}`);

                cy.get("[data-cy=collection-info-panel]").should("not.contain", "This is an old version");
                cy.get("[data-cy=read-only-icon]").should("not.exist");
                cy.get("[data-cy=collection-version-number]").should("contain", "1");
                cy.get("[data-cy=collection-info-panel]").should("contain", colName);
                cy.get("[data-cy=collection-files-panel]").should("contain", "foo").and("contain", "bar");

                // Modify collection, expect version number change
                cy.get("[data-cy=collection-files-panel]").contains("foo").rightclick();
                cy.get("[data-cy=context-menu]").contains("Remove").click();
                cy.get("[data-cy=confirmation-dialog]").should("contain", "Removing file");
                cy.get("[data-cy=confirmation-dialog-ok-btn]").click();
                cy.get("[data-cy=collection-version-number]").should("contain", "2");
                cy.get("[data-cy=collection-files-panel]").should("not.contain", "foo").and("contain", "bar");

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
                cy.get("[data-cy=collection-info-panel]").should("contain", "This is an old version");
                cy.get("[data-cy=read-only-icon]").should("exist");
                cy.get("[data-cy=collection-version-number]").should("contain", "1");
                cy.get("[data-cy=collection-info-panel]").should("contain", colName);
                cy.get("[data-cy=collection-files-panel]").should("contain", "foo").and("contain", "bar");

                // Check that only old collection action are available on context menu
                cy.get("[data-cy=collection-panel-options-btn]").click();
                cy.get("[data-cy=context-menu]").should("contain", "Restore version").and("not.contain", "Add to favorites");
                cy.get("body").click(); // Collapse the menu avoiding details panel expansion

                // Click on "head version" link, confirm that it's the latest version.
                cy.get("[data-cy=collection-info-panel]").contains("head version").click();
                cy.get("[data-cy=collection-info-panel]").should("not.contain", "This is an old version");
                cy.get("[data-cy=read-only-icon]").should("not.exist");
                cy.get("[data-cy=collection-version-number]").should("contain", "2");
                cy.get("[data-cy=collection-info-panel]").should("contain", colName);
                cy.get("[data-cy=collection-files-panel]").should("not.contain", "foo").and("contain", "bar");

                // Check that old collection action isn't available on context menu
                cy.get("[data-cy=collection-panel-options-btn]").click();
                cy.get("[data-cy=context-menu]").should("not.contain", "Restore version");
                cy.get("body").click(); // Collapse the menu avoiding details panel expansion

                // Make another change, confirm new version.
                cy.get("[data-cy=collection-panel-options-btn]").click();
                cy.get("[data-cy=context-menu]").contains("Edit collection").click();
                cy.get("[data-cy=form-dialog]")
                    .should("contain", "Edit Collection")
                    .within(() => {
                        // appends some text
                        cy.get("input").first().type(" renamed");
                    });
                cy.get("[data-cy=form-submit-btn]").click();
                cy.get("[data-cy=collection-info-panel]").should("not.contain", "This is an old version");
                cy.get("[data-cy=read-only-icon]").should("not.exist");
                cy.get("[data-cy=collection-version-number]").should("contain", "3");
                cy.get("[data-cy=collection-info-panel]").should("contain", colName + " renamed");
                cy.get("[data-cy=collection-files-panel]").should("not.contain", "foo").and("contain", "bar");
                cy.get("[data-cy=collection-version-browser-select-3]").should("contain", "3").and("contain", "3 B");

                // Check context menus on version browser
                cy.waitForDom();
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
                cy.get("[data-cy=collection-version-browser]").within(() => {
                    cy.get("[data-cy=collection-version-browser-select-1]").click();
                });
                cy.get("[data-cy=collection-panel-options-btn]").click();
                cy.get("[data-cy=context-menu]").contains("Restore version").click();
                cy.get("[data-cy=confirmation-dialog]").should("contain", "Restore version");
                cy.get("[data-cy=confirmation-dialog-ok-btn]").click();
                cy.get("[data-cy=collection-info-panel]").should("not.contain", "This is an old version");
                cy.get("[data-cy=collection-version-number]").should("contain", "4");
                cy.get("[data-cy=collection-info-panel]").should("contain", colName);
                cy.get("[data-cy=collection-files-panel]").should("contain", "foo").and("contain", "bar");
            });
    });

    it("copies selected files into new collection", () => {
        cy.createCollection(adminUser.token, {
            name: `Test Collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: activeUser.user.uuid,
            preserve_version: true,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:foo 0:3:bar\n",
        })
            .as("collection")
            .then(function () {
                // Visit collection, check basic information
                cy.loginAs(activeUser);
                cy.goToPath(`/collections/${this.collection.uuid}`);

                cy.get("[data-cy=collection-files-panel]").within(() => {
                    cy.get("input[type=checkbox]").first().click();
                });

                cy.get("[data-cy=collection-files-panel-options-btn]").click();
                cy.get("[data-cy=context-menu]").contains("Copy selected into new collection").click();

                cy.get("[data-cy=form-dialog]").contains("Projects").click();

                cy.get("[data-cy=form-submit-btn]").click();

                cy.waitForDom().get(".layout-pane-primary", { timeout: 12000 }).contains("Projects").click();

                cy.waitForDom().get("main").contains(`Files extracted from: ${this.collection.name}`).click();
                cy.get("[data-cy=collection-files-panel]").and("contain", "bar");
            });
    });

    it("copies selected files into existing collection", () => {
        cy.createCollection(adminUser.token, {
            name: `Test Collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: activeUser.user.uuid,
            preserve_version: true,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:foo 0:3:bar\n",
        }).as("sourceCollection");

        cy.createCollection(adminUser.token, {
            name: `Destination Collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: activeUser.user.uuid,
            preserve_version: true,
            manifest_text: "",
        }).as("destinationCollection");

        cy.getAll("@sourceCollection", "@destinationCollection").then(function ([sourceCollection, destinationCollection]) {
            // Visit collection, check basic information
            cy.loginAs(activeUser);
            cy.goToPath(`/collections/${sourceCollection.uuid}`);

            cy.get("[data-cy=collection-files-panel]").within(() => {
                cy.get("input[type=checkbox]").first().click();
            });

            cy.get("[data-cy=collection-files-panel-options-btn]").click();
            cy.get("[data-cy=context-menu]").contains("Copy selected into existing collection").click();

            cy.get("[data-cy=form-dialog]").contains(destinationCollection.name).click();

            cy.get("[data-cy=form-submit-btn]").click();
            cy.wait(2000);

            cy.goToPath(`/collections/${destinationCollection.uuid}`);

            cy.get("main").contains(destinationCollection.name).should("exist");
            cy.get("[data-cy=collection-files-panel]").and("contain", "bar");
        });
    });

    it("copies selected files into separate collections", () => {
        cy.createCollection(adminUser.token, {
            name: `Test Collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: activeUser.user.uuid,
            preserve_version: true,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:foo 0:3:bar\n",
        }).as("sourceCollection");

        cy.getAll("@sourceCollection").then(function ([sourceCollection]) {
            // Visit collection, check basic information
            cy.loginAs(activeUser);
            cy.goToPath(`/collections/${sourceCollection.uuid}`);

            // Select both files
            cy.waitForDom()
                .get("[data-cy=collection-files-panel]")
                .within(() => {
                    cy.get("input[type=checkbox]").first().click();
                    cy.get("input[type=checkbox]").last().click();
                });

            // Copy to separate collections
            cy.get("[data-cy=collection-files-panel-options-btn]").click();
            cy.get("[data-cy=context-menu]").contains("Copy selected into separate collections").click();
            cy.get("[data-cy=form-dialog]").contains("Projects").click();
            cy.get("[data-cy=form-submit-btn]").click();

            // Verify created collections
            cy.waitForDom().get(".layout-pane-primary", { timeout: 12000 }).contains("Projects").click();
            cy.get("main").contains(`File copied from collection ${sourceCollection.name}/foo`).click();
            cy.get("[data-cy=collection-files-panel]").and("contain", "foo");
            cy.get(".layout-pane-primary").contains("Projects").click();
            cy.get("main").contains(`File copied from collection ${sourceCollection.name}/bar`).click();
            cy.get("[data-cy=collection-files-panel]").and("contain", "bar");

            // Verify separate collection menu items not present when single file selected
            // Wait for dom for collection to re-render
            cy.waitForDom()
                .get("[data-cy=collection-files-panel]")
                .within(() => {
                    cy.get("input[type=checkbox]").first().click();
                });
            cy.get("[data-cy=collection-files-panel-options-btn]").click();
            cy.get("[data-cy=context-menu]").should("not.contain", "Copy selected into separate collections");
            cy.get("[data-cy=context-menu]").should("not.contain", "Move selected into separate collections");
        });
    });

    it("moves selected files into new collection", () => {
        cy.createCollection(adminUser.token, {
            name: `Test Collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: activeUser.user.uuid,
            preserve_version: true,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:foo 0:3:bar\n",
        })
            .as("collection")
            .then(function () {
                // Visit collection, check basic information
                cy.loginAs(activeUser);
                cy.goToPath(`/collections/${this.collection.uuid}`);

                cy.get("[data-cy=collection-files-panel]").within(() => {
                    cy.get("input[type=checkbox]").first().click();
                });

                cy.get("[data-cy=collection-files-panel-options-btn]").click();
                cy.get("[data-cy=context-menu]").contains("Move selected into new collection").click();

                cy.get("[data-cy=form-dialog]").contains("Projects").click();

                cy.get("[data-cy=form-submit-btn]").click();

                cy.waitForDom().get(".layout-pane-primary", { timeout: 12000 }).contains("Projects").click();

                cy.get("main").contains(`Files moved from: ${this.collection.name}`).click();
                cy.get("[data-cy=collection-files-panel]").and("contain", "bar");
            });
    });

    it("moves selected files into existing collection", () => {
        cy.createCollection(adminUser.token, {
            name: `Test Collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: activeUser.user.uuid,
            preserve_version: true,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:foo 0:3:bar\n",
        }).as("sourceCollection");

        cy.createCollection(adminUser.token, {
            name: `Destination Collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: activeUser.user.uuid,
            preserve_version: true,
            manifest_text: "",
        }).as("destinationCollection");

        cy.getAll("@sourceCollection", "@destinationCollection").then(function ([sourceCollection, destinationCollection]) {
            // Visit collection, check basic information
            cy.loginAs(activeUser);
            cy.goToPath(`/collections/${sourceCollection.uuid}`);

            cy.get("[data-cy=collection-files-panel]").within(() => {
                cy.get("input[type=checkbox]").first().click();
            });

            cy.get("[data-cy=collection-files-panel-options-btn]").click();
            cy.get("[data-cy=context-menu]").contains("Move selected into existing collection").click();

            cy.get("[data-cy=form-dialog]").contains(destinationCollection.name).click();

            cy.get("[data-cy=form-submit-btn]").click();
            cy.wait(2000);

            cy.goToPath(`/collections/${destinationCollection.uuid}`);

            cy.get("main").contains(destinationCollection.name).should("exist");
            cy.get("[data-cy=collection-files-panel]").and("contain", "bar");
        });
    });

    it("moves selected files into separate collections", () => {
        cy.createCollection(adminUser.token, {
            name: `Test Collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: activeUser.user.uuid,
            preserve_version: true,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:foo 0:3:bar\n",
        }).as("sourceCollection");

        cy.getAll("@sourceCollection").then(function ([sourceCollection]) {
            // Visit collection, check basic information
            cy.loginAs(activeUser);
            cy.goToPath(`/collections/${sourceCollection.uuid}`);

            // Select both files
            cy.get("[data-cy=collection-files-panel]").within(() => {
                cy.get("input[type=checkbox]").first().click();
                cy.get("input[type=checkbox]").last().click();
            });

            // Copy to separate collections
            cy.get("[data-cy=collection-files-panel-options-btn]").click();
            cy.get("[data-cy=context-menu]").contains("Move selected into separate collections").click();
            cy.get("[data-cy=form-dialog]").contains("Projects").click();
            cy.get("[data-cy=form-submit-btn]").click();

            // Verify created collections
            cy.waitForDom().get(".layout-pane-primary", { timeout: 12000 }).contains("Projects").click();
            cy.get("main").contains(`File moved from collection ${sourceCollection.name}/foo`).click();
            cy.get("[data-cy=collection-files-panel]").and("contain", "foo");
            cy.get(".layout-pane-primary").contains("Projects").click();
            cy.get("main").contains(`File moved from collection ${sourceCollection.name}/bar`).click();
            cy.get("[data-cy=collection-files-panel]").and("contain", "bar");
        });
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
        const collName = `[Test collection (${Math.floor(999999 * Math.random())})]`;

        // Select a storage class.
        cy.get("[data-cy=form-dialog]")
            .should("contain", "New collection")
            .and("contain", "Storage classes")
            .and("contain", "default")
            .and("contain", "foo")
            .and("contain", "bar")
            .within(() => {
                cy.get("[data-cy=parent-field]").within(() => {
                    cy.get("input").should("have.value", "Home project");
                });
                cy.get("[data-cy=name-field]").within(() => {
                    cy.get("input").type(collName);
                });
                cy.get("[data-cy=checkbox-foo]").click();
            });

        // Add a property.
        // Key: Color (IDTAGCOLORS) - Value: Magenta (IDVALCOLORS3)
        cy.get("[data-cy=form-dialog]").should("not.contain", "Color: Magenta");
        cy.get("[data-cy=resource-properties-form]").within(() => {
            cy.get("[data-cy=property-field-key]").within(() => {
                cy.get("input").type("Color");
            });
            cy.get("[data-cy=property-field-value]").within(() => {
                cy.get("input").type("Magenta");
            });
            cy.root().submit();
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
        cy.get("[data-cy=collection-info-panel]")
            .should("contain", "default")
            .and("contain", "foo")
            .and("contain", "Color: Magenta")
            .and("not.contain", "bar");
        // Confirm that the collection's properties has the real values.
        cy.doRequest("GET", "/arvados/v1/collections", null, {
            filters: `[["name", "=", "${collName}"]]`,
        })
            .its("body.items")
            .as("collections")
            .then(function () {
                expect(this.collections).to.have.lengthOf(1);
                expect(this.collections[0].properties).to.have.property("IDTAGCOLORS", "IDVALCOLORS3");
            });
    });

    it("shows responsible person for collection if available", () => {
        cy.createCollection(adminUser.token, {
            name: `Test collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: activeUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n",
        }).as("testCollection1");

        cy.createCollection(adminUser.token, {
            name: `Test collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: adminUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n",
        })
            .as("testCollection2")
            .then(function (testCollection2) {
                cy.shareWith(adminUser.token, activeUser.user.uuid, testCollection2.uuid, "can_write");
            });

        cy.getAll("@testCollection1", "@testCollection2").then(function ([testCollection1, testCollection2]) {
            cy.loginAs(activeUser);

            cy.goToPath(`/collections/${testCollection1.uuid}`);
            cy.get("[data-cy=responsible-person-wrapper]").contains(activeUser.user.uuid);

            cy.goToPath(`/collections/${testCollection2.uuid}`);
            cy.get("[data-cy=responsible-person-wrapper]").contains(adminUser.user.uuid);
        });
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
                cy.get("[data-cy=upload-button]").click();
                cy.get("[data-cy=collection-files-panel]").contains("5mb_a.bin").should("not.exist");
                cy.get("[data-cy=collection-file-count]").should("contain", "2");
                cy.fixture("files/5mb.bin", "base64").then(content => {
                    cy.get("[data-cy=drag-and-drop]").upload(content, "5mb_a.bin");
                    cy.get("[data-cy=form-submit-btn]").click();
                    cy.get("[data-cy=form-submit-btn]").should("not.exist");
                    cy.get("[data-cy=collection-files-panel]").contains("5mb_a.bin").should("exist");
                    cy.get("[data-cy=collection-file-count]").should("contain", "3");

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

        it("allows to cancel running upload", () => {
            cy.getAll("@testCollection1").then(function ([testCollection1]) {
                cy.loginAs(activeUser);

                cy.goToPath(`/collections/${testCollection1.uuid}`);

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

                // Confirm initial collection state.
                cy.get("[data-cy=collection-files-panel]").contains("bar").should("exist");
                cy.get("[data-cy=collection-files-panel]").contains("5mb_a.bin").should("not.exist");
                cy.get("[data-cy=collection-files-panel]").contains("5mb_b.bin").should("not.exist");

                cy.get("[data-cy=upload-button]").click();

                cy.fixture("files/5mb.bin", "base64").then(content => {
                    cy.get("[data-cy=drag-and-drop]").upload(content, "5mb_a.bin");
                    cy.get("[data-cy=drag-and-drop]").upload(content, "5mb_b.bin");

                    cy.get("[data-cy=form-submit-btn]").click();

                    cy.get("button[aria-label=Remove]").should("exist");
                    cy.get("button[aria-label=Remove]").click({ multiple: true, force: true });

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
});
