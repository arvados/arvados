// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

describe('Details panel', () => {
  let adminUser;

  before(() => {
    cy.getUser("active", "Active", "User", true, true)
      .as("activeUser")
      .then((user) => {
        adminUser = user;
      });
  });

  // Add this test to the existing describe block in details-panel.cy.js

  it('displays root project details when no items are selected', () => {
    cy.loginAs(adminUser);

    // Navigate to the user's root project
    cy.visit(`/projects/${adminUser.user.uuid}`);

    // Wait for the data table to load
    cy.get('[data-cy=data-table]').should('be.visible');

    // Ensure no items are selected
    cy.get('[data-cy=data-table-row] input[type="checkbox"]:checked').should('not.exist');

    // Open the details panel
    cy.get('[data-cy=details-panel]').should('not.exist');
    cy.get('[data-testid=InfoIcon]').click();
    cy.get('[data-cy=details-panel]').should('be.visible');

    // Check if root project details are displayed
    cy.get('[data-cy=details-panel]').within(() => {
      cy.contains('Type').should('be.visible');
      cy.contains('Root Project').should('be.visible');
      cy.contains('User').should('be.visible');
      cy.contains('Created at').should('be.visible');
      cy.contains('UUID').should('be.visible');

      // Verify specific root project details
      cy.contains(adminUser.user.uuid).should('be.visible');
    });

    // Verify that the Root Project icon is displayed
    cy.get('[data-cy=details-panel]').find('[data-testid=InboxIcon]').should('be.visible');
  });
});

describe('Collection details panel', () => {
  let adminUser;

  before(() => {
    cy.getUser("active", "Active", "User", true, true)
      .as("activeUser")
      .then((user) => {
        adminUser = user;
      });
  });

  it('displays appropriate attributes when a collection is selected', () => {
    cy.loginAs(adminUser);

    // Create a test collection
    const collectionName = `Test Collection ${Math.floor(Math.random() * 999999)}`;
    cy.createCollection(adminUser.token, {
      name: collectionName,
      owner_uuid: adminUser.user.uuid,
      manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:foo\n",
    }).as('testCollection');

    // Navigate to the project containing the collection
    cy.get('@testCollection').then((collection) => {
      cy.visit(`/projects/${adminUser.user.uuid}`);

      // Wait for the data table to load
      cy.get('[data-cy=data-table]').should('be.visible');

      // Find and check the checkbox for the test collection
      cy.contains('[data-cy=data-table-row]', collectionName)
        .find('input[type="checkbox"]')
        .click();

      // Open the details panel
      cy.get('[data-cy=details-panel]').should('not.exist');
      cy.get('[data-testid=InfoIcon]').click();
      cy.get('[data-cy=details-panel]').should('be.visible');

      // Check if appropriate attributes are displayed
      cy.get('[data-cy=details-panel]').within(() => {
        cy.contains('Collection UUID').should('be.visible');
        cy.contains('Portable data hash').should('be.visible');
        cy.contains('Owner').should('be.visible');
        cy.contains('Created at').should('be.visible');
        cy.contains('Last modified').should('be.visible');
        cy.contains('Content size').should('be.visible');
        cy.contains('Number of files').should('be.visible');
        cy.contains('Properties').should('be.visible');
      });

      // Verify specific collection details
      cy.get('[data-cy=details-panel]').within(() => {
        cy.contains(collection.uuid).should('be.visible');
        cy.contains(collection.portable_data_hash).should('be.visible');
        cy.contains(adminUser.user.uuid).should('be.visible');
        cy.contains('1').should('be.visible'); // Number of files
        cy.contains('3 B').should('be.visible'); // Content size
      });
    });
  });

  describe('Collection versioning', () => {
    let adminUser;
  
    before(() => {
      cy.getUser("active", "Active", "User", true, true)
        .as("activeUser")
        .then((user) => {
          adminUser = user;
        });
    });
  
    it('creates a collection, edits it, and verifies version information', () => {
      cy.loginAs(adminUser);
  
      // Create a test collection
      const collectionName = `Test Collection ${Math.floor(Math.random() * 999999)}`;
      cy.createCollection(adminUser.token, {
        name: collectionName,
        owner_uuid: adminUser.user.uuid,
        manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:foo\n",
      }).as('testCollection');
  
      cy.get('@testCollection').then((collection) => {
        // Navigate to the project containing the collection
        cy.visit(`/projects/${adminUser.user.uuid}`);
  
        // Wait for the data table to load
        cy.get('[data-cy=data-table]').should('be.visible');
  
        // Find and open the test collection
        cy.contains('[data-cy=data-table-row]', collectionName).rightclick();
  
        // Edit the collection
        cy.get("[data-cy=context-menu]").within(() => {
          cy.get('[data-cy="Edit collection"]').click();
        });
  
        // Change the name in the edit dialog
        const newName = `${collectionName} (edited)`;
        cy.get('[data-cy=form-dialog]').within(() => {
          cy.get('input[name=name]').clear().type(newName);
          cy.get('[data-cy=form-submit-btn]').click();
        });
  
        // Wait for the update to complete
        cy.contains('[data-cy=data-table]', newName).should('be.visible');

        // open the collection viewer
        cy.contains(newName).click();
  
        // Verify that the version number has increased
        cy.get('[data-cy=collection-version-number]').should('contain', '2');
  
        // Click on the version number to open the details panel
        cy.get('[data-cy=collection-version-number]').click();
  
        // Verify that the details panel is open and the "Versions" tab is selected
        cy.get('[data-cy=details-panel]').should('be.visible');
        cy.get('[data-cy=details-panel-tab-Versions]').should('have.attr', 'aria-selected', 'true');
  
        // Verify that the version number is visible in the details panel
        cy.get('[data-cy=collection-version-browser]').within(() => {
          cy.get('[data-cy=collection-version-browser-select-2]').should('be.visible');
          cy.get('[data-cy=collection-version-browser-select-2]').should('have.class', 'Mui-selected');
        });
      });
    });
  });
});