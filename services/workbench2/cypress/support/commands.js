// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

// ***********************************************
// This example commands.js shows you how to
// create various custom commands and overwrite
// existing commands.
//
// For more comprehensive examples of custom
// commands please read more here:
// https://on.cypress.io/custom-commands
// ***********************************************
//
//
// -- This is a parent command --
// Cypress.Commands.add("login", (email, password) => { ... })
//
//
// -- This is a child command --
// Cypress.Commands.add("drag", { prevSubject: 'element'}, (subject, options) => { ... })
//
//
// -- This is a dual command --
// Cypress.Commands.add("dismiss", { prevSubject: 'optional'}, (subject, options) => { ... })
//
//
// -- This will overwrite an existing command --
// Cypress.Commands.overwrite("visit", (originalFn, url, options) => { ... })

import 'cypress-wait-until';
import { extractFilesData } from "services/collection-service/collection-service-files-response";
import _ from 'lodash';

const controllerURL = Cypress.env("controller_url");
const systemToken = Cypress.env("system_token");
let createdResources = [];

const containerLogFolderPrefix = "log for container ";

// Clean up anything that was created.  You can temporarily add
// 'return' to the top if you need the resources to hang around to
// debug a specific test.
afterEach(function () {
    if (createdResources.length === 0) {
        return;
    }
    cy.log(`Cleaning ${createdResources.length} previously created resource(s).`);
    // delete them in FIFO order because later created resources may
    // be linked to the earlier ones.
    createdResources.reverse().forEach(function ({ suffix, uuid }) {
        // Don't fail when a resource isn't already there, some objects may have
        // been removed, directly or indirectly, from the test that created them.
        cy.deleteResource(systemToken, suffix, uuid, false);
    });
    createdResources = [];
});

Cypress.Commands.add(
    "doRequest",
    (method = "GET", path = "", data = null, qs = null, token = systemToken, auth = false, followRedirect = true, failOnStatusCode = true) => {
        return cy.request({
            method: method,
            url: `${controllerURL.replace(/\/+$/, "")}/${path.replace(/^\/+/, "")}`,
            body: data,
            qs: auth ? qs : Object.assign({ api_token: token }, qs),
            auth: auth ? { bearer: `${token}` } : undefined,
            followRedirect: followRedirect,
            failOnStatusCode: failOnStatusCode,
        });
    }
);

Cypress.Commands.add(
    "doWebDAVRequest",
    (method = "GET", path = "", data = null, qs = null, token = systemToken, auth = false, followRedirect = true, failOnStatusCode = true) => {
        return cy.doRequest("GET", "/arvados/v1/config", null, null).then(({ body: config }) => {
            return cy.request({
                method: method,
                url: `${config.Services.WebDAVDownload.ExternalURL.replace(/\/+$/, "")}/${path.replace(/^\/+/, "")}`,
                body: data,
                qs: auth ? qs : Object.assign({ api_token: token }, qs),
                auth: auth ? { bearer: `${token}` } : undefined,
                followRedirect: followRedirect,
                failOnStatusCode: failOnStatusCode,
            });
        });
    }
);

Cypress.Commands.add("getUser", (username, first_name = "", last_name = "", is_admin = false, is_active = true) => {
    // Create user if not already created
    return (
        cy
            .doRequest(
                "POST",
                "/auth/controller/callback",
                {
                    auth_info: JSON.stringify({
                        email: `${username}@example.local`,
                        username: username,
                        first_name: first_name,
                        last_name: last_name,
                        alternate_emails: [],
                    }),
                    return_to: ",https://controller.api.client.invalid",
                },
                null,
                systemToken,
                true,
                false
            ) // Don't follow redirects so we can catch the token
            .its("headers.location")
            .as("location")
            // Get its token and set the account up as admin and/or active
            .then(function () {
                this.userToken = this.location.split("=")[1];
                assert.isString(this.userToken);
                return cy
                    .doRequest("GET", "/arvados/v1/users", null, {
                        filters: `[["username", "=", "${username}"]]`,
                    })
                    .its("body.items.0")
                    .as("aUser")
                    .then(function () {
                        cy.doRequest("PUT", `/arvados/v1/users/${this.aUser.uuid}`, {
                            user: {
                                is_admin: is_admin,
                                is_active: is_active,
                            },
                        })
                            .its("body")
                            .as("theUser")
                            .then(function () {
                                return { user: this.theUser, token: this.userToken };
                            });
                    });
            })
    );
});

Cypress.Commands.add("createLink", (token, data) => {
    return cy.createResource(token, "links", {
        link: JSON.stringify(data),
    });
});

Cypress.Commands.add("createGroup", (token, data) => {
    return cy.createResource(token, "groups", {
        group: JSON.stringify(data),
        ensure_unique_name: true,
    });
});

Cypress.Commands.add("trashGroup", (token, uuid) => {
    return cy.deleteResource(token, "groups", uuid);
});

Cypress.Commands.add("createWorkflow", (token, data) => {
    return cy.createResource(token, "workflows", {
        workflow: JSON.stringify(data),
        ensure_unique_name: true,
    });
});

Cypress.Commands.add("createCollection", (token, data, keep = false) => {
    return cy.createResource(token, "collections", {
        collection: JSON.stringify(data),
        ensure_unique_name: true,
    }, keep);
});

Cypress.Commands.add("getCollection", (token, uuid) => {
    return cy.getResource(token, "collections", uuid);
});

Cypress.Commands.add("updateCollection", (token, uuid, data) => {
    return cy.updateResource(token, "collections", uuid, {
        collection: JSON.stringify(data),
    });
});

Cypress.Commands.add("collectionReplaceFiles", (token, uuid, data) => {
    return cy.updateResource(token, "collections", uuid, {
        collection: {
            preserve_version: true,
        },
        replace_files: JSON.stringify(data),
    });
});

Cypress.Commands.add("getContainer", (token, uuid) => {
    return cy.getResource(token, "containers", uuid);
});

Cypress.Commands.add("updateContainer", (token, uuid, data) => {
    return cy.updateResource(token, "containers", uuid, {
        container: JSON.stringify(data),
    });
});

Cypress.Commands.add("getContainerRequest", (token, uuid) => {
    return cy.getResource(token, "container_requests", uuid);
});

Cypress.Commands.add("createContainerRequest", (token, data) => {
    return cy.createResource(token, "container_requests", {
        container_request: JSON.stringify(data),
        ensure_unique_name: true,
    });
});

Cypress.Commands.add("updateContainerRequest", (token, uuid, data) => {
    return cy.updateResource(token, "container_requests", uuid, {
        container_request: JSON.stringify(data),
    });
});

/**
 * Requires an admin token for log_uuid modification to succeed
 */
Cypress.Commands.add("appendLog", (token, crUuid, fileName, lines = []) =>
    cy.getContainerRequest(token, crUuid).then(containerRequest => {
        if (containerRequest.log_uuid) {
            cy.listContainerRequestLogs(token, crUuid).then(logFiles => {
                const filePath = `${containerRequest.log_uuid}/${containerLogFolderPrefix}${containerRequest.container_uuid}/${fileName}`;
                if (logFiles.find(file => file.name === fileName)) {
                    // File exists, fetch and append
                    return cy
                        .doWebDAVRequest("GET", `c=${filePath}`, null, null, token)
                        .then(({ body: contents }) =>
                            cy.doWebDAVRequest("PUT", `c=${filePath}`, contents.split("\n").concat(lines).join("\n"), null, token)
                        );
                } else {
                    // File not exists, put new file
                    cy.doWebDAVRequest("PUT", `c=${filePath}`, lines.join("\n"), null, token);
                }
            });
        } else {
            // Create log collection
            return cy
                .createCollection(token, {
                    name: `Test log collection ${Math.floor(Math.random() * 999999)}`,
                    owner_uuid: containerRequest.owner_uuid,
                    manifest_text: "",
                })
                .then(collection => {
                    // Update CR log_uuid to fake log collection
                    cy.updateContainerRequest(token, containerRequest.uuid, {
                        log_uuid: collection.uuid,
                    }).then(() =>
                        // Create empty directory for container uuid
                        cy
                            .collectionReplaceFiles(token, collection.uuid, {
                                [`/${containerLogFolderPrefix}${containerRequest.container_uuid}`]: "d41d8cd98f00b204e9800998ecf8427e+0",
                            })
                            .then(() =>
                                // Put new log file with contents into fake log collection
                                cy.doWebDAVRequest(
                                    "PUT",
                                    `c=${collection.uuid}/${containerLogFolderPrefix}${containerRequest.container_uuid}/${fileName}`,
                                    lines.join("\n"),
                                    null,
                                    token
                                )
                            )
                    );
                });
        }
    })
);

Cypress.Commands.add("listContainerRequestLogs", (token, crUuid) =>
    cy.getContainerRequest(token, crUuid).then(containerRequest =>
        cy
            .doWebDAVRequest(
                "PROPFIND",
                `c=${containerRequest.log_uuid}/${containerLogFolderPrefix}${containerRequest.container_uuid}`,
                null,
                null,
                token
            )
            .then(({ body: data }) => {
                return extractFilesData(new DOMParser().parseFromString(data, "text/xml"));
            })
    )
);

Cypress.Commands.add("createVirtualMachine", (token, data) => {
    return cy.createResource(token, "virtual_machines", {
        virtual_machine: JSON.stringify(data),
        ensure_unique_name: true,
    });
});

Cypress.Commands.add("getResource", (token, suffix, uuid) => {
    return cy
        .doRequest("GET", `/arvados/v1/${suffix}/${uuid}`, null, {}, token)
        .its("body")
        .then(function (resource) {
            return resource;
        });
});

Cypress.Commands.add("createResource", (token, suffix, data, keep = false) => {
    return cy
        .doRequest("POST", "/arvados/v1/" + suffix, data, null, token, true)
        .its("body")
        .then(function (resource) {
            if (! keep) {
                createdResources.push({ suffix, uuid: resource.uuid });
            };
            return resource;
        });
});


Cypress.Commands.add("deleteResource", (token, suffix, uuid, failOnStatusCode = true) => {
    return cy
        .doRequest("DELETE", "/arvados/v1/" + suffix + "/" + uuid, null, null, token, false, true, failOnStatusCode)
        .its("body")
        .then(function (resource) {
            return resource;
        });
});

Cypress.Commands.add("updateResource", (token, suffix, uuid, data) => {
    return cy
        .doRequest("PATCH", "/arvados/v1/" + suffix + "/" + uuid, data, null, token, true)
        .its("body")
        .then(function (resource) {
            return resource;
        });
});

Cypress.Commands.add("loginAs", (user, preserveLocalStorage = false) => {
    // This shouldn't be necessary unless we need to call loginAs multiple times
    // in the same test.
    cy.clearCookies();
    if(preserveLocalStorage === false) {
        cy.clearAllLocalStorage();
        cy.clearAllSessionStorage();
    }
    cy.visit(`/token/?api_token=${user.token}`);
    // Use waitUntil to avoid permafail race conditions with window.location being undefined
    cy.waitUntil(() => cy.window().then(win =>
        win?.location?.href &&
        win.location.href.includes("/projects/")
    ), { timeout: 15000 });
    // Wait for page to settle before getting elements
    cy.waitForDom();
    cy.get("div#root").should("contain", "Arvados Workbench (zzzzz)");
    cy.get("div#root").should("not.contain", "Your account is inactive");
});

Cypress.Commands.add("testEditProjectOrCollection", (container, oldName, newName, newDescription, isProject = true) => {
    cy.get(container).contains(oldName).rightclick();
    cy.get("[data-cy=context-menu]")
        .contains(isProject ? "Edit project" : "Edit collection")
        .click();
    cy.get("[data-cy=form-dialog]").within(() => {
        cy.get("input[name=name]").clear().type(newName);
        cy.get("div[contenteditable=true]")
            .clear()
            .type(newDescription);
        cy.get("[data-cy=form-submit-btn]").click();
    });

    cy.get(container).contains(newName).rightclick();
    cy.get("[data-cy=context-menu]")
        .contains(isProject ? "Edit project" : "Edit collection")
        .click();
    cy.get("[data-cy=form-dialog]").within(() => {
        cy.get("input[name=name]").should("have.value", newName);
        cy.get("span[data-text=true]").contains(newDescription);
        cy.get("[data-cy=form-cancel-btn]").click();
    });
});

Cypress.Commands.add("doSearch", searchTerm => {
    cy.get("[data-cy=searchbar-input-field]").type(`{selectall}${searchTerm}{enter}`);
});

Cypress.Commands.add("goToPath", path => {
    return cy.visit(path);
});

Cypress.Commands.add("getAll", (...elements) => {
    const promise = cy.wrap([], { log: false });

    for (let element of elements) {
        promise.then(arr => cy.get(element).then(got => cy.wrap([...arr, got])));
    }

    return promise;
});

Cypress.Commands.add("shareWith", (srcUserToken, targetUserUUID, itemUUID, permission = "can_write") => {
    cy.createLink(srcUserToken, {
        name: permission,
        link_class: "permission",
        head_uuid: itemUUID,
        tail_uuid: targetUserUUID,
    });
});

Cypress.Commands.add("addToFavorites", (userToken, userUUID, itemUUID) => {
    cy.createLink(userToken, {
        head_uuid: itemUUID,
        link_class: "star",
        name: "",
        owner_uuid: userUUID,
        tail_uuid: userUUID,
    });
});

Cypress.Commands.add("createProject", ({ owningUser, targetUser, projectName, canWrite, addToFavorites }) => {
    const writePermission = canWrite ? "can_write" : "can_read";

    cy.createGroup(owningUser.token, {
        name: `${projectName} ${Math.floor(Math.random() * 999999)}`,
        group_class: "project",
    })
        .as(`${projectName}`)
        .then(project => {
            if (targetUser && targetUser !== owningUser) {
                cy.shareWith(owningUser.token, targetUser.user.uuid, project.uuid, writePermission);
            }
            if (addToFavorites) {
                const user = targetUser ? targetUser : owningUser;
                cy.addToFavorites(user.token, user.user.uuid, project.uuid);
            }
        });
});

Cypress.Commands.add(
    "upload",
    {
        prevSubject: "element",
    },
    (subject, file, fileName, binaryMode = true) => {
        cy.window().then(window => {
            const blob = binaryMode ? b64toBlob(file, "", 512) : new Blob([file], { type: "text/plain" });
            const testFile = new window.File([blob], fileName);

            const dataTransferFile = new File([testFile], testFile.name, {
                type: 'text/plain',
            });

            const data = {
                dataTransfer: {
                    files: [dataTransferFile],
                    items: [
                        {
                            kind: 'dataFile',
                            type: dataTransferFile.type,
                            getAsFile: () => dataTransferFile,
                            webkitGetAsEntry: () => ({
                                isFile: true,
                                isDirectory: false,
                                file: (cb) => cb(dataTransferFile),
                            }),
                        },
                    ],
                    types: ['Files'],
                },
            };

            cy.wrap(subject)
                .trigger('dragenter', data)
                .trigger('dragover', data)
                .trigger('drop', data)
            });
        }
    );

function b64toBlob(b64Data, contentType = "", sliceSize = 512) {
    const byteCharacters = atob(b64Data);
    const byteArrays = [];

    for (let offset = 0; offset < byteCharacters.length; offset += sliceSize) {
        const slice = byteCharacters.slice(offset, offset + sliceSize);

        const byteNumbers = new Array(slice.length);
        for (let i = 0; i < slice.length; i++) {
            byteNumbers[i] = slice.charCodeAt(i);
        }

        const byteArray = new Uint8Array(byteNumbers);

        byteArrays.push(byteArray);
    }

    const blob = new Blob(byteArrays, { type: contentType });
    return blob;
}

// From https://github.com/cypress-io/cypress/issues/7306#issuecomment-1076451070=
// This command requires the async package (https://www.npmjs.com/package/async)
Cypress.Commands.add("waitForDom", () => {
    cy.window({ timeout: 10000 }).then(
        {
            // Don't timeout before waitForDom finishes
            timeout: 10000,
        },
        win => {
            let timeElapsed = 0;

            cy.log("Waiting for DOM mutations to complete");

            return new Cypress.Promise(resolve => {
                // set the required variables
                let async = require("async");
                let observerConfig = { attributes: true, childList: true, subtree: true };
                let items = Array.apply(null, { length: 50 }).map(Number.call, Number);
                win.mutationCount = 0;
                win.previousMutationCount = null;

                // create an observer instance
                let observer = new win.MutationObserver(mutations => {
                    mutations.forEach(mutation => {
                        // Only record "attributes" type mutations that are not a "class" mutation.
                        // If the mutation is not an "attributes" type, then we always record it.
                        if (mutation.type === "attributes" && mutation.attributeName !== "class") {
                            win.mutationCount += 1;
                        } else if (mutation.type !== "attributes") {
                            win.mutationCount += 1;
                        }
                    });

                    // initialize the previousMutationCount
                    if (win.previousMutationCount == null) win.previousMutationCount = 0;
                });

                // watch the document body for the specified mutations
                observer.observe(win.document.body, observerConfig);

                // check the DOM for mutations up to 50 times for a maximum time of 5 seconds
                async.eachSeries(
                    items,
                    function iteratee(item, callback) {
                        // keep track of the elapsed time so we can log it at the end of the command
                        timeElapsed = timeElapsed + 100;

                        // make each iteration of the loop 100ms apart
                        setTimeout(() => {
                            if (win.mutationCount === win.previousMutationCount) {
                                // pass an argument to the async callback to exit the loop
                                return callback("Resolved - DOM changes complete.");
                            } else if (win.previousMutationCount != null) {
                                // only set the previous count if the observer has checked the DOM at least once
                                win.previousMutationCount = win.mutationCount;
                                return callback();
                            } else if (win.mutationCount === 0 && win.previousMutationCount == null && item === 4) {
                                // this is an early exit in case nothing is changing in the DOM. That way we only
                                // wait 500ms instead of the full 5 seconds when no DOM changes are occurring.
                                return callback("Resolved - Exiting early since no DOM changes were detected.");
                            } else {
                                // proceed to the next iteration
                                return callback();
                            }
                        }, 100);
                    },
                    function done() {
                        // Log the total wait time so users can see it
                        cy.log(`DOM mutations ${timeElapsed >= 5000 ? "did not complete" : "completed"} in ${timeElapsed} ms`);

                        // disconnect the observer and resolve the promise
                        observer.disconnect();
                        resolve();
                    }
                );
            });
        }
    );
});

Cypress.Commands.add('waitForLocalStorage', (key, options = {}) => {
    const timeout = options.timeout || 10000;
    const interval = options.interval || 100;

    cy.log(`Waiting for localStorage key: ${key}`)

    const checkLocalStorage = () => {
      return new Cypress.Promise((resolve, reject) => {
        const startTime = Date.now();

        const check = () => {
          const value = localStorage.getItem(key);

          if (value !== null) {
            resolve(value);
          } else if (Date.now() - startTime > timeout) {
            reject(new Error(`Timed out waiting for localStorage key: ${key}`));
          } else {
            setTimeout(check, interval);
          }
        };

        check();
      });
    };

    return cy.wrap(checkLocalStorage());
  });

  //pauses test execution until the localStorage key changes
  Cypress.Commands.add('waitForLocalStorageUpdate', (key, timeout = 10000) => {
    const checkInterval = 200; // Interval to check the localStorage value
    let previousValue = localStorage.getItem(key);

    return new Cypress.Promise((resolve, reject) => {
      const checkValue = () => {
        const currentValue = localStorage.getItem(key);
        if (currentValue !== previousValue) {
          resolve(currentValue);
        } else if (Date.now() - startTime >= timeout) {
          reject(new Error(`Timed out waiting for localStorage key "${key}" to change`));
        } else {
          setTimeout(checkValue, checkInterval);
        }
      };

      const startTime = Date.now();
      checkValue();
    });
  });

Cypress.Commands.add("setupDockerImage", (image_name) => {
    // Create a collection that will be used as a docker image for the tests.
    let activeUser;
    let adminUser;

        cy.getUser("admin", "Admin", "User", true, true)
            .as("adminUser")
            .then(function () {
                adminUser = this.adminUser;
            });

        cy.getUser('activeuser', 'Active', 'User', false, true)
            .as('activeUser').then(function () {
                activeUser = this.activeUser;
            });

    cy.getAll('@activeUser', '@adminUser').then(([activeUser, adminUser]) => {
	cy.createCollection(adminUser.token, {
            name: "docker_image",
            manifest_text:
                ". d21353cfe035e3e384563ee55eadbb2f+67108864 5c77a43e329b9838cbec18ff42790e57+55605760 0:122714624:sha256:d8309758b8fe2c81034ffc8a10c36460b77db7bc5e7b448c4e5b684f9d95a678.tar\n",
        })
            .as("dockerImage")
            .then(function (dockerImage) {
                // Give read permissions to the active user on the docker image.
                cy.createLink(adminUser.token, {
                    link_class: "permission",
                    name: "can_read",
                    tail_uuid: activeUser.user.uuid,
                    head_uuid: dockerImage.uuid,
                })
                    .as("dockerImagePermission")
                    .then(function () {
                        // Set-up docker image collection tags
                        cy.createLink(activeUser.token, {
                            link_class: "docker_image_repo+tag",
                            name: image_name,
                            head_uuid: dockerImage.uuid,
                        }).as("dockerImageRepoTag");
                        cy.createLink(activeUser.token, {
                            link_class: "docker_image_hash",
                            name: "sha256:d8309758b8fe2c81034ffc8a10c36460b77db7bc5e7b448c4e5b684f9d95a678",
                            head_uuid: dockerImage.uuid,
                        }).as("dockerImageHash");
                    });
            });
    });
    return cy.getAll("@dockerImage", "@dockerImageRepoTag", "@dockerImageHash", "@dockerImagePermission").then(function ([dockerImage]) {
        return dockerImage;
    });
});

/**
 * Asserts the url path exactly matches (ignores host and hash)
 *
 * @returns the path for further use if needed
 */
Cypress.Commands.add("assertUrlPathname", (path) => {
    cy.waitForDom();
    return cy.waitUntil(() => cy.location()
        .then((url) => url.pathname.endsWith(path) ? url.pathname : false)
    , {
        errorMsg: "Timed out waiting for URL path to match: " + path,
        timeout: 10000,
    });
});

/**
 * Performs nagation to the specified account menu item
 */
Cypress.Commands.add("doAccountMenuAction", (name) => {
    cy.get('[data-cy=dropdown-menu-button][aria-label="Account Management"]').click();
    cy.get('#account-menu li[role=menuitem]').contains(name).click();
    cy.waitForDom();
});

/**
 * Clicks on the first side panel item exactly matching name
 */
Cypress.Commands.add("doSidePanelNavigation", (name) => {
    cy.waitForDom();
    cy.get('[data-cy=tree-li]')
        .contains(new RegExp("^" + name + "$", "g"))
        .click();
    cy.waitForDom();
});

/**
 * Clicks on the first breadcrumb exactly matching the passed name
 */
Cypress.Commands.add("doBreadcrumbsNavigation", (name) => {
    cy.waitForDom();
    cy.get('[data-cy=breadcrumbs] button')
        .contains(new RegExp("^" + name + "$", "g"))
        .click();
    cy.waitForDom();
});

/**
 * Assert exact breadcrumb contents
 *
 * @returns the current breadcrumbs as a string array
 */
Cypress.Commands.add("assertBreadcrumbs", (names) => {
    cy.waitForDom();
    // waitUntil allows retrying with a non-assert test
    // This allows doing non-wrapped comparisons and improves retryability
    return cy.waitUntil(() => cy.get('[data-cy=breadcrumbs] button')
        .then(crumbs => {
            // Everything within must not be a chai/cypress assertion
            // otherwise it will fail and not retry
            const crumbNames = crumbs.toArray().map(crumb => crumb.innerText);
            return _.isEqual(crumbNames, names) ? crumbNames : false;
        })
    , {
        errorMsg: `Timed out waiting for breadcrumbs to match: [ ${names.map(str => `"${str}"`).join(", ")} ]`,
        timeout: 10000,
    });
});

/**
 * Asserts whether the DE contains a certain item, default to true
 */
Cypress.Commands.add("assertDataExplorerContains", (name, contains = true) => {
    cy.waitForDom();
    contains
        ? cy.get('[data-cy=data-table]').contains(name).should('exist')
        : cy.get('[data-cy=data-table]').contains(name).should('not.exist');
});

/**
 * Finds the toolbar buttons and clicks the one exactly matching name
 *
 * Does not currently handle specifying which toolbar (DE or details card) or handling collapsed toolbar
 */
Cypress.Commands.add("doToolbarAction", (name) => {
    cy.get(`[data-cy=multiselect-toolbar] [data-cy=multiselect-button][aria-label="${name}"]`, { timeout: 5000 }).click();
});

/**
 * Perform a context menu action on an item in the data explorer
 */
Cypress.Commands.add("doDataExplorerContextAction", (name, action) => {
    cy.waitForDom();
    cy.get('[data-cy=data-table]', { timeout: 10000 }).contains(name, { timeout: 10000 }).rightclick();
    cy.doContextMenuAction(action);
});

/**
 * Perform a collections panel options menu action (the top-right menu, not right click)
 */
Cypress.Commands.add("doCollectionPanelOptionsAction", (action) => {
    cy.waitForDom();
    cy.get("[data-cy=collection-files-panel-options-btn]", { timeout: 10000 }).click();
    cy.doContextMenuAction(action);
});

/**
 * Selects data explorer row checkbox by name
 */
Cypress.Commands.add("doDataExplorerSelect", (name) => {
    cy.waitForDom();
    cy.get('[data-cy=data-table]', { timeout: 10000 })
        .contains(name)
        .parents('[data-cy=data-table-row]')
        .find('input[type=checkbox]')
        .first()
        .click()
        .then(() => cy.waitForDom());
});

/**
 * Selects data explorer rows in the collection files panel
 */
Cypress.Commands.add("doCollectionFileSelect", (name) => {
    cy.waitForDom();
    cy.get('[data-cy=collection-files-right-panel]', { timeout: 10000 })
        .contains(name)
        .parents('[data-item=true]')
        .find('input[type=checkbox]')
        .first()
        .click()
        .then(() => cy.waitForDom());
});

/**
 * Navigates to data explorer item by name
 */
Cypress.Commands.add("doDataExplorerNavigate", (name) => {
    cy.waitForDom();
    cy.get('[data-cy=data-table]', { timeout: 10000 })
        .contains(name)
        .click()
        .then(() => cy.waitForDom());
});

/**
 * Inputs value into data explorer search
 *
 * Useful for when there are too many items in a data explorer for the item of interest to be on the first page
 */
Cypress.Commands.add("doDataExplorerSearch", (value) => {
    cy.waitForDom();
    cy.get('[data-cy=search-input]').clear().type(value);
    cy.waitForDom();
});

/**
 * Changes MPV panel tabs
 */
Cypress.Commands.add("doMPVTabSelect", (tabName) => {
    cy.waitForDom();
    cy.get('[data-cy=mpv-tabs] button').contains(tabName).click();
    cy.waitForDom();
});

/**
 * asserts the correct toolbar buttons are displayed
 * by mousing over and checking for the tooltips
 *
 * @param tooltips array of strings
 * */

Cypress.Commands.add("assertToolbarButtons", (tooltips) => {
    cy.get('[data-cy=multiselect-button]').should('have.length', tooltips.length)
        for (let i = 0; i < tooltips.length; i++) {
            cy.get('[data-cy=multiselect-button]').eq(i).trigger('mouseover');
            cy.get('body').contains(tooltips[i]).should('exist', {timeout: 10000})
            cy.get('[data-cy=multiselect-button]').eq(i).trigger('mouseout');
        }
});

/**
 * Checks if checkboxes associated for resources with uuids are checked or not
 *
 * @param uuids array of uuids
 */

Cypress.Commands.add("assertCheckboxes", (uuids, shouldBeChecked) => {
    uuids.forEach(uuid => {
        cy.get(`input[data-cy="multiselect-checkbox-${uuid}"]`).should(shouldBeChecked ? 'be.checked' : 'not.be.checked');
    });
});

/**
 * Reusable perform context menu action - assumes menu is already open
 */
Cypress.Commands.add("doContextMenuAction", (name) => {
    cy.waitForDom();
    cy.get("[data-cy=context-menu]", { timeout: 5000 }).contains(name).click();
    cy.waitForDom();
});
