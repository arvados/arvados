// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { kebabCase } from 'lodash';

const testWFDefinition =
    '{\n    "$graph": [\n        {\n            "class": "Workflow",\n            "doc": "Reverse the lines in a document, then sort those lines.",\n            "hints": [\n                {\n                    "acrContainerImage": "99b0201f4cade456b4c9d343769a3b70+261",\n                    "class": "http://arvados.org/cwl#WorkflowRunnerResources"\n                }\n            ],\n            "id": "#main",\n            "inputs": [\n                {\n                    "default": null,\n                    "doc": "The input file to be processed.",\n                    "id": "#main/input",\n                    "type": "File"\n                },\n                {\n                    "default": true,\n                    "doc": "If true, reverse (decending) sort",\n                    "id": "#main/reverse_sort",\n                    "type": "boolean"\n                }\n            ],\n            "outputs": [\n                {\n                    "doc": "The output with the lines reversed and sorted.",\n                    "id": "#main/output",\n                    "outputSource": "#main/sorted/output",\n                    "type": "File"\n                }\n            ],\n            "steps": [\n                {\n                    "id": "#main/rev",\n                    "in": [\n                        {\n                            "id": "#main/rev/input",\n                            "source": "#main/input"\n                        }\n                    ],\n                    "out": [\n                        "#main/rev/output"\n                    ],\n                    "run": "#revtool.cwl"\n                },\n                {\n                    "id": "#main/sorted",\n                    "in": [\n                        {\n                            "id": "#main/sorted/input",\n                            "source": "#main/rev/output"\n                        },\n                        {\n                            "id": "#main/sorted/reverse",\n                            "source": "#main/reverse_sort"\n                        }\n                    ],\n                    "out": [\n                        "#main/sorted/output"\n                    ],\n                    "run": "#sorttool.cwl"\n                }\n            ]\n        },\n        {\n            "baseCommand": "rev",\n            "class": "CommandLineTool",\n            "doc": "Reverse each line using the `rev` command",\n            "hints": [\n                {\n                    "class": "ResourceRequirement",\n                    "ramMin": 8\n                }\n            ],\n            "id": "#revtool.cwl",\n            "inputs": [\n                {\n                    "id": "#revtool.cwl/input",\n                    "inputBinding": {},\n                    "type": "File"\n                }\n            ],\n            "outputs": [\n                {\n                    "id": "#revtool.cwl/output",\n                    "outputBinding": {\n                        "glob": "output.txt"\n                    },\n                    "type": "File"\n                }\n            ],\n            "stdout": "output.txt"\n        },\n        {\n            "baseCommand": "sort",\n            "class": "CommandLineTool",\n            "doc": "Sort lines using the `sort` command",\n            "hints": [\n                {\n                    "class": "ResourceRequirement",\n                    "ramMin": 8\n                }\n            ],\n            "id": "#sorttool.cwl",\n            "inputs": [\n                {\n                    "id": "#sorttool.cwl/reverse",\n                    "inputBinding": {\n                        "position": 1,\n                        "prefix": "-r"\n                    },\n                    "type": "boolean"\n                },\n                {\n                    "id": "#sorttool.cwl/input",\n                    "inputBinding": {\n                        "position": 2\n                    },\n                    "type": "File"\n                }\n            ],\n            "outputs": [\n                {\n                    "id": "#sorttool.cwl/output",\n                    "outputBinding": {\n                        "glob": "output.txt"\n                    },\n                    "type": "File"\n                }\n            ],\n            "stdout": "output.txt"\n        }\n    ],\n    "cwlVersion": "v1.0"\n}';

const badgeLables = [
    'All',
    'Failed',
    'Cancelled',
    'On hold',
    'Queued',
    'Running',
    'Completed',
];

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
        cy.get('[data-cy=mpv-tabs]').contains("Workflow Runs").click();
        cy.get('[data-cy=progress-badge-bar]').should('exist');
        badgeLables.forEach(label => {
            cy.get(`[data-cy=status-badge-sort-button-${kebabCase(label)}]`).contains('(0)').should('exist').click();
            cy.get('[data-cy=default-view').contains('No workflow runs found').should('exist')
            cy.get('[data-cy=default-view').contains('Filters are applied to the data.').should(label === 'All' ? 'not.exist' : 'exist');
        });
    })
});
