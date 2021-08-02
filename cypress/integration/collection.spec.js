// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

const path = require('path');

describe('Collection panel tests', function () {
    let activeUser;
    let adminUser;
    let downloadsFolder;

    before(function () {
        // Only set up common users once. These aren't set up as aliases because
        // aliases are cleaned up after every test. Also it doesn't make sense
        // to set the same users on beforeEach() over and over again, so we
        // separate a little from Cypress' 'Best Practices' here.
        cy.getUser('admin', 'Admin', 'User', true, true)
            .as('adminUser').then(function () {
                adminUser = this.adminUser;
            }
            );
        cy.getUser('collectionuser1', 'Collection', 'User', false, true)
            .as('activeUser').then(function () {
                activeUser = this.activeUser;
            }
            );
        downloadsFolder = Cypress.config('downloadsFolder');
    });

    beforeEach(function () {
        cy.clearCookies();
        cy.clearLocalStorage();
    });

    it('allows to download mountain duck config for a collection', () => {
        cy.createCollection(adminUser.token, {
            name: `Test collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: activeUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n"
        })
        .as('testCollection').then(function (testCollection) {
            cy.loginAs(activeUser);
            cy.goToPath(`/collections/${testCollection.uuid}`);

            cy.get('[data-cy=collection-panel-options-btn]').click();
            cy.get('[data-cy=context-menu]').contains('Open with 3rd party client').click();
            cy.get('[data-cy=download-button').click();

            const filename = path.join(downloadsFolder, `${testCollection.name}.duck`);

            cy.readFile(filename, { timeout: 15000 })
                .then((body) => {
                    const childrenCollection = Array.prototype.slice.call(Cypress.$(body).find('dict')[0].children);
                    const map = {};
                    let i, j = 2;

                    for (i=0; i < childrenCollection.length; i += j) {
                      map[childrenCollection[i].outerText] = childrenCollection[i + 1].outerText;
                    }

                    cy.get('#simple-tabpanel-0').find('a')
                        .then((a) => {
                            const [host, port] = a.text().split('@')[1].split('/')[0].split(':');
                            expect(map['Protocol']).to.equal('davs');
                            expect(map['UUID']).to.equal(testCollection.uuid);
                            expect(map['Username']).to.equal(activeUser.user.username);
                            expect(map['Port']).to.equal(port);
                            expect(map['Hostname']).to.equal(host);
                            if (map['Path']) {
                                expect(map['Path']).to.equal(`/c=${testCollection.uuid}`);
                            }
                        });
                })
                .then(() => cy.task('clearDownload', { filename }));
        });
    });

    it('uses the property editor with vocabulary terms', function () {
        cy.createCollection(adminUser.token, {
            name: `Test collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: activeUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n"
        })
            .as('testCollection').then(function () {
                cy.loginAs(activeUser);
                cy.goToPath(`/collections/${this.testCollection.uuid}`);

                // Key: Color (IDTAGCOLORS) - Value: Magenta (IDVALCOLORS3)
                cy.get('[data-cy=resource-properties-form]').within(() => {
                    cy.get('[data-cy=property-field-key]').within(() => {
                        cy.get('input').type('Color');
                    });
                    cy.get('[data-cy=property-field-value]').within(() => {
                        cy.get('input').type('Magenta');
                    });
                    cy.root().submit();
                });
                // Confirm proper vocabulary labels are displayed on the UI.
                cy.get('[data-cy=collection-properties-panel]')
                    .should('contain', 'Color')
                    .and('contain', 'Magenta');
                // Confirm proper vocabulary IDs were saved on the backend.
                cy.doRequest('GET', `/arvados/v1/collections/${this.testCollection.uuid}`)
                    .its('body').as('collection')
                    .then(function () {
                        expect(this.collection.properties.IDTAGCOLORS).to.equal('IDVALCOLORS3');
                    });
            });
    });

    it('shows collection by URL', function () {
        cy.loginAs(activeUser);
        [true, false].map(function (isWritable) {
            // Using different file names to avoid test flakyness: the second iteration
            // on this loop may pass an assertion from the first iteration by looking
            // for the same file name.
            const fileName = isWritable ? 'bar' : 'foo';
            const subDirName = 'subdir';
            cy.createGroup(adminUser.token, {
                name: 'Shared project',
                group_class: 'project',
            }).as('sharedGroup').then(function () {
                // Creates the collection using the admin token so we can set up
                // a bogus manifest text without block signatures.
                cy.createCollection(adminUser.token, {
                    name: 'Test collection',
                    owner_uuid: this.sharedGroup.uuid,
                    properties: { someKey: 'someValue' },
                    manifest_text: `. 37b51d194a7513e45b56f6524f2d51f2+3 0:3:${fileName}\n./${subDirName} 37b51d194a7513e45b56f6524f2d51f2+3 0:3:${fileName}\n`
                })
                    .as('testCollection').then(function () {
                        // Share the group with active user.
                        cy.createLink(adminUser.token, {
                            name: isWritable ? 'can_write' : 'can_read',
                            link_class: 'permission',
                            head_uuid: this.sharedGroup.uuid,
                            tail_uuid: activeUser.user.uuid
                        })
                        cy.goToPath(`/collections/${this.testCollection.uuid}`);

                        // Check that name & uuid are correct.
                        cy.get('[data-cy=collection-info-panel]')
                            .should('contain', this.testCollection.name)
                            .and('contain', this.testCollection.uuid)
                            .and('not.contain', 'This is an old version');
                        // Check for the read-only icon
                        cy.get('[data-cy=read-only-icon]').should(`${isWritable ? 'not.' : ''}exist`);
                        // Check that both read and write operations are available on
                        // the 'More options' menu.
                        cy.get('[data-cy=collection-panel-options-btn]')
                            .click()
                        cy.get('[data-cy=context-menu]')
                            .should('contain', 'Add to favorites')
                            .and(`${isWritable ? '' : 'not.'}contain`, 'Edit collection');
                        cy.get('body').click(); // Collapse the menu avoiding details panel expansion
                        cy.get('[data-cy=collection-properties-panel]')
                            .should('contain', 'someKey')
                            .and('contain', 'someValue')
                            .and('not.contain', 'anotherKey')
                            .and('not.contain', 'anotherValue')
                        if (isWritable === true) {
                            // Check that properties can be added.
                            cy.get('[data-cy=resource-properties-form]').within(() => {
                                cy.get('[data-cy=property-field-key]').within(() => {
                                    cy.get('input').type('anotherKey');
                                });
                                cy.get('[data-cy=property-field-value]').within(() => {
                                    cy.get('input').type('anotherValue');
                                });
                                cy.root().submit();
                            })
                            cy.get('[data-cy=collection-properties-panel]')
                                .should('contain', 'anotherKey')
                                .and('contain', 'anotherValue')
                        } else {
                            // Properties form shouldn't be displayed.
                            cy.get('[data-cy=resource-properties-form]').should('not.exist');
                        }
                        // Check that the file listing show both read & write operations
                        cy.get('[data-cy=collection-files-panel]').within(() => {
                            cy.root().should('contain', fileName);
                            if (isWritable) {
                                cy.get('[data-cy=upload-button]')
                                    .should(`${isWritable ? '' : 'not.'}contain`, 'Upload data');
                            }
                        });
                        // Test context menus
                        cy.get('[data-cy=collection-files-panel]')
                            .contains(fileName).rightclick({ force: true });
                        cy.get('[data-cy=context-menu]')
                            .should('contain', 'Download')
                            .and('contain', 'Open in new tab')
                            .and('contain', 'Copy to clipboard')
                            .and(`${isWritable ? '' : 'not.'}contain`, 'Rename')
                            .and(`${isWritable ? '' : 'not.'}contain`, 'Remove');
                        cy.get('body').click(); // Collapse the menu
                        cy.get('[data-cy=collection-files-panel]')
                            .contains(subDirName).rightclick({ force: true });
                        cy.get('[data-cy=context-menu]')
                            .should('not.contain', 'Download')
                            .and('contain', 'Open in new tab')
                            .and('contain', 'Copy to clipboard')
                            .and(`${isWritable ? '' : 'not.'}contain`, 'Rename')
                            .and(`${isWritable ? '' : 'not.'}contain`, 'Remove');
                        cy.get('body').click(); // Collapse the menu
                        // Hamburger 'more options' menu button
                        cy.get('[data-cy=collection-files-panel-options-btn]')
                            .click()
                        cy.get('[data-cy=context-menu]')
                            .should('contain', 'Select all')
                            .click()
                        cy.get('[data-cy=collection-files-panel-options-btn]')
                            .click()
                        cy.get('[data-cy=context-menu]')
                            .should(`${isWritable ? '' : 'not.'}contain`, 'Remove selected')
                        cy.get('body').click(); // Collapse the menu
                    })
            })
        })
    })

    it('renames a file using valid names', function () {
        function eachPair(lst, func){
            for(var i=0; i < lst.length - 1; i++){
                func(lst[i], lst[i + 1])
            }
        }
        // Creates the collection using the admin token so we can set up
        // a bogus manifest text without block signatures.
        cy.createCollection(adminUser.token, {
            name: `Test collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: activeUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n"
        })
            .as('testCollection').then(function () {
                cy.loginAs(activeUser);
                cy.goToPath(`/collections/${this.testCollection.uuid}`);

                const names = [
                    'bar', // initial name already set
                    '&',
                    'foo',
                    '&amp;',
                    'I ❤️ ⛵️',
                    '...',
                    '#..',
                    'some name with whitespaces',
                    'some name with #2',
                    'is this name legal? I hope it is',
                    'some_file.pdf#',
                    'some_file.pdf?',
                    '?some_file.pdf',
                    'some%file.pdf',
                    'some%2Ffile.pdf',
                    'some%22file.pdf',
                    'some%20file.pdf',
                    "G%C3%BCnter's%20file.pdf",
                    'table%&?*2',
                    'bar' // make sure we can go back to the original name as a last step
                ];
                eachPair(names, (from, to) => {
                    cy.get('[data-cy=collection-files-panel]')
                        .contains(`${from}`).rightclick();
                    cy.get('[data-cy=context-menu]')
                        .contains('Rename')
                        .click();
                    cy.get('[data-cy=form-dialog]')
                        .should('contain', 'Rename')
                        .within(() => {
                            cy.get('input')
                                .type('{selectall}{backspace}')
                                .type(to, { parseSpecialCharSequences: false });
                        });
                    cy.get('[data-cy=form-submit-btn]').click();
                    cy.get('[data-cy=collection-files-panel]')
                        .should('not.contain', `${from}`)
                        .and('contain', `${to}`);
                })
            });
    });

    it('renames a file to a different directory', function () {
        // Creates the collection using the admin token so we can set up
        // a bogus manifest text without block signatures.
        cy.createCollection(adminUser.token, {
            name: `Test collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: activeUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n"
        })
            .as('testCollection').then(function () {
                cy.loginAs(activeUser);
                cy.goToPath(`/collections/${this.testCollection.uuid}`);

                ['subdir', 'G%C3%BCnter\'s%20file', 'table%&?*2'].forEach((subdir) => {
                    cy.get('[data-cy=collection-files-panel]')
                        .contains('bar').rightclick({force: true});
                    cy.get('[data-cy=context-menu]')
                        .contains('Rename')
                        .click();
                    cy.get('[data-cy=form-dialog]')
                        .should('contain', 'Rename')
                        .within(() => {
                            cy.get('input').type(`{selectall}{backspace}${subdir}/foo`);
                        });
                    cy.get('[data-cy=form-submit-btn]').click();
                    cy.get('[data-cy=collection-files-panel]')
                        .should('not.contain', 'bar')
                        .and('contain', subdir);
                    // Look for the "arrow icon" and expand the "subdir" directory.
                    cy.get('[data-cy=virtual-file-tree] > div > i').click();
                    // Rename 'subdir/foo' to 'foo'
                    cy.get('[data-cy=collection-files-panel]')
                        .contains('foo').rightclick();
                    cy.get('[data-cy=context-menu]')
                        .contains('Rename')
                        .click();
                    cy.get('[data-cy=form-dialog]')
                        .should('contain', 'Rename')
                        .within(() => {
                            cy.get('input')
                                .should('have.value', `${subdir}/foo`)
                                .type(`{selectall}{backspace}bar`);
                        });
                    cy.get('[data-cy=form-submit-btn]').click();
                    cy.get('[data-cy=collection-files-panel]')
                        .should('contain', subdir) // empty dir kept
                        .and('contain', 'bar');

                    cy.get('[data-cy=collection-files-panel]')
                        .contains(subdir).rightclick();
                    cy.get('[data-cy=context-menu]')
                        .contains('Remove')
                        .click();
                    cy.get('[data-cy=confirmation-dialog-ok-btn]').click();
                });
            });
    });

    it('tries to rename a file with illegal names', function () {
        // Creates the collection using the admin token so we can set up
        // a bogus manifest text without block signatures.
        cy.createCollection(adminUser.token, {
            name: `Test collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: activeUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n"
        })
            .as('testCollection').then(function () {
                cy.loginAs(activeUser);
                cy.goToPath(`/collections/${this.testCollection.uuid}`);

                const illegalNamesFromUI = [
                    ['.', "Name cannot be '.' or '..'"],
                    ['..', "Name cannot be '.' or '..'"],
                    ['', 'This field is required'],
                    [' ', 'Leading/trailing whitespaces not allowed'],
                    [' foo', 'Leading/trailing whitespaces not allowed'],
                    ['foo ', 'Leading/trailing whitespaces not allowed'],
                    ['//foo', 'Empty dir name not allowed']
                ]
                illegalNamesFromUI.forEach(([name, errMsg]) => {
                    cy.get('[data-cy=collection-files-panel]')
                        .contains('bar').rightclick();
                    cy.get('[data-cy=context-menu]')
                        .contains('Rename')
                        .click();
                    cy.get('[data-cy=form-dialog]')
                        .should('contain', 'Rename')
                        .within(() => {
                            cy.get('input').type(`{selectall}{backspace}${name}`);
                        });
                    cy.get('[data-cy=form-dialog]')
                        .should('contain', 'Rename')
                        .within(() => {
                            cy.contains(`${errMsg}`);
                        });
                    cy.get('[data-cy=form-cancel-btn]').click();
                })
            });
    });

    it('can correctly display old versions', function () {
        const colName = `Versioned Collection ${Math.floor(Math.random() * 999999)}`;
        let colUuid = '';
        let oldVersionUuid = '';
        // Make sure no other collections with this name exist
        cy.doRequest('GET', '/arvados/v1/collections', null, {
            filters: `[["name", "=", "${colName}"]]`,
            include_old_versions: true
        })
            .its('body.items').as('collections')
            .then(function () {
                expect(this.collections).to.be.empty;
            });
        // Creates the collection using the admin token so we can set up
        // a bogus manifest text without block signatures.
        cy.createCollection(adminUser.token, {
            name: colName,
            owner_uuid: activeUser.user.uuid,
            preserve_version: true,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n"
        })
            .as('originalVersion').then(function () {
                // Change the file name to create a new version.
                cy.updateCollection(adminUser.token, this.originalVersion.uuid, {
                    manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:foo\n"
                })
                colUuid = this.originalVersion.uuid;
            });
        // Confirm that there are 2 versions of the collection
        cy.doRequest('GET', '/arvados/v1/collections', null, {
            filters: `[["name", "=", "${colName}"]]`,
            include_old_versions: true
        })
            .its('body.items').as('collections')
            .then(function () {
                expect(this.collections).to.have.lengthOf(2);
                this.collections.map(function (aCollection) {
                    expect(aCollection.current_version_uuid).to.equal(colUuid);
                    if (aCollection.uuid !== aCollection.current_version_uuid) {
                        oldVersionUuid = aCollection.uuid;
                    }
                });
                // Check the old version displays as what it is.
                cy.loginAs(activeUser)
                cy.goToPath(`/collections/${oldVersionUuid}`);

                cy.get('[data-cy=collection-info-panel]').should('contain', 'This is an old version');
                cy.get('[data-cy=read-only-icon]').should('exist');
                cy.get('[data-cy=collection-info-panel]').should('contain', colName);
                cy.get('[data-cy=collection-files-panel]').should('contain', 'bar');
            });
    });

    it('views & edits storage classes data', function () {
        const colName= `Test Collection ${Math.floor(Math.random() * 999999)}`;
        cy.createCollection(adminUser.token, {
            name: colName,
            owner_uuid: activeUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:some-file\n",
        }).as('collection').then(function () {
            expect(this.collection.storage_classes_desired).to.deep.equal(['default'])

            cy.loginAs(activeUser)
            cy.goToPath(`/collections/${this.collection.uuid}`);

            // Initial check: it should show the 'default' storage class
            cy.get('[data-cy=collection-info-panel]')
                .should('contain', 'Storage classes')
                .and('contain', 'default')
                .and('not.contain', 'foo')
                .and('not.contain', 'bar');
            // Edit collection: add storage class 'foo'
            cy.get('[data-cy=collection-panel-options-btn]').click();
            cy.get('[data-cy=context-menu]').contains('Edit collection').click();
            cy.get('[data-cy=form-dialog]')
                .should('contain', 'Edit Collection')
                .and('contain', 'Storage classes')
                .and('contain', 'default')
                .and('contain', 'foo')
                .and('contain', 'bar')
                .within(() => {
                    cy.get('[data-cy=checkbox-foo]').click();
                });
            cy.get('[data-cy=form-submit-btn]').click();
            cy.get('[data-cy=collection-info-panel]')
                .should('contain', 'default')
                .and('contain', 'foo')
                .and('not.contain', 'bar');
            cy.doRequest('GET', `/arvados/v1/collections/${this.collection.uuid}`)
                .its('body').as('updatedCollection')
                .then(function () {
                    expect(this.updatedCollection.storage_classes_desired).to.deep.equal(['default', 'foo']);
                });
            // Edit collection: remove storage class 'default'
            cy.get('[data-cy=collection-panel-options-btn]').click();
            cy.get('[data-cy=context-menu]').contains('Edit collection').click();
            cy.get('[data-cy=form-dialog]')
                .should('contain', 'Edit Collection')
                .and('contain', 'Storage classes')
                .and('contain', 'default')
                .and('contain', 'foo')
                .and('contain', 'bar')
                .within(() => {
                    cy.get('[data-cy=checkbox-default]').click();
                });
            cy.get('[data-cy=form-submit-btn]').click();
            cy.get('[data-cy=collection-info-panel]')
                .should('not.contain', 'default')
                .and('contain', 'foo')
                .and('not.contain', 'bar');
            cy.doRequest('GET', `/arvados/v1/collections/${this.collection.uuid}`)
                .its('body').as('updatedCollection')
                .then(function () {
                    expect(this.updatedCollection.storage_classes_desired).to.deep.equal(['foo']);
                });
        })
    });

    it('uses the collection version browser to view a previous version', function () {
        const colName = `Test Collection ${Math.floor(Math.random() * 999999)}`;

        // Creates the collection using the admin token so we can set up
        // a bogus manifest text without block signatures.
        cy.createCollection(adminUser.token, {
            name: colName,
            owner_uuid: activeUser.user.uuid,
            preserve_version: true,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:foo 0:3:bar\n"
        })
            .as('collection').then(function () {
                // Visit collection, check basic information
                cy.loginAs(activeUser)
                cy.goToPath(`/collections/${this.collection.uuid}`);

                cy.get('[data-cy=collection-info-panel]').should('not.contain', 'This is an old version');
                cy.get('[data-cy=read-only-icon]').should('not.exist');
                cy.get('[data-cy=collection-version-number]').should('contain', '1');
                cy.get('[data-cy=collection-info-panel]').should('contain', colName);
                cy.get('[data-cy=collection-files-panel]').should('contain', 'foo').and('contain', 'bar');

                // Modify collection, expect version number change
                cy.get('[data-cy=collection-files-panel]').contains('foo').rightclick();
                cy.get('[data-cy=context-menu]').contains('Remove').click();
                cy.get('[data-cy=confirmation-dialog]').should('contain', 'Removing file');
                cy.get('[data-cy=confirmation-dialog-ok-btn]').click();
                cy.get('[data-cy=collection-version-number]').should('contain', '2');
                cy.get('[data-cy=collection-files-panel]').should('not.contain', 'foo').and('contain', 'bar');

                // Click on version number, check version browser. Click on past version.
                cy.get('[data-cy=collection-version-browser]').should('not.exist');
                cy.get('[data-cy=collection-version-number]').contains('2').click();
                cy.get('[data-cy=collection-version-browser]')
                    .should('contain', 'Nr').and('contain', 'Size').and('contain', 'Date')
                    .within(() => {
                        // Version 1: 6 bytes in size
                        cy.get('[data-cy=collection-version-browser-select-1]')
                            .should('contain', '1').and('contain', '6 B');
                        // Version 2: 3 bytes in size (one file removed)
                        cy.get('[data-cy=collection-version-browser-select-2]')
                            .should('contain', '2').and('contain', '3 B');
                        cy.get('[data-cy=collection-version-browser-select-3]')
                            .should('not.exist');
                        cy.get('[data-cy=collection-version-browser-select-1]')
                            .click();
                    });
                cy.get('[data-cy=collection-info-panel]').should('contain', 'This is an old version');
                cy.get('[data-cy=read-only-icon]').should('exist');
                cy.get('[data-cy=collection-version-number]').should('contain', '1');
                cy.get('[data-cy=collection-info-panel]').should('contain', colName);
                cy.get('[data-cy=collection-files-panel]')
                    .should('contain', 'foo').and('contain', 'bar');

                // Check that only old collection action are available on context menu
                cy.get('[data-cy=collection-panel-options-btn]').click();
                cy.get('[data-cy=context-menu]')
                    .should('contain', 'Restore version')
                    .and('not.contain', 'Add to favorites');
                cy.get('body').click(); // Collapse the menu avoiding details panel expansion

                // Click on "head version" link, confirm that it's the latest version.
                cy.get('[data-cy=collection-info-panel]').contains('head version').click();
                cy.get('[data-cy=collection-info-panel]')
                    .should('not.contain', 'This is an old version');
                cy.get('[data-cy=read-only-icon]').should('not.exist');
                cy.get('[data-cy=collection-version-number]').should('contain', '2');
                cy.get('[data-cy=collection-info-panel]').should('contain', colName);
                cy.get('[data-cy=collection-files-panel]').
                    should('not.contain', 'foo').and('contain', 'bar');

                // Check that old collection action isn't available on context menu
                cy.get('[data-cy=collection-panel-options-btn]').click()
                cy.get('[data-cy=context-menu]').should('not.contain', 'Restore version')
                cy.get('body').click(); // Collapse the menu avoiding details panel expansion

                // Make another change, confirm new version.
                cy.get('[data-cy=collection-panel-options-btn]').click();
                cy.get('[data-cy=context-menu]').contains('Edit collection').click();
                cy.get('[data-cy=form-dialog]')
                    .should('contain', 'Edit Collection')
                    .within(() => {
                        // appends some text
                        cy.get('input').first().type(' renamed');
                    });
                cy.get('[data-cy=form-submit-btn]').click();
                cy.get('[data-cy=collection-info-panel]')
                    .should('not.contain', 'This is an old version');
                cy.get('[data-cy=read-only-icon]').should('not.exist');
                cy.get('[data-cy=collection-version-number]').should('contain', '3');
                cy.get('[data-cy=collection-info-panel]').should('contain', colName + ' renamed');
                cy.get('[data-cy=collection-files-panel]')
                    .should('not.contain', 'foo').and('contain', 'bar');
                cy.get('[data-cy=collection-version-browser-select-3]')
                    .should('contain', '3').and('contain', '3 B');

                // Check context menus on version browser
                cy.get('[data-cy=collection-version-browser-select-3]').rightclick()
                cy.get('[data-cy=context-menu]')
                    .should('contain', 'Add to favorites')
                    .and('contain', 'Make a copy')
                    .and('contain', 'Edit collection');
                cy.get('body').click();
                // (and now an old version...)
                cy.get('[data-cy=collection-version-browser-select-1]').rightclick()
                cy.get('[data-cy=context-menu]')
                    .should('not.contain', 'Add to favorites')
                    .and('contain', 'Make a copy')
                    .and('not.contain', 'Edit collection');
                cy.get('body').click();

                // Restore first version
                cy.get('[data-cy=collection-version-browser]').within(() => {
                    cy.get('[data-cy=collection-version-browser-select-1]').click();
                });
                cy.get('[data-cy=collection-panel-options-btn]').click()
                cy.get('[data-cy=context-menu]').contains('Restore version').click();
                cy.get('[data-cy=confirmation-dialog]').should('contain', 'Restore version');
                cy.get('[data-cy=confirmation-dialog-ok-btn]').click();
                cy.get('[data-cy=collection-info-panel]')
                    .should('not.contain', 'This is an old version');
                cy.get('[data-cy=collection-version-number]').should('contain', '4');
                cy.get('[data-cy=collection-info-panel]').should('contain', colName);
                cy.get('[data-cy=collection-files-panel]')
                    .should('contain', 'foo').and('contain', 'bar');
            });
    });

    it('creates new collection on home project', function () {
        cy.loginAs(activeUser);
        cy.goToPath(`/projects/${activeUser.user.uuid}`);
        cy.get('[data-cy=breadcrumb-first]').should('contain', 'Projects');
        cy.get('[data-cy=breadcrumb-last]').should('not.exist');
        // Create new collection
        cy.get('[data-cy=side-panel-button]').click();
        cy.get('[data-cy=side-panel-new-collection]').click();
        // Name between brackets tests bugfix #17582
        const collName = `[Test collection (${Math.floor(999999 * Math.random())})]`;
        cy.get('[data-cy=form-dialog]')
            .should('contain', 'New collection')
            .and('contain', 'Storage classes')
            .and('contain', 'default')
            .and('contain', 'foo')
            .and('contain', 'bar')
            .within(() => {
                cy.get('[data-cy=parent-field]').within(() => {
                    cy.get('input').should('have.value', 'Home project');
                });
                cy.get('[data-cy=name-field]').within(() => {
                    cy.get('input').type(collName);
                });
                cy.get('[data-cy=checkbox-foo]').click();
            })
        cy.get('[data-cy=form-submit-btn]').click();
        // Confirm that the user was taken to the newly created thing
        cy.get('[data-cy=form-dialog]').should('not.exist');
        cy.get('[data-cy=breadcrumb-first]').should('contain', 'Projects');
        cy.get('[data-cy=breadcrumb-last]').should('contain', collName);
        cy.get('[data-cy=collection-info-panel]')
            .should('contain', 'default')
            .and('contain', 'foo')
            .and('not.contain', 'bar');
    });

    it('shows responsible person for collection if available', () => {
        cy.createCollection(adminUser.token, {
            name: `Test collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: activeUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n"
        })
            .as('testCollection1');

        cy.createCollection(adminUser.token, {
            name: `Test collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: adminUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n"
        })
            .as('testCollection2').then(function (testCollection2) {
                cy.shareWith(adminUser.token, activeUser.user.uuid, testCollection2.uuid, 'can_write');
            });

        cy.getAll('@testCollection1', '@testCollection2')
            .then(function ([testCollection1, testCollection2]) {
                cy.loginAs(activeUser);

                cy.goToPath(`/collections/${testCollection1.uuid}`);
                cy.get('[data-cy=responsible-person-wrapper]')
                    .contains(activeUser.user.uuid);

                cy.goToPath(`/collections/${testCollection2.uuid}`);
                cy.get('[data-cy=responsible-person-wrapper]')
                    .contains(adminUser.user.uuid);
            });
    });
})
