// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { kebabCase } from 'lodash';

const badgeLables = ['All', 'Failed', 'Cancelled', 'On hold', 'Queued', 'Running', 'Completed'];

describe('ProgressBadgeBar', () => {
    let activeUser;
    let adminUser;

    before(function () {
        cy.getUser('admin', 'Admin', 'User', true, true)
            .as('adminUser')
            .then(function () {
                adminUser = this.adminUser;
            });
        cy.getUser('activeuser', 'Active', 'User', false, true)
            .as('activeUser')
            .then(function () {
                activeUser = this.activeUser;
            });
    });

    it('should display progress badge bar with default views', () => {
        cy.loginAs(activeUser);
        cy.get('[data-cy=mpv-tabs]').contains('Workflow Runs').click();

        // remove any leftover processes
        cy.get('body').then(($body) => {
            if ($body.find('[data-cy=data-table-row]').length > 0) {
                cy.get('[data-cy=data-table-multiselect-popover]').click();
                cy.get('[data-cy=multiselect-popover-All]').click();
                cy.get('[data-title=Remove]').should('exist').click();
                cy.get('[data-cy=confirmation-dialog]').within(() => {
                    cy.get('[data-cy=confirmation-dialog-ok-btn]').click();
                });
                cy.waitForDom();
                cy.get('data-cy-data-table-row').should('not.exist');
            }
        });

        cy.get('[data-cy=progress-badge-bar]').should('exist');
        badgeLables.forEach((label) => {
            cy.get(`[data-cy=status-badge-sort-button-${kebabCase(label)}]`)
                .contains('(0)')
                .should('exist')
                .click();
            cy.get('[data-cy=default-view').contains('No workflow runs found').should('exist');
            cy.get('[data-cy=default-view')
                .contains('Filters are applied to the data.')
                .should(label === 'All' ? 'not.exist' : 'exist');
        });
    });
});