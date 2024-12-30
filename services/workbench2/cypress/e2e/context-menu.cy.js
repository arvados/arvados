// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { projectOrder, collectionOrder, workflowOrder } from 'views-components/context-menu/menu-item-sort';
import { ContextMenuActionNames } from 'views-components/context-menu/context-menu-action-set';

describe('ContextMenu', () => {
    let adminUser;
    let activeUser;

    before(function () {
        // Only set up common users once. These aren't set up as aliases because
        // aliases are cleaned up after every test. Also it doesn't make sense
        // to set the same users on beforeEach() over and over again, so we
        // separate a little from Cypress' 'Best Practices' here.
        cy.getUser('admin', 'Admin', 'User', true, true)
            .as('adminUser')
            .then(function () {
                adminUser = this.adminUser;
            });
        cy.getUser('user', 'Active', 'User', false, true)
            .as('activeUser')
            .then(function () {
                activeUser = this.activeUser;
            });
    });

    describe('Basic context menu tests', () => {
        it('opens context menu on right click and closes on click outside', () => {
            cy.createGroup(adminUser.token, {
                name: `my-context-menu-project-1`,
                group_class: 'project',
            });

            cy.loginAs(adminUser);

            cy.get('[data-cy="data-table-row"]').contains('my-context-menu-project-1').rightclick();
            cy.get('[data-cy="context-menu"]').then(($el) => {
                // Click outside the context menu
                const rect = $el[0].getBoundingClientRect();
                const x = rect.right + 300;
                const y = rect.top + rect.height / 2; // Vertically centered

                cy.get('body').click(x, y);
                cy.get('[data-cy="context-menu"]').should('not.exist');
            });
        });

        it('executes menu item action when clicked', () => {
            cy.createGroup(adminUser.token, {
                name: `my-context-menu-project-2`,
                group_class: 'project',
            });

            cy.loginAs(adminUser);
            // Set up intercept for the action

            cy.get('[data-cy="data-table-row"]').contains('my-context-menu-project-2').rightclick();
            cy.get('[data-cy="context-menu"]').contains('Share').click();

            // Verify the menu closed
            cy.get('[data-cy="context-menu"]').should('not.exist');
            // Verify the sharing dialog opened
            cy.get('[data-cy="sharing-dialog"]').should('be.visible');
        });
    });

    describe('Shows correct menu items', () => {
        it('shows correct Project menu items', () => {
            cy.createGroup(adminUser.token, {
                name: `my-context-menu-project-3`,
                group_class: 'project',
            });

            cy.loginAs(adminUser);

            // Right click on a project
            cy.get('[data-cy="data-table-row"]').contains('my-context-menu-project-3').rightclick();
            // Check for project-specific menu items
            cy.get('[data-cy="context-menu"]').within(() => {
                projectOrder.forEach((name) => {
                    if (name === ContextMenuActionNames.DIVIDER) return;
                    cy.contains(name).should('exist');
                });
            });
        });

        it('filters menu items based on user permissions', () => {
            cy.createGroup(activeUser.token, {
                name: `my-context-menu-project-4`,
                group_class: 'project',
            });

            // Test as non-admin user
            cy.loginAs(activeUser);

            cy.get('[data-cy="data-table-row"]').contains('my-context-menu-project-4').rightclick();
            cy.get('[data-cy="context-menu"]').within(() => {
                // Admin-only options should not be visible
                cy.contains('Add to public favorites').should('not.exist');
            });
        });

        it('shows correct Collection menu items', () => {
            cy.createCollection(adminUser.token, {
                name: `my-context-menu-collection`,
                owner_uuid: adminUser.uuid,
                manifest_text: '. 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n',
            });
            cy.loginAs(adminUser);

            // Right click on a project
            cy.get('[data-cy="data-table-row"]').contains('my-context-menu-collection').rightclick();
            // Check for project-specific menu items
            cy.get('[data-cy="context-menu"]').within(() => {
                collectionOrder.forEach((name) => {
                    if (name === ContextMenuActionNames.DIVIDER) return;
                    cy.contains(name).should('exist');
                });
            });
        });

        it('shows correct Workflow menu items', () => {
            cy.createWorkflow(adminUser.token, {
                name: `my-context-menu-workflow.cwl`,
                definition:
                    '{\n    "$graph": [\n        {\n            "class": "Workflow",\n            "doc": "Reverse the lines in a document, then sort those lines.",\n            "hints": [\n                {\n                    "acrContainerImage": "99b0201f4cade456b4c9d343769a3b70+261",\n                    "class": "http://arvados.org/cwl#WorkflowRunnerResources"\n                }\n            ],\n            "id": "#main",\n            "inputs": [\n                {\n                    "default": null,\n                    "doc": "The input file to be processed.",\n                    "id": "#main/input",\n                    "type": "File"\n                },\n                {\n                    "default": true,\n                    "doc": "If true, reverse (decending) sort",\n                    "id": "#main/reverse_sort",\n                    "type": "boolean"\n                }\n            ],\n            "outputs": [\n                {\n                    "doc": "The output with the lines reversed and sorted.",\n                    "id": "#main/output",\n                    "outputSource": "#main/sorted/output",\n                    "type": "File"\n                }\n            ],\n            "steps": [\n                {\n                    "id": "#main/rev",\n                    "in": [\n                        {\n                            "id": "#main/rev/input",\n                            "source": "#main/input"\n                        }\n                    ],\n                    "out": [\n                        "#main/rev/output"\n                    ],\n                    "run": "#revtool.cwl"\n                },\n                {\n                    "id": "#main/sorted",\n                    "in": [\n                        {\n                            "id": "#main/sorted/input",\n                            "source": "#main/rev/output"\n                        },\n                        {\n                            "id": "#main/sorted/reverse",\n                            "source": "#main/reverse_sort"\n                        }\n                    ],\n                    "out": [\n                        "#main/sorted/output"\n                    ],\n                    "run": "#sorttool.cwl"\n                }\n            ]\n        },\n        {\n            "baseCommand": "rev",\n            "class": "CommandLineTool",\n            "doc": "Reverse each line using the `rev` command",\n            "hints": [\n                {\n                    "class": "ResourceRequirement",\n                    "ramMin": 8\n                }\n            ],\n            "id": "#revtool.cwl",\n            "inputs": [\n                {\n                    "id": "#revtool.cwl/input",\n                    "inputBinding": {},\n                    "type": "File"\n                }\n            ],\n            "outputs": [\n                {\n                    "id": "#revtool.cwl/output",\n                    "outputBinding": {\n                        "glob": "output.txt"\n                    },\n                    "type": "File"\n                }\n            ],\n            "stdout": "output.txt"\n        },\n        {\n            "baseCommand": "sort",\n            "class": "CommandLineTool",\n            "doc": "Sort lines using the `sort` command",\n            "hints": [\n                {\n                    "class": "ResourceRequirement",\n                    "ramMin": 8\n                }\n            ],\n            "id": "#sorttool.cwl",\n            "inputs": [\n                {\n                    "id": "#sorttool.cwl/reverse",\n                    "inputBinding": {\n                        "position": 1,\n                        "prefix": "-r"\n                    },\n                    "type": "boolean"\n                },\n                {\n                    "id": "#sorttool.cwl/input",\n                    "inputBinding": {\n                        "position": 2\n                    },\n                    "type": "File"\n                }\n            ],\n            "outputs": [\n                {\n                    "id": "#sorttool.cwl/output",\n                    "outputBinding": {\n                        "glob": "output.txt"\n                    },\n                    "type": "File"\n                }\n            ],\n            "stdout": "output.txt"\n        }\n    ],\n    "cwlVersion": "v1.0"\n}',
                owner_uuid: adminUser.uuid,
            });

            cy.loginAs(adminUser);

            // Right click on a workflow
            cy.get('[data-cy="data-table-row"]').contains('my-context-menu-workflow.cwl').rightclick();
            // Check for project-specific menu items
            cy.get('[data-cy="context-menu"]').within(() => {
                workflowOrder.forEach((name) => {
                    if (name === ContextMenuActionNames.DIVIDER) return;
                    cy.contains(name).should('exist');
                });
            });
        });
    });
});
