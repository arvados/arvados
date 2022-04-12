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
                // Should should all logs
                cy.get('[data-cy=process-logs-filter]').should('contain', 'All logs');
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
});
