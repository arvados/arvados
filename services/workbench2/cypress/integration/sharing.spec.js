// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

describe('Sharing tests', function () {
    let activeUser;
    let adminUser;

    before(function () {
        // Only set up common users once. These aren't set up as aliases because
        // aliases are cleaned up after every test. Also it doesn't make sense
        // to set the same users on beforeEach() over and over again, so we
        // separate a little from Cypress' 'Best Practices' here.
        cy.getUser('admin', 'Admin', 'User', true, true)
            .as('adminUser').then(function () {
                adminUser = this.adminUser;
            });
        cy.getUser('collectionuser1', 'Collection', 'User', false, true)
            .as('activeUser').then(function () {
                activeUser = this.activeUser;
            });
    })

    beforeEach(function () {
        cy.clearCookies()
        cy.clearLocalStorage()
    });

    it('can create and delete sharing URLs on collections', () => {
        const collName = 'shared-collection ' + new Date().getTime();
        cy.createCollection(adminUser.token, {
            name: collName,
            owner_uuid: adminUser.uuid,
        }).as('sharedCollection').then(function (sharedCollection) {
            cy.loginAs(adminUser);

            cy.get('main').contains(sharedCollection.name).rightclick();
            cy.get('[data-cy=context-menu]').within(() => {
                cy.contains('Share').click();
            });
            cy.get('.sharing-dialog').within(() => {
                cy.contains('Sharing URLs').click();
                cy.contains('Create sharing URL');
                cy.contains('No sharing URLs');
                cy.should('not.contain', 'Token');
                cy.should('not.contain', 'expiring at:');

                cy.contains('Create sharing URL').click();
                cy.should('not.contain', 'No sharing URLs');
                cy.contains('Token');
                cy.contains('expiring at:');

                cy.get('[data-cy=remove-url-btn]').find('button').click();
                cy.contains('No sharing URLs');
                cy.should('not.contain', 'Token');
                cy.should('not.contain', 'expiring at:');
            })
        })
    });

    it('can share projects to other users', () => {
        cy.loginAs(adminUser);

        cy.createGroup(adminUser.token, {
            name: `my-shared-writable-project ${Math.floor(Math.random() * 999999)}`,
            group_class: 'project',
        }).as('mySharedWritableProject').then(function (mySharedWritableProject) {
            cy.contains('Refresh').click();
            cy.get('main').contains(mySharedWritableProject.name).rightclick();
            cy.get('[data-cy=context-menu]').within(() => {
                cy.contains('Share').click();
            });
            cy.get('[id="select-permissions"]').as('selectPermissions');
            cy.get('@selectPermissions').click();
            cy.contains('Write').click();
            cy.get('.sharing-dialog').as('sharingDialog');
            cy.get('[data-cy=invite-people-field]').find('input').type(activeUser.user.email);
            cy.get('[role=tooltip]').click();
            cy.get('@sharingDialog').within(() => {
                cy.get('[data-cy=add-invited-people]').click();
                cy.contains('Close').click();
            });
        });

        cy.createGroup(adminUser.token, {
            name: `my-shared-readonly-project ${Math.floor(Math.random() * 999999)}`,
            group_class: 'project',
        }).as('mySharedReadonlyProject').then(function (mySharedReadonlyProject) {
            cy.contains('Refresh').click();
            cy.get('main').contains(mySharedReadonlyProject.name).rightclick();
            cy.get('[data-cy=context-menu]').within(() => {
                cy.contains('Share').click();
            });
            cy.get('.sharing-dialog').as('sharingDialog');
            cy.get('[data-cy=invite-people-field]').find('input').type(activeUser.user.email);
            cy.get('[role=tooltip]').click();
            cy.get('@sharingDialog').within(() => {
                cy.get('[data-cy=add-invited-people]').click();
                cy.contains('Close').click();
            });
        });

        cy.getAll('@mySharedWritableProject', '@mySharedReadonlyProject')
            .then(function ([mySharedWritableProject, mySharedReadonlyProject]) {
                cy.loginAs(activeUser);

                cy.contains('Shared with me').click();

                // Test search
                cy.get('[data-cy=search-input] input').type('readonly');
                cy.get('main').should('not.contain', mySharedWritableProject.name);
                cy.get('main').should('contain', mySharedReadonlyProject.name);
                cy.get('[data-cy=search-input] input').clear();

                // Test filter
                cy.waitForDom().get('th').contains('Type').click();
                cy.get('div[role=presentation]').contains('Project').click();
                cy.waitForDom().get('main table tr td').contains('Project').should('not.exist');
                cy.get('div[role=presentation]').contains('Project').click();
                cy.waitForDom().get('div[role=presentation] button').contains('Close').click();

                // Test move to trash
                cy.get('main').contains(mySharedWritableProject.name).rightclick();
                cy.get('[data-cy=context-menu]').should('contain', 'Move to trash');
                cy.get('[data-cy=context-menu]').contains('Move to trash').click();

                // GUARD: Let's wait for the above removed project to disappear
                // before continuing, to avoid intermittent failures.
                cy.get('main').should('not.contain', mySharedWritableProject.name);

                cy.get('main').contains(mySharedReadonlyProject.name).rightclick();
                cy.get('[data-cy=context-menu]').should('not.contain', 'Move to trash');
            });
    });

    it('can edit project in shared with me', () => {
        cy.createProject({
            owningUser: adminUser,
            targetUser: activeUser,
            projectName: 'mySharedWritableProject',
            canWrite: true,
            addToFavorites: true
        });

        cy.getAll('@mySharedWritableProject')
            .then(function ([mySharedWritableProject]) {
                cy.loginAs(activeUser);

                cy.get('[data-cy=side-panel-tree]').contains('Shared with me').click();

                const newProjectName = `New project name ${mySharedWritableProject.name}`;
                const newProjectDescription = `New project description ${mySharedWritableProject.name}`;

                cy.testEditProjectOrCollection('main', mySharedWritableProject.name, newProjectName, newProjectDescription);
            });
    });

    it('can share only when target users are present', () => {
        const collName = `mySharedCollectionForUsers-${new Date().getTime()}`;
        cy.createCollection(adminUser.token, {
            name: collName,
            owner_uuid: adminUser.uuid,
        }).as('mySharedCollectionForUsers')

        cy.getAll('@mySharedCollectionForUsers')
            .then(function ([]) {
                cy.loginAs(adminUser);
                cy.get('[data-cy=project-panel]').contains(collName).rightclick();
                cy.get('[data-cy=context-menu]').contains('Share').click();
                cy.get('button').get('[data-cy=add-invited-people]').should('be.disabled');
                cy.get('[data-cy=invite-people-field] input').type('Anonymous');
                cy.get('div[role=tooltip]').contains('anonymous').click();
                cy.get('button').get('[data-cy=add-invited-people]').should('not.be.disabled');
                cy.get('[data-cy=invite-people-field] div[role=button]').contains('anonymous').parent().find('svg').click();
                cy.get('button').get('[data-cy=add-invited-people]').should('be.disabled');
            });
    });
});
