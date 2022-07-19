// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

describe('Process tests', function() {
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

    beforeEach(function() {
        cy.clearCookies();
        cy.clearLocalStorage();
    });

    function setupDockerImage(image_name) {
        // Create a collection that will be used as a docker image for the tests.
        cy.createCollection(adminUser.token, {
            name: 'docker_image',
            manifest_text: ". d21353cfe035e3e384563ee55eadbb2f+67108864 5c77a43e329b9838cbec18ff42790e57+55605760 0:122714624:sha256:d8309758b8fe2c81034ffc8a10c36460b77db7bc5e7b448c4e5b684f9d95a678.tar\n"
        }).as('dockerImage').then(function(dockerImage) {
            // Give read permissions to the active user on the docker image.
            cy.createLink(adminUser.token, {
                link_class: 'permission',
                name: 'can_read',
                tail_uuid: activeUser.user.uuid,
                head_uuid: dockerImage.uuid
            }).as('dockerImagePermission').then(function() {
                // Set-up docker image collection tags
                cy.createLink(activeUser.token, {
                    link_class: 'docker_image_repo+tag',
                    name: image_name,
                    head_uuid: dockerImage.uuid,
                }).as('dockerImageRepoTag');
                cy.createLink(activeUser.token, {
                    link_class: 'docker_image_hash',
                    name: 'sha256:d8309758b8fe2c81034ffc8a10c36460b77db7bc5e7b448c4e5b684f9d95a678',
                    head_uuid: dockerImage.uuid,
                }).as('dockerImageHash');
            })
        });
        return cy.getAll('@dockerImage', '@dockerImageRepoTag', '@dockerImageHash',
            '@dockerImagePermission').then(function([dockerImage]) {
                return dockerImage;
            });
    }

    function createContainerRequest(user, name, docker_image, command, reuse = false, state = 'Uncommitted') {
        return setupDockerImage(docker_image).then(function(dockerImage) {
            return cy.createContainerRequest(user.token, {
                name: name,
                command: command,
                container_image: dockerImage.portable_data_hash, // for some reason, docker_image doesn't work here
                output_path: 'stdout.txt',
                priority: 1,
                runtime_constraints: {
                    vcpus: 1,
                    ram: 1,
                },
                use_existing: reuse,
                state: state,
                mounts: {
                    foo: {
                        kind: 'tmp',
                        path: '/tmp/foo',
                    }
                }
            });
        });
    }

    it('shows process logs', function() {
        const crName = 'test_container_request';
        createContainerRequest(
            activeUser,
            crName,
            'arvados/jobs',
            ['echo', 'hello world'],
            false, 'Committed')
        .then(function(containerRequest) {
            cy.loginAs(activeUser);
            cy.goToPath(`/processes/${containerRequest.uuid}`);
            cy.get('[data-cy=process-details]').should('contain', crName);
            cy.get('[data-cy=process-logs]')
                .should('contain', 'No logs yet')
                .and('not.contain', 'hello world');
            cy.createLog(activeUser.token, {
                object_uuid: containerRequest.container_uuid,
                properties: {
                    text: 'hello world'
                },
                event_type: 'stdout'
            }).then(function(log) {
                cy.get('[data-cy=process-logs]')
                    .should('not.contain', 'No logs yet')
                    .and('contain', 'hello world');
            })
        });
    });

    it('filters process logs by event type', function() {
        const nodeInfoLogs = [
            'Host Information',
            'Linux compute-99cb150b26149780de44b929577e1aed-19rgca8vobuvc4p 5.4.0-1059-azure #62~18.04.1-Ubuntu SMP Tue Sep 14 17:53:18 UTC 2021 x86_64 x86_64 x86_64 GNU/Linux',
            'CPU Information',
            'processor  : 0',
            'vendor_id  : GenuineIntel',
            'cpu family : 6',
            'model      : 79',
            'model name : Intel(R) Xeon(R) CPU E5-2673 v4 @ 2.30GHz'
        ];
        const crunchRunLogs = [
            '2022-03-22T13:56:22.542417997Z using local keepstore process (pid 3733) at http://localhost:46837, writing logs to keepstore.txt in log collection',
            '2022-03-22T13:56:26.237571754Z crunch-run 2.4.0~dev20220321141729 (go1.17.1) started',
            '2022-03-22T13:56:26.244704134Z crunch-run process has uid=0(root) gid=0(root) groups=0(root)',
            '2022-03-22T13:56:26.244862836Z Executing container \'zzzzz-dz642-1wokwvcct9s9du3\' using docker runtime',
            '2022-03-22T13:56:26.245037738Z Executing on host \'compute-99cb150b26149780de44b929577e1aed-19rgca8vobuvc4p\'',
        ];
        const stdoutLogs = [
            'Lorem ipsum dolor sit amet, consectetur adipiscing elit. Donec dui nisi, hendrerit porta sapien a, pretium dignissim purus.',
            'Integer viverra, mauris finibus aliquet ultricies, dui mauris cursus justo, ut venenatis nibh ex eget neque.',
            'In hac habitasse platea dictumst.',
            'Fusce fringilla turpis id accumsan faucibus. Donec congue congue ex non posuere. In semper mi quis tristique rhoncus.',
            'Interdum et malesuada fames ac ante ipsum primis in faucibus.',
            'Quisque fermentum tortor ex, ut suscipit velit feugiat faucibus.',
            'Donec vitae porta risus, at luctus nulla. Mauris gravida iaculis ipsum, id sagittis tortor egestas ac.',
            'Maecenas condimentum volutpat nulla. Integer lacinia maximus risus eu posuere.',
            'Donec vitae leo id augue gravida bibendum.',
            'Nam libero libero, pretium ac faucibus elementum, mattis nec ex.',
            'Nullam id laoreet nibh. Vivamus tellus metus, pretium quis justo ut, bibendum varius metus. Pellentesque vitae accumsan lorem, quis tincidunt augue.',
            'Aliquam viverra nisi nulla, et efficitur dolor mattis in.',
            'Sed at enim sit amet nulla tincidunt mattis. Aenean eget aliquet ex, non ultrices ex. Nulla ex tortor, vestibulum aliquam tempor ac, aliquam vel est.',
            'Fusce auctor faucibus libero id venenatis. Etiam sodales, odio eu cursus efficitur, quam sem blandit ex, quis porttitor enim dui quis lectus. In id tincidunt felis.',
            'Phasellus non ex quis arcu tempus faucibus molestie in sapien.',
            'Duis tristique semper dolor, vitae pulvinar risus.',
            'Aliquam tortor elit, luctus nec tortor eget, porta tristique nulla.',
            'Nulla eget mollis ipsum.',
        ];

        createContainerRequest(
            activeUser,
            'test_container_request',
            'arvados/jobs',
            ['echo', 'hello world'],
            false, 'Committed')
        .then(function(containerRequest) {
            cy.logsForContainer(activeUser.token, containerRequest.container_uuid,
                'node-info', nodeInfoLogs).as('nodeInfoLogs');
            cy.logsForContainer(activeUser.token, containerRequest.container_uuid,
                'crunch-run', crunchRunLogs).as('crunchRunLogs');
            cy.logsForContainer(activeUser.token, containerRequest.container_uuid,
                'stdout', stdoutLogs).as('stdoutLogs');
            cy.getAll('@stdoutLogs', '@nodeInfoLogs', '@crunchRunLogs').then(function() {
                cy.loginAs(activeUser);
                cy.goToPath(`/processes/${containerRequest.uuid}`);
                // Should show main logs by default
                cy.get('[data-cy=process-logs-filter]').should('contain', 'Main logs');
                cy.get('[data-cy=process-logs]')
                    .should('contain', stdoutLogs[Math.floor(Math.random() * stdoutLogs.length)])
                    .and('not.contain', nodeInfoLogs[Math.floor(Math.random() * nodeInfoLogs.length)])
                    .and('contain', crunchRunLogs[Math.floor(Math.random() * crunchRunLogs.length)]);
                // Select 'All logs'
                cy.get('[data-cy=process-logs-filter]').click();
                cy.get('body').contains('li', 'All logs').click();
                cy.get('[data-cy=process-logs]')
                    .should('contain', stdoutLogs[Math.floor(Math.random() * stdoutLogs.length)])
                    .and('contain', nodeInfoLogs[Math.floor(Math.random() * nodeInfoLogs.length)])
                    .and('contain', crunchRunLogs[Math.floor(Math.random() * crunchRunLogs.length)]);
                // Select 'node-info' logs
                cy.get('[data-cy=process-logs-filter]').click();
                cy.get('body').contains('li', 'node-info').click();
                cy.get('[data-cy=process-logs]')
                    .should('not.contain', stdoutLogs[Math.floor(Math.random() * stdoutLogs.length)])
                    .and('contain', nodeInfoLogs[Math.floor(Math.random() * nodeInfoLogs.length)])
                    .and('not.contain', crunchRunLogs[Math.floor(Math.random() * crunchRunLogs.length)]);
                // Select 'stdout' logs
                cy.get('[data-cy=process-logs-filter]').click();
                cy.get('body').contains('li', 'stdout').click();
                cy.get('[data-cy=process-logs]')
                    .should('contain', stdoutLogs[Math.floor(Math.random() * stdoutLogs.length)])
                    .and('not.contain', nodeInfoLogs[Math.floor(Math.random() * nodeInfoLogs.length)])
                    .and('not.contain', crunchRunLogs[Math.floor(Math.random() * crunchRunLogs.length)]);
            });
        });
    });

    it('should show runtime status indicators', function() {
        // Setup running container with runtime_status error & warning messages
        createContainerRequest(
            activeUser,
            'test_container_request',
            'arvados/jobs',
            ['echo', 'hello world'],
            false, 'Committed')
        .as('containerRequest')
        .then(function(containerRequest) {
            expect(containerRequest.state).to.equal('Committed');
            expect(containerRequest.container_uuid).not.to.be.equal('');

            cy.getContainer(activeUser.token, containerRequest.container_uuid)
            .then(function(queuedContainer) {
                expect(queuedContainer.state).to.be.equal('Queued');
            });
            cy.updateContainer(adminUser.token, containerRequest.container_uuid, {
                state: 'Locked'
            }).then(function(lockedContainer) {
                expect(lockedContainer.state).to.be.equal('Locked');

                cy.updateContainer(adminUser.token, lockedContainer.uuid, {
                    state: 'Running',
                    runtime_status: {
                        error: 'Something went wrong',
                        errorDetail: 'Process exited with status 1',
                        warning: 'Free disk space is low',
                    }
                })
                .as('runningContainer')
                .then(function(runningContainer) {
                    expect(runningContainer.state).to.be.equal('Running');
                    expect(runningContainer.runtime_status).to.be.deep.equal({
                        'error': 'Something went wrong',
                        'errorDetail': 'Process exited with status 1',
                        'warning': 'Free disk space is low',
                    });
                });
            })
        });
        // Test that the UI shows the error and warning messages
        cy.getAll('@containerRequest', '@runningContainer').then(function([containerRequest]) {
            cy.loginAs(activeUser);
            cy.goToPath(`/processes/${containerRequest.uuid}`);
            cy.get('[data-cy=process-runtime-status-error]')
                .should('contain', 'Something went wrong')
                .and('contain', 'Process exited with status 1');
            cy.get('[data-cy=process-runtime-status-warning]')
                .should('contain', 'Free disk space is low')
                .and('contain', 'No additional warning details available');
        });


        // Force container_count for testing
        let containerCount = 2;
        cy.intercept({method: 'GET', url: '**/arvados/v1/container_requests/*'}, (req) => {
            req.reply((res) => {
                res.body.container_count = containerCount;
            });
        });

        cy.getAll('@containerRequest').then(function([containerRequest]) {
            cy.goToPath(`/processes/${containerRequest.uuid}`);
            cy.get('[data-cy=process-runtime-status-retry-warning]')
                .should('contain', 'Process retried 1 time');
        });

        cy.getAll('@containerRequest').then(function([containerRequest]) {
            containerCount = 3;
            cy.goToPath(`/processes/${containerRequest.uuid}`);
            cy.get('[data-cy=process-runtime-status-retry-warning]')
                .should('contain', 'Process retried 2 times');
        });
    });


    it('displays IO parameters with keep links and previews', function() {
        const testInputs = [
            {
                definition: {
                    "id": "#main/input_file",
                    "label": "Label Description",
                    "type": "File"
                },
                input: {
                    "input_file": {
                        "basename": "input1.tar",
                        "class": "File",
                        "location": "keep:00000000000000000000000000000000+01/input1.tar",
                        "secondaryFiles": [
                            {
                                "basename": "input1-2.txt",
                                "class": "File",
                                "location": "keep:00000000000000000000000000000000+01/input1-2.txt"
                            },
                            {
                                "basename": "input1-3.txt",
                                "class": "File",
                                "location": "keep:00000000000000000000000000000000+01/input1-3.txt"
                            },
                            {
                                "basename": "input1-4.txt",
                                "class": "File",
                                "location": "keep:00000000000000000000000000000000+01/input1-4.txt"
                            }
                        ]
                    }
                }
            },
            {
                definition: {
                    "id": "#main/input_dir",
                    "doc": "Doc Description",
                    "type": "Directory"
                },
                input: {
                    "input_dir": {
                        "basename": "11111111111111111111111111111111+01",
                        "class": "Directory",
                        "location": "keep:11111111111111111111111111111111+01"
                    }
                }
            },
            {
                definition: {
                    "id": "#main/input_bool",
                    "doc": ["Doc desc 1", "Doc desc 2"],
                    "type": "boolean"
                },
                input: {
                    "input_bool": true,
                }
            },
            {
                definition: {
                    "id": "#main/input_int",
                    "type": "int"
                },
                input: {
                    "input_int": 1,
                }
            },
            {
                definition: {
                    "id": "#main/input_long",
                    "type": "long"
                },
                input: {
                    "input_long" : 1,
                }
            },
            {
                definition: {
                    "id": "#main/input_float",
                    "type": "float"
                },
                input: {
                    "input_float": 1.5,
                }
            },
            {
                definition: {
                    "id": "#main/input_double",
                    "type": "double"
                },
                input: {
                    "input_double": 1.3,
                }
            },
            {
                definition: {
                    "id": "#main/input_string",
                    "type": "string"
                },
                input: {
                    "input_string": "Hello World",
                }
            },
            {
                definition: {
                    "id": "#main/input_file_array",
                    "type": {
                      "items": "File",
                      "type": "array"
                    }
                },
                input: {
                    "input_file_array": [
                        {
                            "basename": "input2.tar",
                            "class": "File",
                            "location": "keep:00000000000000000000000000000000+02/input2.tar"
                        },
                        {
                            "basename": "input3.tar",
                            "class": "File",
                            "location": "keep:00000000000000000000000000000000+03/input3.tar",
                            "secondaryFiles": [
                                {
                                    "basename": "input3-2.txt",
                                    "class": "File",
                                    "location": "keep:00000000000000000000000000000000+03/input3-2.txt"
                                }
                            ]
                        }
                    ]
                }
            },
            {
                definition: {
                    "id": "#main/input_dir_array",
                    "type": {
                      "items": "Directory",
                      "type": "array"
                    }
                },
                input: {
                    "input_dir_array": [
                        {
                            "basename": "11111111111111111111111111111111+02",
                            "class": "Directory",
                            "location": "keep:11111111111111111111111111111111+02"
                        },
                        {
                            "basename": "11111111111111111111111111111111+03",
                            "class": "Directory",
                            "location": "keep:11111111111111111111111111111111+03"
                        }
                    ]
                }
            },
            {
                definition: {
                    "id": "#main/input_int_array",
                    "type": {
                      "items": "int",
                      "type": "array"
                    }
                },
                input: {
                    "input_int_array": [
                        1,
                        3,
                        5
                    ]
                }
            },
            {
                definition: {
                    "id": "#main/input_long_array",
                    "type": {
                      "items": "long",
                      "type": "array"
                    }
                },
                input: {
                    "input_long_array": [
                        10,
                        20
                    ]
                }
            },
            {
                definition: {
                    "id": "#main/input_float_array",
                    "type": {
                      "items": "float",
                      "type": "array"
                    }
                },
                input: {
                    "input_float_array": [
                        10.2,
                        10.4,
                        10.6
                    ]
                }
            },
            {
                definition: {
                    "id": "#main/input_double_array",
                    "type": {
                      "items": "double",
                      "type": "array"
                    }
                },
                input: {
                    "input_double_array": [
                        20.1,
                        20.2,
                        20.3
                    ]
                }
            },
            {
                definition: {
                    "id": "#main/input_string_array",
                    "type": {
                      "items": "string",
                      "type": "array"
                    }
                },
                input: {
                    "input_string_array": [
                        "Hello",
                        "World",
                        "!"
                    ]
                }
            }
        ];

        const testOutputs = [
            {
                definition: {
                    "id": "#main/output_file",
                    "label": "Label Description",
                    "type": "File"
                },
                output: {
                    "output_file": {
                        "basename": "cat.png",
                        "class": "File",
                        "location": "cat.png"
                    }
                }
            },
            {
                definition: {
                    "id": "#main/output_file_with_secondary",
                    "doc": "Doc Description",
                    "type": "File"
                },
                output: {
                    "output_file_with_secondary": {
                        "basename": "main.dat",
                        "class": "File",
                        "location": "main.dat",
                        "secondaryFiles": [
                            {
                                "basename": "secondary.dat",
                                "class": "File",
                                "location": "secondary.dat"
                            },
                            {
                                "basename": "secondary2.dat",
                                "class": "File",
                                "location": "secondary2.dat"
                            }
                        ]
                    }
                }
            },
            {
                definition: {
                    "id": "#main/output_dir",
                    "doc": ["Doc desc 1", "Doc desc 2"],
                    "type": "Directory"
                },
                output: {
                    "output_dir": {
                        "basename": "outdir1",
                        "class": "Directory",
                        "location": "outdir1"
                    }
                }
            },
            {
                definition: {
                    "id": "#main/output_bool",
                    "type": "boolean"
                },
                output: {
                    "output_bool": true
                }
            },
            {
                definition: {
                    "id": "#main/output_int",
                    "type": "int"
                },
                output: {
                    "output_int": 1
                }
            },
            {
                definition: {
                    "id": "#main/output_long",
                    "type": "long"
                },
                output: {
                    "output_long": 1
                }
            },
            {
                definition: {
                    "id": "#main/output_float",
                    "type": "float"
                },
                output: {
                    "output_float": 100.5
                }
            },
            {
                definition: {
                    "id": "#main/output_double",
                    "type": "double"
                },
                output: {
                    "output_double": 100.3
                }
            },
            {
                definition: {
                    "id": "#main/output_string",
                    "type": "string"
                },
                output: {
                    "output_string": "Hello output"
                }
            },
            {
                definition: {
                    "id": "#main/output_file_array",
                    "type": {
                        "items": "File",
                        "type": "array"
                    }
                },
                output: {
                    "output_file_array": [
                        {
                            "basename": "output2.tar",
                            "class": "File",
                            "location": "output2.tar"
                        },
                        {
                            "basename": "output3.tar",
                            "class": "File",
                            "location": "output3.tar"
                        }
                    ]
                }
            },
            {
                definition: {
                    "id": "#main/output_dir_array",
                    "type": {
                        "items": "Directory",
                        "type": "array"
                    }
                },
                output: {
                    "output_dir_array": [
                        {
                            "basename": "outdir2",
                            "class": "Directory",
                            "location": "outdir2"
                        },
                        {
                            "basename": "outdir3",
                            "class": "Directory",
                            "location": "outdir3"
                        }
                    ]
                }
            },
            {
                definition: {
                    "id": "#main/output_int_array",
                    "type": {
                        "items": "int",
                        "type": "array"
                    }
                },
                output: {
                    "output_int_array": [
                        10,
                        11,
                        12
                    ]
                }
            },
            {
                definition: {
                    "id": "#main/output_long_array",
                    "type": {
                        "items": "long",
                        "type": "array"
                    }
                },
                output: {
                    "output_long_array": [
                        51,
                        52
                    ]
                }
            },
            {
                definition: {
                    "id": "#main/output_float_array",
                    "type": {
                        "items": "float",
                        "type": "array"
                    }
                },
                output: {
                    "output_float_array": [
                        100.2,
                        100.4,
                        100.6
                    ]
                }
            },
            {
                definition: {
                    "id": "#main/output_double_array",
                    "type": {
                        "items": "double",
                        "type": "array"
                    }
                },
                output: {
                    "output_double_array": [
                        100.1,
                        100.2,
                        100.3
                    ]
                }
            },
            {
                definition: {
                    "id": "#main/output_string_array",
                    "type": {
                        "items": "string",
                        "type": "array"
                    }
                },
                output: {
                    "output_string_array": [
                        "Hello",
                        "Output",
                        "!"
                    ]
                }
            }
        ];

        const verifyParameter = (name, doc, val) => {
            cy.get('table tr').contains(name).parents('tr').within(() => {
                doc && cy.contains(doc);
                val && cy.contains(val);
            });
        };

        const verifyImage = (name, url) => {
            cy.get('table tr').contains(name).parents('tr').within(() => {
                cy.get('[alt="Inline Preview"]')
                    .should('be.visible')
                    .and(($img) => {
                        expect($img[0].naturalWidth).to.be.greaterThan(0);
                        expect($img[0].src).contains(url);
                    })
            });
        }

        // Create output collection for real files
        cy.createCollection(adminUser.token, {
            name: `Test collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: activeUser.user.uuid,
        }).then((testOutputCollection) => {
                    cy.loginAs(activeUser);

                    cy.goToPath(`/collections/${testOutputCollection.uuid}`);

                    cy.get('[data-cy=upload-button]').click();

                    cy.fixture('files/cat.png', 'base64').then(content => {
                        cy.get('[data-cy=drag-and-drop]').upload(content, 'cat.png');
                        cy.get('[data-cy=form-submit-btn]').click();
                        cy.waitForDom().get('[data-cy=form-submit-btn]').should('not.exist');
                        // Confirm final collection state.
                        cy.get('[data-cy=collection-files-panel]')
                            .contains('cat.png').should('exist');
                    });

                    cy.getCollection(activeUser.token, testOutputCollection.uuid).as('testOutputCollection');
                });

        // Get updated collection pdh
        cy.getAll('@testOutputCollection').then(([testOutputCollection]) => {
            // Add output uuid and inputs to container request
            cy.intercept({method: 'GET', url: '**/arvados/v1/container_requests/*'}, (req) => {
                req.reply((res) => {
                    res.body.output_uuid = testOutputCollection.uuid;
                    res.body.mounts["/var/lib/cwl/cwl.input.json"] = {
                        content: testInputs.map((param) => (param.input)).reduce((acc, val) => (Object.assign(acc, val)), {})
                    };
                    res.body.mounts["/var/lib/cwl/workflow.json"] = {
                        content: {
                            $graph: [{
                                id: "#main",
                                inputs: testInputs.map((input) => (input.definition)),
                                outputs: testOutputs.map((output) => (output.definition))
                            }]
                        }
                    };
                });
            });

            // Stub fake output collection
            cy.intercept({method: 'GET', url: `**/arvados/v1/collections/${testOutputCollection.uuid}*`}, {
                statusCode: 200,
                body: {
                    uuid: testOutputCollection.uuid,
                    portable_data_hash: testOutputCollection.portable_data_hash,
                }
            });

            // Stub fake output json
            cy.intercept({method: 'GET', url: '**/c%3Dzzzzz-4zz18-zzzzzzzzzzzzzzz/cwl.output.json'}, {
                statusCode: 200,
                body: testOutputs.map((param) => (param.output)).reduce((acc, val) => (Object.assign(acc, val)), {})
            });

            // Stub webdav response, points to output json
            cy.intercept({method: 'PROPFIND', url: '*'}, {
                fixture: 'webdav-propfind-outputs.xml',
            });
        });

        createContainerRequest(
            activeUser,
            'test_container_request',
            'arvados/jobs',
            ['echo', 'hello world'],
            false, 'Committed')
        .as('containerRequest');

        cy.getAll('@containerRequest', '@testOutputCollection').then(function([containerRequest, testOutputCollection]) {
            cy.goToPath(`/processes/${containerRequest.uuid}`);
            cy.get('[data-cy=process-io-card] h6').contains('Inputs')
                .parents('[data-cy=process-io-card]').within(() => {
                    cy.wait(2000);
                    cy.waitForDom();
                    verifyParameter('input_file', "Label Description", 'keep:00000000000000000000000000000000+01/input1.tar');
                    verifyParameter('input_file', "Label Description", 'keep:00000000000000000000000000000000+01/input1-2.txt');
                    verifyParameter('input_file', "Label Description", 'keep:00000000000000000000000000000000+01/input1-3.txt');
                    verifyParameter('input_file', "Label Description", 'keep:00000000000000000000000000000000+01/input1-4.txt');
                    verifyParameter('input_dir', "Doc Description", 'keep:11111111111111111111111111111111+01/');
                    verifyParameter('input_bool', "Doc desc 1, Doc desc 2", 'true');
                    verifyParameter('input_int', null, '1');
                    verifyParameter('input_long', null, '1');
                    verifyParameter('input_float', null, '1.5');
                    verifyParameter('input_double', null, '1.3');
                    verifyParameter('input_string', null, 'Hello World');
                    verifyParameter('input_file_array', null, 'keep:00000000000000000000000000000000+02/input2.tar');
                    verifyParameter('input_file_array', null, 'keep:00000000000000000000000000000000+03/input3.tar');
                    verifyParameter('input_file_array', null, 'keep:00000000000000000000000000000000+03/input3-2.txt');
                    verifyParameter('input_dir_array', null, 'keep:11111111111111111111111111111111+02/');
                    verifyParameter('input_dir_array', null, 'keep:11111111111111111111111111111111+03/');
                    verifyParameter('input_int_array', null, '1, 3, 5');
                    verifyParameter('input_long_array', null, '10, 20');
                    verifyParameter('input_float_array', null, '10.2, 10.4, 10.6');
                    verifyParameter('input_double_array', null, '20.1, 20.2, 20.3');
                    verifyParameter('input_string_array', null, 'Hello, World, !');
                });
            cy.get('[data-cy=process-io-card] h6').contains('Outputs')
                .parents('[data-cy=process-io-card]').within((ctx) => {
                    cy.get(ctx).scrollIntoView();
                    const outPdh = testOutputCollection.portable_data_hash;

                    verifyParameter('output_file', "Label Description", `keep:${outPdh}/cat.png`);
                    verifyImage('output_file', `/c=${outPdh}/cat.png`);
                    verifyParameter('output_file_with_secondary', "Doc Description", `keep:${outPdh}/main.dat`);
                    verifyParameter('output_file_with_secondary', "Doc Description", `keep:${outPdh}/secondary.dat`);
                    verifyParameter('output_file_with_secondary', "Doc Description", `keep:${outPdh}/secondary2.dat`);
                    verifyParameter('output_dir', "Doc desc 1, Doc desc 2", `keep:${outPdh}/outdir1`);
                    verifyParameter('output_bool', null, 'true');
                    verifyParameter('output_int', null, '1');
                    verifyParameter('output_long', null, '1');
                    verifyParameter('output_float', null, '100.5');
                    verifyParameter('output_double', null, '100.3');
                    verifyParameter('output_string', null, 'Hello output');
                    verifyParameter('output_file_array', null, `keep:${outPdh}/output2.tar`);
                    verifyParameter('output_file_array', null, `keep:${outPdh}/output3.tar`);
                    verifyParameter('output_dir_array', null, `keep:${outPdh}/outdir2`);
                    verifyParameter('output_dir_array', null, `keep:${outPdh}/outdir3`);
                    verifyParameter('output_int_array', null, '10, 11, 12');
                    verifyParameter('output_long_array', null, '51, 52');
                    verifyParameter('output_float_array', null, '100.2, 100.4, 100.6');
                    verifyParameter('output_double_array', null, '100.1, 100.2, 100.3');
                    verifyParameter('output_string_array', null, 'Hello, Output, !');
                });
        });
    });

});
