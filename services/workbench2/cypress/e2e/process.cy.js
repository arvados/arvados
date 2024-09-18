// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContainerState } from "models/container";

describe("Process tests", function () {
    let activeUser;
    let adminUser;

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
        cy.getUser("activeuser", "Active", "User", false, true)
            .as("activeUser")
            .then(function () {
                activeUser = this.activeUser;
            });
    });


    function createContainerRequest(user, name, docker_image, command, reuse = false, state = "Uncommitted") {
        return cy.setupDockerImage(docker_image).then(function (dockerImage) {
            return cy.createContainerRequest(user.token, {
                name: name,
                command: command,
                container_image: dockerImage.portable_data_hash, // for some reason, docker_image doesn't work here
                output_path: "stdout.txt",
                priority: 1,
                runtime_constraints: {
                    vcpus: 1,
                    ram: 1,
                },
                use_existing: reuse,
                state: state,
                mounts: {
                    foo: {
                        kind: "tmp",
                        path: "/tmp/foo",
                    },
                },
            });
        });
    }

    describe('Multiselect Toolbar', () => {
        it('shows the appropriate buttons in the toolbar', () => {

            const msButtonTooltips = [
                'View details',
                'Open in new tab',
                'Outputs',
                'API Details',
                'Edit process',
                'Copy and re-run process',
                'CANCEL',
                'Remove',
                'Add to favorites',
            ];

            createContainerRequest(
                activeUser,
                `test_container_request ${Math.floor(Math.random() * 999999)}`,
                "arvados/jobs",
                ["echo", "hello world"],
                false,
                "Committed"
            ).then(function (containerRequest) {
                cy.loginAs(activeUser);
                cy.goToPath(`/processes/${containerRequest.uuid}`);
                cy.get("[data-cy=process-details]").should("contain", containerRequest.name);
                cy.get("[data-cy=process-details-attributes-modifiedby-user]").contains(`Active User (${activeUser.user.uuid})`);
                cy.get("[data-cy=process-details-attributes-runtime-user]").should("not.exist");
                cy.get("[data-cy=side-panel-tree]").contains("Home Projects").click();
                cy.waitForDom();
                cy.get('[data-cy=mpv-tabs]').contains("Workflow Runs").click();
                cy.get('[data-cy=data-table-row]').contains(containerRequest.name).should('exist').parents('[data-cy=data-table-row]').click()
                cy.waitForDom();
                cy.get('[data-cy=multiselect-button]').should('have.length', msButtonTooltips.length)
                for (let i = 0; i < msButtonTooltips.length; i++) {
                    cy.get('[data-cy=multiselect-button]').eq(i).trigger('mouseover');
                    cy.get('body').contains(msButtonTooltips[i]).should('exist')
                    cy.get('[data-cy=multiselect-button]').eq(i).trigger('mouseout');
                }
            });
        })
    })

    describe("Details panel", function () {
        it("shows process details", function () {
            createContainerRequest(
                activeUser,
                `test_container_request ${Math.floor(Math.random() * 999999)}`,
                "arvados/jobs",
                ["echo", "hello world"],
                false,
                "Committed"
            ).then(function (containerRequest) {
                cy.loginAs(activeUser);
                cy.goToPath(`/processes/${containerRequest.uuid}`);
                cy.get("[data-cy=process-details]").should("contain", containerRequest.name);
                cy.get("[data-cy=process-details-attributes-modifiedby-user]").contains(`Active User (${activeUser.user.uuid})`);
                cy.get("[data-cy=process-details-attributes-runtime-user]").should("not.exist");
            });

            // Fake submitted by another user to test "runtime user" field.
            //
            // Need to override both group contents and direct get,
            // because it displays the the cached value from
            // 'contents' for a few moments while requesting the full
            // object.
            cy.intercept({ method: "GET", url: "**/arvados/v1/groups/*/contents?*" }, req => {
                req.on('response', res => {
                    if (!res.body.items) {
                        return;
                    }
                    res.body.items.forEach(item => {
                        item.modified_by_user_uuid = "zzzzz-tpzed-000000000000000";
                    });
                });
            });
            cy.intercept({ method: "GET", url: "**/arvados/v1/container_requests/*" }, req => {
                req.on('response', res => {
                    res.body.modified_by_user_uuid = "zzzzz-tpzed-000000000000000";
                });
            });

            createContainerRequest(
                activeUser,
                `test_container_request ${Math.floor(Math.random() * 999999)}`,
                "arvados/jobs",
                ["echo", "hello world"],
                false,
                "Committed"
            ).then(function (containerRequest) {
                cy.loginAs(activeUser);
                cy.goToPath(`/processes/${containerRequest.uuid}`);
                cy.get("[data-cy=process-details]").should("contain", containerRequest.name);
                cy.get("[data-cy=process-details-attributes-modifiedby-user]").contains(`zzzzz-tpzed-000000000000000`);
                cy.get("[data-cy=process-details-attributes-runtime-user]").contains(`Active User (${activeUser.user.uuid})`);
            });
        });

        it("should show runtime status indicators", function () {
            // Setup running container with runtime_status error & warning messages
            createContainerRequest(activeUser, "test_container_request", "arvados/jobs", ["echo", "hello world"], false, "Committed")
                .as("containerRequest")
                .then(function (containerRequest) {
                    expect(containerRequest.state).to.equal("Committed");
                    expect(containerRequest.container_uuid).not.to.be.equal("");

                    cy.getContainer(activeUser.token, containerRequest.container_uuid).then(function (queuedContainer) {
                        expect(queuedContainer.state).to.be.equal("Queued");
                    });
                    cy.updateContainer(adminUser.token, containerRequest.container_uuid, {
                        state: "Locked",
                    }).then(function (lockedContainer) {
                        expect(lockedContainer.state).to.be.equal("Locked");

                        cy.updateContainer(adminUser.token, lockedContainer.uuid, {
                            state: "Running",
                            runtime_status: {
                                error: "Something went wrong",
                                errorDetail: "Process exited with status 1",
                                warning: "Free disk space is low",
                            },
                        })
                            .as("runningContainer")
                            .then(function (runningContainer) {
                                expect(runningContainer.state).to.be.equal("Running");
                                expect(runningContainer.runtime_status).to.be.deep.equal({
                                    error: "Something went wrong",
                                    errorDetail: "Process exited with status 1",
                                    warning: "Free disk space is low",
                                });
                            });
                    });
                });
            // Test that the UI shows the error and warning messages
            cy.getAll("@containerRequest", "@runningContainer").then(function ([containerRequest]) {
                cy.loginAs(activeUser);
                cy.goToPath(`/processes/${containerRequest.uuid}`);
                cy.get("[data-cy=process-runtime-status-error]")
                    .should("contain", "Something went wrong")
                    .and("contain", "Process exited with status 1");
                cy.get("[data-cy=process-runtime-status-warning]")
                    .should("contain", "Free disk space is low")
                    .and("contain", "No additional warning details available");
            });

            // Force container_count for testing
            let containerCount = 2;
            cy.intercept({ method: "GET", url: "**/arvados/v1/container_requests/*" }, req => {
                req.on('response', res => {
                    res.body.container_count = containerCount;
                });
            });

            cy.getAll("@containerRequest", "@runningContainer").then(function ([containerRequest]) {
                cy.goToPath(`/processes/${containerRequest.uuid}`);
                cy.reload();
                cy.get("[data-cy=process-runtime-status-retry-warning]", { timeout: 7000 }).should("contain", "Process retried 1 time")
            }).as("retry1");

            cy.getAll("@containerRequest", "@runningContainer", "@retry1").then(function ([containerRequest]) {
                containerCount = 3;
                cy.goToPath(`/processes/${containerRequest.uuid}`);
                cy.reload();
                cy.get("[data-cy=process-runtime-status-retry-warning]", { timeout: 7000 }).should("contain", "Process retried 2 times");
            });
        });

        it("allows copying processes", function () {
            const crName = "first_container_request";
            const copiedCrName = "copied_container_request";
            createContainerRequest(activeUser, crName, "arvados/jobs", ["echo", "hello world"], false, "Committed").then(function (containerRequest) {
                cy.loginAs(activeUser);
                cy.goToPath(`/processes/${containerRequest.uuid}`);
                cy.get("[data-cy=process-details]").should("contain", crName);
                cy.get("[data-cy=process-details]").find('button[aria-label="More options"]').click();
                cy.get("ul[data-cy=context-menu]").contains("Copy and re-run process").click();
            });

            cy.get("[data-cy=form-dialog]").within(() => {
                cy.get("input[name=name]").clear().type(copiedCrName);
                cy.get("[data-cy=projects-tree-home-tree-picker]").click();
                cy.get("[data-cy=form-submit-btn]").click();
            });

            cy.get("[data-cy=process-details]").should("contain", copiedCrName);
            cy.get("[data-cy=process-details]").find("button").contains("Run");
        });

        const getFakeContainer = fakeContainerUuid => ({
            href: `/containers/${fakeContainerUuid}`,
            kind: "arvados#container",
            etag: "ecfosljpnxfari9a8m7e4yv06",
            uuid: fakeContainerUuid,
            owner_uuid: "zzzzz-tpzed-000000000000000",
            created_at: "2023-02-13T15:55:47.308915000Z",
            modified_by_user_uuid: "zzzzz-tpzed-000000000000000",
            modified_at: "2023-02-15T19:12:45.987086000Z",
            command: [
                "arvados-cwl-runner",
                "--api=containers",
                "--local",
                "--project-uuid=zzzzz-j7d0g-yr18k784zplfeza",
                "/var/lib/cwl/workflow.json#main",
                "/var/lib/cwl/cwl.input.json",
            ],
            container_image: "4ad7d11381df349e464694762db14e04+303",
            cwd: "/var/spool/cwl",
            environment: {},
            exit_code: null,
            finished_at: null,
            locked_by_uuid: null,
            log: null,
            output: null,
            output_path: "/var/spool/cwl",
            progress: null,
            runtime_constraints: {
                API: true,
                cuda: {
                    device_count: 0,
                    driver_version: "",
                    hardware_capability: "",
                },
                keep_cache_disk: 2147483648,
                keep_cache_ram: 0,
                ram: 1342177280,
                vcpus: 1,
            },
            runtime_status: {},
            started_at: null,
            auth_uuid: null,
            scheduling_parameters: {
                max_run_time: 0,
                partitions: [],
                preemptible: false,
            },
            runtime_user_uuid: "zzzzz-tpzed-vllbpebicy84rd5",
            runtime_auth_scopes: ["all"],
            lock_count: 2,
            gateway_address: null,
            interactive_session_started: false,
            output_storage_classes: ["default"],
            output_properties: {},
            cost: 0.0,
            subrequests_cost: 0.0,
        });

        it("shows cancel button when appropriate", function () {
            // Ignore collection requests
            cy.intercept(
                { method: "GET", url: `**/arvados/v1/collections/*` },
                {
                    statusCode: 200,
                    body: {},
                }
            );

            // Uncommitted container
            const crUncommitted = `Test process ${Math.floor(Math.random() * 999999)}`;
            createContainerRequest(activeUser, crUncommitted, "arvados/jobs", ["echo", "hello world"], false, "Uncommitted").then(function (
                containerRequest
            ) {
                cy.loginAs(activeUser);
                // Navigate to process and verify run / cancel button
                cy.goToPath(`/processes/${containerRequest.uuid}`);
                cy.waitForDom();
                cy.get("[data-cy=process-details]").should("contain", crUncommitted);
                cy.get("[data-cy=process-run-button]").should("exist");
                cy.get("[data-cy=process-cancel-button]").should("not.exist");
            });

            // Queued container
            const crQueued = `Test process ${Math.floor(Math.random() * 999999)}`;
            const fakeCrUuid = "zzzzz-dz642-000000000000001";
            createContainerRequest(activeUser, crQueued, "arvados/jobs", ["echo", "hello world"], false, "Committed").then(function (
                containerRequest
            ) {
                // Fake container uuid
                cy.intercept({ method: "GET", url: `**/arvados/v1/container_requests/${containerRequest.uuid}` }, req => {
                    req.on('response', res => {
                        res.body.output_uuid = fakeCrUuid;
                        res.body.priority = 500;
                        res.body.state = "Committed";
                    });
                });

                // Fake container
                const container = getFakeContainer(fakeCrUuid);
                cy.intercept(
                    { method: "GET", url: `**/arvados/v1/container/${fakeCrUuid}` },
                    {
                        statusCode: 200,
                        body: { ...container, state: "Queued", priority: 500 },
                    }
                );

                // Navigate to process and verify cancel button
                cy.goToPath(`/processes/${containerRequest.uuid}`);
                cy.waitForDom();
                cy.get("[data-cy=process-details]").should("contain", crQueued);
                cy.get("[data-cy=process-cancel-button]").contains("Cancel");
            });

            // Locked container
            const crLocked = `Test process ${Math.floor(Math.random() * 999999)}`;
            const fakeCrLockedUuid = "zzzzz-dz642-000000000000002";
            createContainerRequest(activeUser, crLocked, "arvados/jobs", ["echo", "hello world"], false, "Committed").then(function (
                containerRequest
            ) {
                // Fake container uuid
                cy.intercept({ method: "GET", url: `**/arvados/v1/container_requests/${containerRequest.uuid}` }, req => {
                    req.on('response', res => {
                        res.body.output_uuid = fakeCrLockedUuid;
                        res.body.priority = 500;
                        res.body.state = "Committed";
                    });
                });

                // Fake container
                const container = getFakeContainer(fakeCrLockedUuid);
                cy.intercept(
                    { method: "GET", url: `**/arvados/v1/container/${fakeCrLockedUuid}` },
                    {
                        statusCode: 200,
                        body: { ...container, state: "Locked", priority: 500 },
                    }
                );

                // Navigate to process and verify cancel button
                cy.goToPath(`/processes/${containerRequest.uuid}`);
                cy.waitForDom();
                cy.get("[data-cy=process-details]").should("contain", crLocked);
                cy.get("[data-cy=process-cancel-button]").contains("Cancel");
            });

            // On Hold container
            const crOnHold = `Test process ${Math.floor(Math.random() * 999999)}`;
            const fakeCrOnHoldUuid = "zzzzz-dz642-000000000000003";
            createContainerRequest(activeUser, crOnHold, "arvados/jobs", ["echo", "hello world"], false, "Committed").then(function (
                containerRequest
            ) {
                // Fake container uuid
                cy.intercept({ method: "GET", url: `**/arvados/v1/container_requests/${containerRequest.uuid}` }, req => {
                    req.on('response', res => {
                        res.body.output_uuid = fakeCrOnHoldUuid;
                        res.body.priority = 0;
                        res.body.state = "Committed";
                    });
                });

                // Fake container
                const container = getFakeContainer(fakeCrOnHoldUuid);
                cy.intercept(
                    { method: "GET", url: `**/arvados/v1/container/${fakeCrOnHoldUuid}` },
                    {
                        statusCode: 200,
                        body: { ...container, state: "Queued", priority: 0 },
                    }
                );

                // Navigate to process and verify cancel button
                cy.goToPath(`/processes/${containerRequest.uuid}`);
                cy.waitForDom();
                cy.get("[data-cy=process-details]").should("contain", crOnHold);
                cy.get("[data-cy=process-run-button]").should("exist");
                cy.get("[data-cy=process-cancel-button]").should("not.exist");
            });
        });
    });

    describe("Logs panel", function () {
        it("shows live process logs", function () {
            cy.intercept({ method: "GET", url: "**/arvados/v1/containers/*" }, req => {
                req.on('response', res => {
                    res.body.state = ContainerState.RUNNING;
                });
            });

            const crName = "test_container_request";
            createContainerRequest(activeUser, crName, "arvados/jobs", ["echo", "hello world"], false, "Committed").then(function (containerRequest) {
                // Create empty log file before loading process page
                cy.appendLog(adminUser.token, containerRequest.uuid, "stdout.txt", [""]);

                cy.loginAs(activeUser);
                cy.goToPath(`/processes/${containerRequest.uuid}`);
                cy.get("[data-cy=process-details]").should("contain", crName);
                cy.get("[data-cy=process-logs]").should("contain", "No logs yet").and("not.contain", "hello world");

                // Append a log line
                cy.appendLog(adminUser.token, containerRequest.uuid, "stdout.txt", ["2023-07-18T20:14:48.128642814Z hello world"]).then(() => {
                    cy.get("[data-cy=process-logs]", { timeout: 7000 }).should("not.contain", "No logs yet").and("contain", "hello world");
                });

                // Append new log line to different file
                cy.appendLog(adminUser.token, containerRequest.uuid, "stderr.txt", ["2023-07-18T20:14:49.128642814Z hello new line"]).then(() => {
                    cy.get("[data-cy=process-logs]", { timeout: 7000 }).should("not.contain", "No logs yet").and("contain", "hello new line");
                });
            });
        });

        it("filters process logs by event type", function () {
            const nodeInfoLogs = [
                "Host Information",
                "Linux compute-99cb150b26149780de44b929577e1aed-19rgca8vobuvc4p 5.4.0-1059-azure #62~18.04.1-Ubuntu SMP Tue Sep 14 17:53:18 UTC 2021 x86_64 x86_64 x86_64 GNU/Linux",
                "CPU Information",
                "processor  : 0",
                "vendor_id  : GenuineIntel",
                "cpu family : 6",
                "model      : 79",
                "model name : Intel(R) Xeon(R) CPU E5-2673 v4 @ 2.30GHz",
            ];
            const crunchRunLogs = [
                "2022-03-22T13:56:22.542417997Z using local keepstore process (pid 3733) at http://localhost:46837, writing logs to keepstore.txt in log collection",
                "2022-03-22T13:56:26.237571754Z crunch-run 2.4.0~dev20220321141729 (go1.17.1) started",
                "2022-03-22T13:56:26.244704134Z crunch-run process has uid=0(root) gid=0(root) groups=0(root)",
                "2022-03-22T13:56:26.244862836Z Executing container 'zzzzz-dz642-1wokwvcct9s9du3' using docker runtime",
                "2022-03-22T13:56:26.245037738Z Executing on host 'compute-99cb150b26149780de44b929577e1aed-19rgca8vobuvc4p'",
            ];
            const stdoutLogs = [
                "2022-03-22T13:56:22.542417987Z Lorem ipsum dolor sit amet, consectetur adipiscing elit. Donec dui nisi, hendrerit porta sapien a, pretium dignissim purus.",
                "2022-03-22T13:56:22.542417997Z Integer viverra, mauris finibus aliquet ultricies, dui mauris cursus justo, ut venenatis nibh ex eget neque.",
                "2022-03-22T13:56:22.542418007Z In hac habitasse platea dictumst.",
                "2022-03-22T13:56:22.542418027Z Fusce fringilla turpis id accumsan faucibus. Donec congue congue ex non posuere. In semper mi quis tristique rhoncus.",
                "2022-03-22T13:56:22.542418037Z Interdum et malesuada fames ac ante ipsum primis in faucibus.",
                "2022-03-22T13:56:22.542418047Z Quisque fermentum tortor ex, ut suscipit velit feugiat faucibus.",
                "2022-03-22T13:56:22.542418057Z Donec vitae porta risus, at luctus nulla. Mauris gravida iaculis ipsum, id sagittis tortor egestas ac.",
                "2022-03-22T13:56:22.542418067Z Maecenas condimentum volutpat nulla. Integer lacinia maximus risus eu posuere.",
                "2022-03-22T13:56:22.542418077Z Donec vitae leo id augue gravida bibendum.",
                "2022-03-22T13:56:22.542418087Z Nam libero libero, pretium ac faucibus elementum, mattis nec ex.",
                "2022-03-22T13:56:22.542418097Z Nullam id laoreet nibh. Vivamus tellus metus, pretium quis justo ut, bibendum varius metus. Pellentesque vitae accumsan lorem, quis tincidunt augue.",
                "2022-03-22T13:56:22.542418107Z Aliquam viverra nisi nulla, et efficitur dolor mattis in.",
                "2022-03-22T13:56:22.542418117Z Sed at enim sit amet nulla tincidunt mattis. Aenean eget aliquet ex, non ultrices ex. Nulla ex tortor, vestibulum aliquam tempor ac, aliquam vel est.",
                "2022-03-22T13:56:22.542418127Z Fusce auctor faucibus libero id venenatis. Etiam sodales, odio eu cursus efficitur, quam sem blandit ex, quis porttitor enim dui quis lectus. In id tincidunt felis.",
                "2022-03-22T13:56:22.542418137Z Phasellus non ex quis arcu tempus faucibus molestie in sapien.",
                "2022-03-22T13:56:22.542418147Z Duis tristique semper dolor, vitae pulvinar risus.",
                "2022-03-22T13:56:22.542418157Z Aliquam tortor elit, luctus nec tortor eget, porta tristique nulla.",
                "2022-03-22T13:56:22.542418167Z Nulla eget mollis ipsum.",
            ];

            createContainerRequest(activeUser, "test_container_request", "arvados/jobs", ["echo", "hello world"], false, "Committed").then(function (
                containerRequest
            ) {
                cy.appendLog(adminUser.token, containerRequest.uuid, "node-info.txt", nodeInfoLogs).as("nodeInfoLogs");
                cy.appendLog(adminUser.token, containerRequest.uuid, "crunch-run.txt", crunchRunLogs).as("crunchRunLogs");
                cy.appendLog(adminUser.token, containerRequest.uuid, "stdout.txt", stdoutLogs).as("stdoutLogs");

                cy.getAll("@stdoutLogs", "@nodeInfoLogs", "@crunchRunLogs").then(function () {
                    cy.loginAs(activeUser);
                    cy.goToPath(`/processes/${containerRequest.uuid}`);
                    cy.waitForDom();
                    // Should show main logs by default
                    cy.get("[data-cy=process-logs-filter]", { timeout: 7000 }).should("contain", "Main logs");
                    cy.get("[data-cy=process-logs]")
                        .should("contain", stdoutLogs[Math.floor(Math.random() * stdoutLogs.length)])
                        .and("not.contain", nodeInfoLogs[Math.floor(Math.random() * nodeInfoLogs.length)])
                        .and("contain", crunchRunLogs[Math.floor(Math.random() * crunchRunLogs.length)]);
                    // Select 'All logs'
                    cy.get("[data-cy=process-logs-filter]").click();
                    cy.get("body").contains("li", "All logs").click();
                    cy.get("[data-cy=process-logs]")
                        .should("contain", stdoutLogs[Math.floor(Math.random() * stdoutLogs.length)])
                        .and("contain", nodeInfoLogs[Math.floor(Math.random() * nodeInfoLogs.length)])
                        .and("contain", crunchRunLogs[Math.floor(Math.random() * crunchRunLogs.length)]);
                    // Select 'node-info' logs
                    cy.get("[data-cy=process-logs-filter]").click();
                    cy.get("body").contains("li", "node-info").click();
                    cy.get("[data-cy=process-logs]")
                        .should("not.contain", stdoutLogs[Math.floor(Math.random() * stdoutLogs.length)])
                        .and("contain", nodeInfoLogs[Math.floor(Math.random() * nodeInfoLogs.length)])
                        .and("not.contain", crunchRunLogs[Math.floor(Math.random() * crunchRunLogs.length)]);
                    // Select 'stdout' logs
                    cy.get("[data-cy=process-logs-filter]").click();
                    cy.get("body").contains("li", "stdout").click();
                    cy.get("[data-cy=process-logs]")
                        .should("contain", stdoutLogs[Math.floor(Math.random() * stdoutLogs.length)])
                        .and("not.contain", nodeInfoLogs[Math.floor(Math.random() * nodeInfoLogs.length)])
                        .and("not.contain", crunchRunLogs[Math.floor(Math.random() * crunchRunLogs.length)]);
                });
            });
        });

        it("sorts combined logs", function () {
            const crName = "test_container_request";
            createContainerRequest(activeUser, crName, "arvados/jobs", ["echo", "hello world"], false, "Committed").then(function (containerRequest) {
                cy.appendLog(adminUser.token, containerRequest.uuid, "node-info.txt", [
                    "3: nodeinfo 1",
                    "2: nodeinfo 2",
                    "1: nodeinfo 3",
                    "2: nodeinfo 4",
                    "3: nodeinfo 5",
                ]).as("node-info");

                cy.appendLog(adminUser.token, containerRequest.uuid, "stdout.txt", [
                    "2023-07-18T20:14:48.128642814Z first",
                    "2023-07-18T20:14:49.128642814Z third",
                ]).as("stdout");

                cy.appendLog(adminUser.token, containerRequest.uuid, "stderr.txt", ["2023-07-18T20:14:48.528642814Z second"]).as("stderr");

                cy.loginAs(activeUser);
                cy.goToPath(`/processes/${containerRequest.uuid}`);
                cy.get("[data-cy=process-details]").should("contain", crName);
                cy.get("[data-cy=process-logs]").should("contain", "No logs yet");

                cy.getAll("@node-info", "@stdout", "@stderr").then(() => {
                    // Verify sorted main logs
                    cy.get("[data-cy=process-logs] span > p", { timeout: 7000 }).eq(0).should("contain", "2023-07-18T20:14:48.128642814Z first");
                    cy.get("[data-cy=process-logs] span > p").eq(1).should("contain", "2023-07-18T20:14:48.528642814Z second");
                    cy.get("[data-cy=process-logs] span > p").eq(2).should("contain", "2023-07-18T20:14:49.128642814Z third");

                    // Switch to All logs
                    cy.get("[data-cy=process-logs-filter]").click();
                    cy.get("body").contains("li", "All logs").click();
                    // Verify non-sorted lines were preserved
                    cy.get("[data-cy=process-logs] span > p").eq(0).should("contain", "3: nodeinfo 1");
                    cy.get("[data-cy=process-logs] span > p").eq(1).should("contain", "2: nodeinfo 2");
                    cy.get("[data-cy=process-logs] span > p").eq(2).should("contain", "1: nodeinfo 3");
                    cy.get("[data-cy=process-logs] span > p").eq(3).should("contain", "2: nodeinfo 4");
                    cy.get("[data-cy=process-logs] span > p").eq(4).should("contain", "3: nodeinfo 5");
                    // Verify sorted logs
                    cy.get("[data-cy=process-logs] span > p").eq(5).should("contain", "2023-07-18T20:14:48.128642814Z first");
                    cy.get("[data-cy=process-logs] span > p").eq(6).should("contain", "2023-07-18T20:14:48.528642814Z second");
                    cy.get("[data-cy=process-logs] span > p").eq(7).should("contain", "2023-07-18T20:14:49.128642814Z third");
                });
            });
        });

        it("preserves original ordering of lines within the same log type", function () {
            const crName = "test_container_request";
            createContainerRequest(activeUser, crName, "arvados/jobs", ["echo", "hello world"], false, "Committed").then(function (containerRequest) {
                cy.appendLog(adminUser.token, containerRequest.uuid, "stdout.txt", [
                    // Should come first
                    "2023-07-18T20:14:46.000000000Z A out 1",
                    // Comes fourth in a contiguous block
                    "2023-07-18T20:14:48.128642814Z A out 2",
                    "2023-07-18T20:14:48.128642814Z X out 3",
                    "2023-07-18T20:14:48.128642814Z A out 4",
                ]).as("stdout");

                cy.appendLog(adminUser.token, containerRequest.uuid, "stderr.txt", [
                    // Comes second
                    "2023-07-18T20:14:47.000000000Z Z err 1",
                    // Comes third in a contiguous block
                    "2023-07-18T20:14:48.128642814Z B err 2",
                    "2023-07-18T20:14:48.128642814Z C err 3",
                    "2023-07-18T20:14:48.128642814Z Y err 4",
                    "2023-07-18T20:14:48.128642814Z Z err 5",
                    "2023-07-18T20:14:48.128642814Z A err 6",
                ]).as("stderr");

                cy.loginAs(activeUser);
                cy.goToPath(`/processes/${containerRequest.uuid}`);
                cy.get("[data-cy=process-details]").should("contain", crName);
                cy.get("[data-cy=process-logs]").should("contain", "No logs yet");

                cy.getAll("@stdout", "@stderr").then(() => {
                    // Switch to All logs
                    cy.get("[data-cy=process-logs-filter]").click();
                    cy.get("body").contains("li", "All logs").click();
                    // Verify sorted logs
                    cy.get("[data-cy=process-logs] span > p").eq(0).should("contain", "2023-07-18T20:14:46.000000000Z A out 1");
                    cy.get("[data-cy=process-logs] span > p").eq(1).should("contain", "2023-07-18T20:14:47.000000000Z Z err 1");
                    cy.get("[data-cy=process-logs] span > p").eq(2).should("contain", "2023-07-18T20:14:48.128642814Z B err 2");
                    cy.get("[data-cy=process-logs] span > p").eq(3).should("contain", "2023-07-18T20:14:48.128642814Z C err 3");
                    cy.get("[data-cy=process-logs] span > p").eq(4).should("contain", "2023-07-18T20:14:48.128642814Z Y err 4");
                    cy.get("[data-cy=process-logs] span > p").eq(5).should("contain", "2023-07-18T20:14:48.128642814Z Z err 5");
                    cy.get("[data-cy=process-logs] span > p").eq(6).should("contain", "2023-07-18T20:14:48.128642814Z A err 6");
                    cy.get("[data-cy=process-logs] span > p").eq(7).should("contain", "2023-07-18T20:14:48.128642814Z A out 2");
                    cy.get("[data-cy=process-logs] span > p").eq(8).should("contain", "2023-07-18T20:14:48.128642814Z X out 3");
                    cy.get("[data-cy=process-logs] span > p").eq(9).should("contain", "2023-07-18T20:14:48.128642814Z A out 4");
                });
            });
        });

        it("correctly generates sniplines", function () {
            const SNIPLINE = `================ ✀ ================ ✀ ========= Some log(s) were skipped ========= ✀ ================ ✀ ================`;
            const crName = "test_container_request";
            createContainerRequest(activeUser, crName, "arvados/jobs", ["echo", "hello world"], false, "Committed").then(function (containerRequest) {
                cy.appendLog(adminUser.token, containerRequest.uuid, "stdout.txt", [
                    "X".repeat(63999) + "_" + "O".repeat(100) + "_" + "X".repeat(63999),
                ]).as("stdout");

                cy.loginAs(activeUser);
                cy.goToPath(`/processes/${containerRequest.uuid}`);
                cy.get("[data-cy=process-details]").should("contain", crName);
                cy.get("[data-cy=process-logs]").should("contain", "No logs yet");

                // Switch to stdout since lines are unsortable (no timestamp)
                cy.get("[data-cy=process-logs-filter]").click();
                cy.get("body").contains("li", "stdout").click();

                cy.getAll("@stdout").then(() => {
                    // Verify first 64KB and snipline
                    cy.get("[data-cy=process-logs] span > p", { timeout: 7000 })
                        .eq(0)
                        .should("contain", "X".repeat(63999) + "_\n" + SNIPLINE);
                    // Verify last 64KB
                    cy.get("[data-cy=process-logs] span > p")
                        .eq(1)
                        .should("contain", "_" + "X".repeat(63999));
                    // Verify none of the Os got through
                    cy.get("[data-cy=process-logs] span > p").should("not.contain", "O");
                });
            });
        });

        it("correctly break long lines when no obvious line separation exists", function () {
            function randomString(length) {
                const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
                let res = '';
                for (let i = 0; i < length; i++) {
                    res += chars.charAt(Math.floor(Math.random() * chars.length));
                }
                return res;
            }

            const logLinesQty = 10;
            const logLines = [];
            for (let i = 0; i < logLinesQty; i++) {
                const length = Math.floor(Math.random() * 500) + 500;
                logLines.push(randomString(length));
            }

            createContainerRequest(activeUser, "test_container_request", "arvados/jobs", ["echo", "hello world"], false, "Committed").then(function (
                containerRequest
            ) {
                cy.appendLog(adminUser.token, containerRequest.uuid, "stdout.txt", logLines).as("stdoutLogs");

                cy.getAll("@stdoutLogs").then(function () {
                    cy.loginAs(activeUser);
                    cy.goToPath(`/processes/${containerRequest.uuid}`);
                    // Select 'stdout' log filter
                    cy.get("[data-cy=process-logs-filter]").click();
                    cy.get("body").contains("li", "stdout").click();
                    cy.get("[data-cy=process-logs] span > p")
                        .should('have.length', logLinesQty)
                        .each($p => {
                            expect($p.text().length).to.be.greaterThan(499);

                            // This looks like an ugly hack, but I was not able
                            // to get [client|scroll]Width attributes through
                            // the usual Cypress methods.
                            const parentClientWidth = $p[0].parentElement.clientWidth;
                            const parentScrollWidth = $p[0].parentElement.scrollWidth
                            // Scrollbar should not be visible
                            expect(parentClientWidth).to.be.eq(parentScrollWidth);
                        });
                });
            });
        });
    });

    describe("I/O panel", function () {
        const testInputs = [
            {
                definition: {
                    id: "#main/input_file",
                    label: "Label Description",
                    type: "File",
                },
                input: {
                    input_file: {
                        basename: "input1.tar",
                        class: "File",
                        location: "keep:00000000000000000000000000000000+01/input1.tar",
                        secondaryFiles: [
                            {
                                basename: "input1-2.txt",
                                class: "File",
                                location: "keep:00000000000000000000000000000000+01/input1-2.txt",
                            },
                            {
                                basename: "input1-3.txt",
                                class: "File",
                                location: "keep:00000000000000000000000000000000+01/input1-3.txt",
                            },
                            {
                                basename: "input1-4.txt",
                                class: "File",
                                location: "keep:00000000000000000000000000000000+01/input1-4.txt",
                            },
                        ],
                    },
                },
            },
            {
                definition: {
                    id: "#main/input_dir",
                    doc: "Doc Description",
                    type: "Directory",
                },
                input: {
                    input_dir: {
                        basename: "11111111111111111111111111111111+01",
                        class: "Directory",
                        location: "keep:11111111111111111111111111111111+01",
                    },
                },
            },
            {
                definition: {
                    id: "#main/input_bool",
                    doc: ["Doc desc 1", "Doc desc 2"],
                    type: "boolean",
                },
                input: {
                    input_bool: true,
                },
            },
            {
                definition: {
                    id: "#main/input_int",
                    type: "int",
                },
                input: {
                    input_int: 1,
                },
            },
            {
                definition: {
                    id: "#main/input_long",
                    type: "long",
                },
                input: {
                    input_long: 1,
                },
            },
            {
                definition: {
                    id: "#main/input_float",
                    type: "float",
                },
                input: {
                    input_float: 1.5,
                },
            },
            {
                definition: {
                    id: "#main/input_double",
                    type: "double",
                },
                input: {
                    input_double: 1.3,
                },
            },
            {
                definition: {
                    id: "#main/input_string",
                    type: "string",
                },
                input: {
                    input_string: "Hello World",
                },
            },
            {
                definition: {
                    id: "#main/input_file_array",
                    type: {
                        items: "File",
                        type: "array",
                    },
                },
                input: {
                    input_file_array: [
                        {
                            basename: "input2.tar",
                            class: "File",
                            location: "keep:00000000000000000000000000000000+02/input2.tar",
                        },
                        {
                            basename: "input3.tar",
                            class: "File",
                            location: "keep:00000000000000000000000000000000+03/input3.tar",
                            secondaryFiles: [
                                {
                                    basename: "input3-2.txt",
                                    class: "File",
                                    location: "keep:00000000000000000000000000000000+03/input3-2.txt",
                                },
                            ],
                        },
                        {
                            $import: "import_path",
                        },
                    ],
                },
            },
            {
                definition: {
                    id: "#main/input_dir_array",
                    type: {
                        items: "Directory",
                        type: "array",
                    },
                },
                input: {
                    input_dir_array: [
                        {
                            basename: "11111111111111111111111111111111+02",
                            class: "Directory",
                            location: "keep:11111111111111111111111111111111+02",
                        },
                        {
                            basename: "11111111111111111111111111111111+03",
                            class: "Directory",
                            location: "keep:11111111111111111111111111111111+03",
                        },
                        {
                            $import: "import_path",
                        },
                    ],
                },
            },
            {
                definition: {
                    id: "#main/input_int_array",
                    type: {
                        items: "int",
                        type: "array",
                    },
                },
                input: {
                    input_int_array: [
                        1,
                        3,
                        5,
                        {
                            $import: "import_path",
                        },
                    ],
                },
            },
            {
                definition: {
                    id: "#main/input_long_array",
                    type: {
                        items: "long",
                        type: "array",
                    },
                },
                input: {
                    input_long_array: [
                        10,
                        20,
                        {
                            $import: "import_path",
                        },
                    ],
                },
            },
            {
                definition: {
                    id: "#main/input_float_array",
                    type: {
                        items: "float",
                        type: "array",
                    },
                },
                input: {
                    input_float_array: [
                        10.2,
                        10.4,
                        10.6,
                        {
                            $import: "import_path",
                        },
                    ],
                },
            },
            {
                definition: {
                    id: "#main/input_double_array",
                    type: {
                        items: "double",
                        type: "array",
                    },
                },
                input: {
                    input_double_array: [
                        20.1,
                        20.2,
                        20.3,
                        {
                            $import: "import_path",
                        },
                    ],
                },
            },
            {
                definition: {
                    id: "#main/input_string_array",
                    type: {
                        items: "string",
                        type: "array",
                    },
                },
                input: {
                    input_string_array: [
                        "Hello",
                        "World",
                        "!",
                        {
                            $import: "import_path",
                        },
                    ],
                },
            },
            {
                definition: {
                    id: "#main/input_bool_include",
                    type: "boolean",
                },
                input: {
                    input_bool_include: {
                        $include: "include_path",
                    },
                },
            },
            {
                definition: {
                    id: "#main/input_int_include",
                    type: "int",
                },
                input: {
                    input_int_include: {
                        $include: "include_path",
                    },
                },
            },
            {
                definition: {
                    id: "#main/input_float_include",
                    type: "float",
                },
                input: {
                    input_float_include: {
                        $include: "include_path",
                    },
                },
            },
            {
                definition: {
                    id: "#main/input_string_include",
                    type: "string",
                },
                input: {
                    input_string_include: {
                        $include: "include_path",
                    },
                },
            },
            {
                definition: {
                    id: "#main/input_file_include",
                    type: "File",
                },
                input: {
                    input_file_include: {
                        $include: "include_path",
                    },
                },
            },
            {
                definition: {
                    id: "#main/input_directory_include",
                    type: "Directory",
                },
                input: {
                    input_directory_include: {
                        $include: "include_path",
                    },
                },
            },
            {
                definition: {
                    id: "#main/input_file_url",
                    type: "File",
                },
                input: {
                    input_file_url: {
                        basename: "index.html",
                        class: "File",
                        location: "http://example.com/index.html",
                    },
                },
            },
        ];

        const testOutputs = [
            {
                definition: {
                    id: "#main/output_file",
                    label: "Label Description",
                    type: "File",
                },
                output: {
                    output_file: {
                        basename: "cat.png",
                        class: "File",
                        location: "cat.png",
                    },
                },
            },
            {
                definition: {
                    id: "#main/output_file_with_secondary",
                    doc: "Doc Description",
                    type: "File",
                },
                output: {
                    output_file_with_secondary: {
                        basename: "main.dat",
                        class: "File",
                        location: "main.dat",
                        secondaryFiles: [
                            {
                                basename: "secondary.dat",
                                class: "File",
                                location: "secondary.dat",
                            },
                            {
                                basename: "secondary2.dat",
                                class: "File",
                                location: "secondary2.dat",
                            },
                        ],
                    },
                },
            },
            {
                definition: {
                    id: "#main/output_dir",
                    doc: ["Doc desc 1", "Doc desc 2"],
                    type: "Directory",
                },
                output: {
                    output_dir: {
                        basename: "outdir1",
                        class: "Directory",
                        location: "outdir1",
                    },
                },
            },
            {
                definition: {
                    id: "#main/output_bool",
                    type: "boolean",
                },
                output: {
                    output_bool: true,
                },
            },
            {
                definition: {
                    id: "#main/output_int",
                    type: "int",
                },
                output: {
                    output_int: 1,
                },
            },
            {
                definition: {
                    id: "#main/output_long",
                    type: "long",
                },
                output: {
                    output_long: 1,
                },
            },
            {
                definition: {
                    id: "#main/output_float",
                    type: "float",
                },
                output: {
                    output_float: 100.5,
                },
            },
            {
                definition: {
                    id: "#main/output_double",
                    type: "double",
                },
                output: {
                    output_double: 100.3,
                },
            },
            {
                definition: {
                    id: "#main/output_string",
                    type: "string",
                },
                output: {
                    output_string: "Hello output",
                },
            },
            {
                definition: {
                    id: "#main/output_file_array",
                    type: {
                        items: "File",
                        type: "array",
                    },
                },
                output: {
                    output_file_array: [
                        {
                            basename: "output2.tar",
                            class: "File",
                            location: "output2.tar",
                        },
                        {
                            basename: "output3.tar",
                            class: "File",
                            location: "output3.tar",
                        },
                    ],
                },
            },
            {
                definition: {
                    id: "#main/output_dir_array",
                    type: {
                        items: "Directory",
                        type: "array",
                    },
                },
                output: {
                    output_dir_array: [
                        {
                            basename: "outdir2",
                            class: "Directory",
                            location: "outdir2",
                        },
                        {
                            basename: "outdir3",
                            class: "Directory",
                            location: "outdir3",
                        },
                    ],
                },
            },
            {
                definition: {
                    id: "#main/output_int_array",
                    type: {
                        items: "int",
                        type: "array",
                    },
                },
                output: {
                    output_int_array: [10, 11, 12],
                },
            },
            {
                definition: {
                    id: "#main/output_long_array",
                    type: {
                        items: "long",
                        type: "array",
                    },
                },
                output: {
                    output_long_array: [51, 52],
                },
            },
            {
                definition: {
                    id: "#main/output_float_array",
                    type: {
                        items: "float",
                        type: "array",
                    },
                },
                output: {
                    output_float_array: [100.2, 100.4, 100.6],
                },
            },
            {
                definition: {
                    id: "#main/output_double_array",
                    type: {
                        items: "double",
                        type: "array",
                    },
                },
                output: {
                    output_double_array: [100.1, 100.2, 100.3],
                },
            },
            {
                definition: {
                    id: "#main/output_string_array",
                    type: {
                        items: "string",
                        type: "array",
                    },
                },
                output: {
                    output_string_array: ["Hello", "Output", "!"],
                },
            },
        ];

        const verifyIOParameter = (name, label, doc, val, collection, multipleRows) => {
            cy.get("table tr")
                .contains(name)
                .parents("tr")
                .within($mainRow => {
                    cy.get($mainRow).scrollIntoView();
                    label && cy.contains(label);

                    if (multipleRows) {
                        cy.get($mainRow).nextUntil('[data-cy="process-io-param"]').as("secondaryRows");
                        if (val) {
                            if (Array.isArray(val)) {
                                val.forEach(v => cy.get("@secondaryRows").contains(v));
                            } else {
                                cy.get("@secondaryRows").contains(val);
                            }
                        }
                        if (collection) {
                            cy.get("@secondaryRows").contains(collection);
                        }
                    } else {
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
                    }
                });
        };

        const verifyIOParameterImage = (name, url) => {
            cy.get("table tr")
                .contains(name)
                .parents("tr")
                .within(() => {
                    cy.get('[alt="Inline Preview"]')
                        .should("be.visible")
                        .and($img => {
                            expect($img[0].naturalWidth).to.be.greaterThan(0);
                            expect($img[0].src).contains(url);
                        });
                });
        };

        it("displays IO parameters with keep links and previews", function () {
            // Create output collection for real files
            cy.createCollection(adminUser.token, {
                name: `Test collection ${Math.floor(Math.random() * 999999)}`,
                owner_uuid: activeUser.user.uuid,
            }).then(testOutputCollection => {
                cy.loginAs(activeUser);

                cy.goToPath(`/collections/${testOutputCollection.uuid}`);

                cy.get("[data-cy=upload-button]").click();

                cy.fixture("files/cat.png", "base64").then(content => {
                    cy.get("[data-cy=drag-and-drop]").upload(content, "cat.png");
                    cy.get("[data-cy=form-submit-btn]").click();
                    cy.waitForDom().get("[data-cy=form-submit-btn]").should("not.exist");
                    // Confirm final collection state.
                    cy.get("[data-cy=collection-files-panel]").contains("cat.png").should("exist");
                });

                cy.getCollection(activeUser.token, testOutputCollection.uuid).as("testOutputCollection");
            });

            // Get updated collection pdh
            cy.getAll("@testOutputCollection").then(([testOutputCollection]) => {
                // Add output uuid and inputs to container request
                cy.intercept({ method: "GET", url: "**/arvados/v1/container_requests/*" }, req => {
                    req.on('response', res => {
                        if (!res.body.mounts) {
                            return;
                        }
                        res.body.output_uuid = testOutputCollection.uuid;
                        res.body.mounts["/var/lib/cwl/cwl.input.json"] = {
                            content: testInputs.map(param => param.input).reduce((acc, val) => Object.assign(acc, val), {}),
                        };
                        res.body.mounts["/var/lib/cwl/workflow.json"] = {
                            content: {
                                $graph: [
                                    {
                                        id: "#main",
                                        inputs: testInputs.map(input => input.definition),
                                        outputs: testOutputs.map(output => output.definition),
                                    },
                                ],
                            },
                        };
                    });
                });

                // Stub fake output collection
                cy.intercept(
                    { method: "GET", url: `**/arvados/v1/collections/${testOutputCollection.uuid}*` },
                    {
                        statusCode: 200,
                        body: {
                            uuid: testOutputCollection.uuid,
                            portable_data_hash: testOutputCollection.portable_data_hash,
                        },
                    }
                );

                // Stub fake output json
                cy.intercept(
                    { method: "GET", url: "**/c%3Dzzzzz-4zz18-zzzzzzzzzzzzzzz/cwl.output.json" },
                    {
                        statusCode: 200,
                        body: testOutputs.map(param => param.output).reduce((acc, val) => Object.assign(acc, val), {}),
                    }
                );

                // Stub webdav response, points to output json
                cy.intercept(
                    { method: "PROPFIND", url: "*" },
                    {
                        fixture: "webdav-propfind-outputs.xml",
                    }
                );
            });

            createContainerRequest(activeUser, "test_container_request", "arvados/jobs", ["echo", "hello world"], false, "Committed").as(
                "containerRequest"
            );

            cy.getAll("@containerRequest", "@testOutputCollection").then(function ([containerRequest, testOutputCollection]) {
                cy.goToPath(`/processes/${containerRequest.uuid}`);
                cy.get("[data-cy=process-io-card] h6")
                    .contains("Input Parameters")
                    .parents("[data-cy=process-io-card]")
                    .within((ctx) => {
                        cy.get(ctx).scrollIntoView();
                        verifyIOParameter("input_file", null, "Label Description", "input1.tar", "00000000000000000000000000000000+01");
                        verifyIOParameter("input_file", null, "Label Description", "input1-2.txt", undefined, true);
                        verifyIOParameter("input_file", null, "Label Description", "input1-3.txt", undefined, true);
                        verifyIOParameter("input_file", null, "Label Description", "input1-4.txt", undefined, true);
                        verifyIOParameter("input_dir", null, "Doc Description", "/", "11111111111111111111111111111111+01");
                        verifyIOParameter("input_bool", null, "Doc desc 1, Doc desc 2", "true");
                        verifyIOParameter("input_int", null, null, "1");
                        verifyIOParameter("input_long", null, null, "1");
                        verifyIOParameter("input_float", null, null, "1.5");
                        verifyIOParameter("input_double", null, null, "1.3");
                        verifyIOParameter("input_string", null, null, "Hello World");
                        verifyIOParameter("input_file_array", null, null, "input2.tar", "00000000000000000000000000000000+02");
                        verifyIOParameter("input_file_array", null, null, "input3.tar", undefined, true);
                        verifyIOParameter("input_file_array", null, null, "input3-2.txt", undefined, true);
                        verifyIOParameter("input_file_array", null, null, "Cannot display value", undefined, true);
                        verifyIOParameter("input_dir_array", null, null, "/", "11111111111111111111111111111111+02");
                        verifyIOParameter("input_dir_array", null, null, "/", "11111111111111111111111111111111+03", true);
                        verifyIOParameter("input_dir_array", null, null, "Cannot display value", undefined, true);
                        verifyIOParameter("input_int_array", null, null, ["1", "3", "5", "Cannot display value"]);
                        verifyIOParameter("input_long_array", null, null, ["10", "20", "Cannot display value"]);
                        verifyIOParameter("input_float_array", null, null, ["10.2", "10.4", "10.6", "Cannot display value"]);
                        verifyIOParameter("input_double_array", null, null, ["20.1", "20.2", "20.3", "Cannot display value"]);
                        verifyIOParameter("input_string_array", null, null, ["Hello", "World", "!", "Cannot display value"]);
                        verifyIOParameter("input_bool_include", null, null, "Cannot display value");
                        verifyIOParameter("input_int_include", null, null, "Cannot display value");
                        verifyIOParameter("input_float_include", null, null, "Cannot display value");
                        verifyIOParameter("input_string_include", null, null, "Cannot display value");
                        verifyIOParameter("input_file_include", null, null, "Cannot display value");
                        verifyIOParameter("input_directory_include", null, null, "Cannot display value");
                        verifyIOParameter("input_file_url", null, null, "http://example.com/index.html");
                    });
                cy.get("[data-cy=process-io-card] h6")
                    .contains("Output Parameters")
                    .parents("[data-cy=process-io-card]")
                    .within(ctx => {
                        cy.get(ctx).scrollIntoView();
                        const outPdh = testOutputCollection.portable_data_hash;

                        verifyIOParameter("output_file", null, "Label Description", "cat.png", `${outPdh}`);
                        // Disabled until image preview returns
                        // verifyIOParameterImage("output_file", `/c=${outPdh}/cat.png`);
                        verifyIOParameter("output_file_with_secondary", null, "Doc Description", "main.dat", `${outPdh}`);
                        verifyIOParameter("output_file_with_secondary", null, "Doc Description", "secondary.dat", undefined, true);
                        verifyIOParameter("output_file_with_secondary", null, "Doc Description", "secondary2.dat", undefined, true);
                        verifyIOParameter("output_dir", null, "Doc desc 1, Doc desc 2", "outdir1", `${outPdh}`);
                        verifyIOParameter("output_bool", null, null, "true");
                        verifyIOParameter("output_int", null, null, "1");
                        verifyIOParameter("output_long", null, null, "1");
                        verifyIOParameter("output_float", null, null, "100.5");
                        verifyIOParameter("output_double", null, null, "100.3");
                        verifyIOParameter("output_string", null, null, "Hello output");
                        verifyIOParameter("output_file_array", null, null, "output2.tar", `${outPdh}`);
                        verifyIOParameter("output_file_array", null, null, "output3.tar", undefined, true);
                        verifyIOParameter("output_dir_array", null, null, "outdir2", `${outPdh}`);
                        verifyIOParameter("output_dir_array", null, null, "outdir3", undefined, true);
                        verifyIOParameter("output_int_array", null, null, ["10", "11", "12"]);
                        verifyIOParameter("output_long_array", null, null, ["51", "52"]);
                        verifyIOParameter("output_float_array", null, null, ["100.2", "100.4", "100.6"]);
                        verifyIOParameter("output_double_array", null, null, ["100.1", "100.2", "100.3"]);
                        verifyIOParameter("output_string_array", null, null, ["Hello", "Output", "!"]);
                    });
            });
        });

        it("displays IO parameters with no value", function () {
            const fakeOutputUUID = "zzzzz-4zz18-abcdefghijklmno";
            const fakeOutputPDH = "11111111111111111111111111111111+99/";

            cy.loginAs(activeUser);

            // Add output uuid and inputs to container request
            cy.intercept({ method: "GET", url: "**/arvados/v1/container_requests/*" }, req => {
                req.on('response', res => {
                    if (!res.body.mounts) {
                        return;
                    }
                    res.body.output_uuid = fakeOutputUUID;
                    res.body.mounts["/var/lib/cwl/cwl.input.json"] = {
                        content: {},
                    };
                    res.body.mounts["/var/lib/cwl/workflow.json"] = {
                        content: {
                            $graph: [
                                {
                                    id: "#main",
                                    inputs: testInputs.map(input => input.definition),
                                    outputs: testOutputs.map(output => output.definition),
                                },
                            ],
                        },
                    };
                });
            });

            // Stub fake output collection
            cy.intercept(
                { method: "GET", url: `**/arvados/v1/collections/${fakeOutputUUID}*` },
                {
                    statusCode: 200,
                    body: {
                        uuid: fakeOutputUUID,
                        portable_data_hash: fakeOutputPDH,
                    },
                }
            );

            // Stub fake output json
            cy.intercept(
                { method: "GET", url: `**/c%3D${fakeOutputUUID}/cwl.output.json` },
                {
                    statusCode: 200,
                    body: {},
                }
            );

            cy.readFile("cypress/fixtures/webdav-propfind-outputs.xml").then(data => {
                // Stub webdav response, points to output json
                cy.intercept(
                    { method: "PROPFIND", url: "*" },
                    {
                        statusCode: 200,
                        body: data.replace(/zzzzz-4zz18-zzzzzzzzzzzzzzz/g, fakeOutputUUID),
                    }
                );
            });

            createContainerRequest(activeUser, "test_container_request", "arvados/jobs", ["echo", "hello world"], false, "Committed").as(
                "containerRequest"
            );

            cy.getAll("@containerRequest").then(function ([containerRequest]) {
                cy.goToPath(`/processes/${containerRequest.uuid}`);
                cy.waitForDom();

                cy.get("[data-cy=process-io-card] h6")
                    .contains("Input Parameters")
                    .parents("[data-cy=process-io-card]")
                    .within((ctx) => {
                        cy.get(ctx).scrollIntoView();
                        cy.wait(2000);
                        cy.waitForDom();

                        testInputs.map((input) => {
                            verifyIOParameter(input.definition.id.split('/').slice(-1)[0], null, null, "No value");
                        });
                    });
                cy.get("[data-cy=process-io-card] h6")
                    .contains("Output Parameters")
                    .parents("[data-cy=process-io-card]")
                    .within((ctx) => {
                        cy.get(ctx).scrollIntoView();

                        testOutputs.map((output) => {
                            verifyIOParameter(output.definition.id.split('/').slice(-1)[0], null, null, "No value");
                        });
                    });
            });
        });
    });
});
