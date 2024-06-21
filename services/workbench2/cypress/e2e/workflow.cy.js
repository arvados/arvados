// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

describe('Registered workflow panel tests', function() {
    let activeUser;
    let adminUser;

    before(function() {
        // Only set up common users once. These aren't set up as aliases because
        // aliases are cleaned up after every test. Also it doesn't make sense
        // to set the same users on beforeEach() over and over again, so we
        // separate a little from Cypress' 'Best Practices' here.
        cy.getUser('admin', 'Admin', 'User', true, true)
            .as('adminUser').then(function() {
                adminUser = this.adminUser;
            }
        );
        cy.getUser('user', 'Active', 'User', false, true)
            .as('activeUser').then(function() {
                activeUser = this.activeUser;
            }
        );
    });

    it('should handle null definition', function() {
        cy.createResource(activeUser.token, "workflows", {workflow: {name: "Test wf"}})
            .then(function(workflowResource) {
                cy.loginAs(activeUser);
                cy.goToPath(`/workflows/${workflowResource.uuid}`);
                cy.get('[data-cy=registered-workflow-info-panel]').should('contain', workflowResource.name);
                cy.get('[data-cy=workflow-details-attributes-modifiedby-user]').contains(`Active User (${activeUser.user.uuid})`);
            });
    });

    it('should handle malformed definition', function() {
        cy.createResource(activeUser.token, "workflows", {workflow: {name: "Test wf", definition: "zap:"}})
            .then(function(workflowResource) {
                cy.loginAs(activeUser);
                cy.goToPath(`/workflows/${workflowResource.uuid}`);
                cy.get('[data-cy=registered-workflow-info-panel]').should('contain', workflowResource.name);
                cy.get('[data-cy=workflow-details-attributes-modifiedby-user]').contains(`Active User (${activeUser.user.uuid})`);
            });
    });

    it('should handle malformed run', function() {
        cy.createResource(activeUser.token, "workflows", {workflow: {
            name: "Test wf",
            definition: JSON.stringify({
                cwlVersion: "v1.2",
                $graph: [
                    {
                        "class": "Workflow",
                        "id": "#main",
                        "inputs": [],
                        "outputs": [],
                        "requirements": [
                            {
                                "class": "SubworkflowFeatureRequirement"
                            }
                        ],
                        "steps": [
                            {
                                "id": "#main/cat1-testcli.cwl (v1.2.0-109-g9b091ed)",
                                "in": [],
                                "label": "cat1-testcli.cwl (v1.2.0-109-g9b091ed)",
                                "out": [
                                    {
                                        "id": "#main/step/args"
                                    }
                                ],
                                "run": `keep:undefined/bar`
                            }
                        ]
                    }
                ],
                "cwlVersion": "v1.2",
                "http://arvados.org/cwl#gitBranch": "1.2.1_proposed",
                "http://arvados.org/cwl#gitCommit": "9b091ed7e0bef98b3312e9478c52b89ba25792de",
                "http://arvados.org/cwl#gitCommitter": "GitHub <noreply@github.com>",
                "http://arvados.org/cwl#gitDate": "Sun, 11 Sep 2022 21:24:42 +0200",
                "http://arvados.org/cwl#gitDescribe": "v1.2.0-109-g9b091ed",
                "http://arvados.org/cwl#gitOrigin": "git@github.com:common-workflow-language/cwl-v1.2",
                "http://arvados.org/cwl#gitPath": "tests/cat1-testcli.cwl",
                "http://arvados.org/cwl#gitStatus": ""
            })
        }}).then(function(workflowResource) {
            cy.loginAs(activeUser);
            cy.goToPath(`/workflows/${workflowResource.uuid}`);
            cy.get('[data-cy=registered-workflow-info-panel]').should('contain', workflowResource.name);
            cy.get('[data-cy=workflow-details-attributes-modifiedby-user]').contains(`Active User (${activeUser.user.uuid})`);
        });
    });

    const verifyIOParameter = (name, label, doc, val, collection) => {
        cy.get('table tr').contains(name).parents('tr').within(($mainRow) => {
            label && cy.contains(label);

            if (val) {
                if (Array.isArray(val)) {
                    val.forEach(v => cy.contains(v));
                } else {
                    cy.contains(val);
                }
            }
            if (collection) {
                cy.contains(collection);
            }
        });
    };

    it('shows workflow details', function() {
        cy.createCollection(adminUser.token, {
            name: `Test collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: activeUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n"
        })
            .then(function(collectionResource) {
                cy.createResource(activeUser.token, "workflows", {workflow: {
                    name: "Test wf",
                    definition: JSON.stringify({
                        cwlVersion: "v1.2",
                        $graph: [
                            {
                                "class": "Workflow",
                                "hints": [
                                    {
                                        "class": "DockerRequirement",
                                        "dockerPull": "python:2-slim"
                                    }
                                ],
                                "id": "#main",
                                "inputs": [
                                    {
                                        "id": "#main/file1",
                                        "type": "File"
                                    },
                                    {
                                        "id": "#main/numbering",
                                        "type": [
                                            "null",
                                            "boolean"
                                        ]
                                    },
                                    {
                                        "default": {
                                            "basename": "args.py",
                                            "class": "File",
                                            "location": "keep:de738550734533c5027997c87dc5488e+53/args.py",
                                            "nameext": ".py",
                                            "nameroot": "args",
                                            "size": 179
                                        },
                                        "id": "#main/args.py",
                                        "type": "File"
                                    }
                                ],
                                "outputs": [
                                    {
                                        "id": "#main/args",
                                        "outputSource": "#main/step/args",
                                        "type": {
                                            "items": "string",
                                            "name": "_:b0adccc1-502d-476f-8a5b-c8ef7119e2dc",
                                            "type": "array"
                                        }
                                    }
                                ],
                                "requirements": [
                                    {
                                        "class": "SubworkflowFeatureRequirement"
                                    }
                                ],
                                "steps": [
                                    {
                                        "id": "#main/cat1-testcli.cwl (v1.2.0-109-g9b091ed)",
                                        "in": [
                                            {
                                                "id": "#main/step/file1",
                                                "source": "#main/file1"
                                            },
                                            {
                                                "id": "#main/step/numbering",
                                                "source": "#main/numbering"
                                            },
                                            {
                                                "id": "#main/step/args.py",
                                                "source": "#main/args.py"
                                            }
                                        ],
                                        "label": "cat1-testcli.cwl (v1.2.0-109-g9b091ed)",
                                        "out": [
                                            {
                                                "id": "#main/step/args"
                                            }
                                        ],
                                        "run": `keep:${collectionResource.portable_data_hash}/bar`
                                    }
                                ]
                            }
                        ],
                        "cwlVersion": "v1.2",
                        "http://arvados.org/cwl#gitBranch": "1.2.1_proposed",
                        "http://arvados.org/cwl#gitCommit": "9b091ed7e0bef98b3312e9478c52b89ba25792de",
                        "http://arvados.org/cwl#gitCommitter": "GitHub <noreply@github.com>",
                        "http://arvados.org/cwl#gitDate": "Sun, 11 Sep 2022 21:24:42 +0200",
                        "http://arvados.org/cwl#gitDescribe": "v1.2.0-109-g9b091ed",
                        "http://arvados.org/cwl#gitOrigin": "git@github.com:common-workflow-language/cwl-v1.2",
                        "http://arvados.org/cwl#gitPath": "tests/cat1-testcli.cwl",
                        "http://arvados.org/cwl#gitStatus": ""
                    })
                }}).then(function(workflowResource) {
                    cy.loginAs(activeUser);
                    cy.goToPath(`/workflows/${workflowResource.uuid}`);
                    cy.get('[data-cy=registered-workflow-info-panel]').should('contain', workflowResource.name);
                    cy.get('[data-cy=workflow-details-attributes-modifiedby-user]').contains(`Active User (${activeUser.user.uuid})`);
                    cy.get('[data-cy=registered-workflow-info-panel')
                        .should('contain', 'gitCommit: 9b091ed7e0bef98b3312e9478c52b89ba25792de')

                    cy.get('[data-cy=process-io-card] h6').contains('Input Parameters')
                        .parents('[data-cy=process-io-card]').within(() => {
                            verifyIOParameter('file1', null, '', '', '');
                            verifyIOParameter('numbering', null, '', '', '');
                            verifyIOParameter('args.py', null, '', 'args.py', 'de738550734533c5027997c87dc5488e+53');
                        });
                    cy.get('[data-cy=process-io-card] h6').contains('Output Parameters')
                        .parents('[data-cy=process-io-card]').within(() => {
                            verifyIOParameter('args', null, '', '', '');
                        });
                    cy.get('[data-cy=collection-files-panel]').within(() => {
                        cy.get('[data-cy=collection-files-right-panel]', { timeout: 5000 })
                            .should('contain', 'bar');
                    });
                });
            });
    });

    it('can delete a workflow', function() {
        cy.createResource(activeUser.token, "workflows", {workflow: {name: "Test wf"}})
            .then(function(workflowResource) {
                cy.loginAs(activeUser);
                cy.goToPath(`/projects/${activeUser.user.uuid}`);
                cy.get('[data-cy=project-panel] table tbody').contains(workflowResource.name).rightclick();
                cy.get('[data-cy=context-menu]').contains('Delete Workflow').click();
                cy.get('[data-cy=confirmation-dialog-ok-btn]').should('exist').click();
                cy.get('[data-cy=project-panel] table tbody').should('not.contain', workflowResource.name);
            });
    });

    it('can delete multiple workflows', function() {
        const wfNames = ["Test wf1", "Test wf2", "Test wf3"];

        wfNames.forEach((wfName) => {
            cy.createResource(activeUser.token, "workflows", {workflow: {name: wfName}})
        });
        
        cy.loginAs(activeUser);

        wfNames.forEach((wfName) => {
            cy.get('tr').contains('td', wfName).should('exist').parent('tr').find('input[type="checkbox"]').click();
        });
        
        cy.waitForDom().get('[data-cy=multiselect-button]', {timeout: 10000}).should('be.visible')
        cy.get('[data-cy=multiselect-button]', {timeout: 10000}).should('have.length', '1').trigger('mouseover');
        cy.get('body').contains('Delete Workflow', {timeout: 10000}).should('exist')
        cy.get('[data-cy=multiselect-button]').eq(0).click();
        cy.get('[data-cy=confirmation-dialog-ok-btn]').should('exist').click();

        wfNames.forEach((wfName) => {
            cy.get('tr').contains(wfName).should('not.exist');
        });
    });

    it('cannot delete readonly workflow', function() {
        cy.createProject({
            owningUser: adminUser,
            targetUser: activeUser,
            projectName: 'mySharedReadonlyProject',
            canWrite: false,
        });
        cy.getAll('@mySharedReadonlyProject')
            .then(function ([mySharedReadonlyProject]) {
                cy.createResource(adminUser.token, "workflows", {workflow: {name: "Test wf", owner_uuid: mySharedReadonlyProject.uuid}})
                    .then(function(workflowResource) {
                        cy.loginAs(activeUser);
                        cy.goToPath(`/shared-with-me`);
                        cy.contains("mySharedReadonlyProject").click();
                        cy.get('[data-cy=project-panel] table tbody').contains(workflowResource.name).rightclick();
                        cy.get('[data-cy=context-menu]').should("not.contain", 'Delete Workflow');
                    });
            });
    });

    it('shows the appropriate buttons in the multiselect toolbar', () => {

        const msButtonTooltips = [
            'View details',
            'Open in new tab',
            'Copy link to clipboard',
            'API Details',
            'Run Workflow',
            'Delete Workflow',
        ];

        cy.createResource(activeUser.token, "workflows", {workflow: {name: "Test wf"}})
            .then(function(workflowResource) {
                cy.loginAs(activeUser);
                cy.get("[data-cy=side-panel-tree]").contains("Home Projects").click();
                cy.waitForDom()
                cy.get('[data-cy=data-table-row]').contains(workflowResource.name).should('exist').parent().parent().parent().click()
                cy.get('[data-cy=multiselect-button]').should('have.length', msButtonTooltips.length)
                for (let i = 0; i < msButtonTooltips.length; i++) {
                        cy.get('[data-cy=multiselect-button]').eq(i).trigger('mouseover');
                        cy.get('body').contains(msButtonTooltips[i]).should('exist')
                        cy.get('[data-cy=multiselect-button]').eq(i).trigger('mouseout');
                    }
                });
    })

});
