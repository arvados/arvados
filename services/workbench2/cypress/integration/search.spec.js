// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

describe('Search tests', function() {
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
        cy.getUser('collectionuser1', 'Collection', 'User', false, true)
            .as('activeUser').then(function() {
                activeUser = this.activeUser;
            }
        );
    })

    beforeEach(function() {
        cy.clearCookies()
        cy.clearLocalStorage()
    })

    it('can search for old collection versions', function() {
        const colName = `Versioned Collection ${Math.floor(Math.random() * Math.floor(999999))}`;
        let colUuid = '';
        let oldVersionUuid = '';
        // Make sure no other collections with this name exist
        cy.doRequest('GET', '/arvados/v1/collections', null, {
            filters: `[["name", "=", "${colName}"]]`,
            include_old_versions: true
        })
        .its('body.items').as('collections')
        .then(function() {
            expect(this.collections).to.be.empty;
        });
        // Creates the collection using the admin token so we can set up
        // a bogus manifest text without block signatures.
        cy.createCollection(adminUser.token, {
            name: colName,
            owner_uuid: activeUser.user.uuid,
            preserve_version: true,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n"})
        .as('originalVersion').then(function() {
            // Change the file name to create a new version.
            cy.updateCollection(adminUser.token, this.originalVersion.uuid, {
                manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:foo\n"
            })
            colUuid = this.originalVersion.uuid;
        });
        // Confirm that there are 2 versions of the collection
        cy.doRequest('GET', '/arvados/v1/collections', null, {
            filters: `[["name", "=", "${colName}"]]`,
            include_old_versions: true
        })
        .its('body.items').as('collections')
        .then(function() {
            expect(this.collections).to.have.lengthOf(2);
            this.collections.map(function(aCollection) {
                expect(aCollection.current_version_uuid).to.equal(colUuid);
                if (aCollection.uuid !== aCollection.current_version_uuid) {
                    oldVersionUuid = aCollection.uuid;
                }
            });
            cy.loginAs(activeUser);
            const searchQuery = `${colName} type:arvados#collection`;
            // Search for only collection's current version
            cy.doSearch(`${searchQuery}`);
            cy.get('[data-cy=search-results]').should('contain', 'head version');
            cy.get('[data-cy=search-results]').should('not.contain', 'version 1');
            // ...and then, include old versions.
            cy.doSearch(`${searchQuery} is:pastVersion`);
            cy.get('[data-cy=search-results]').should('contain', 'head version');
            cy.get('[data-cy=search-results]').should('contain', 'version 1');
        });
    });

    it('can display path of the selected item', function() {
        const colName = `Collection ${Math.floor(Math.random() * Math.floor(999999))}`;

        // Creates the collection using the admin token so we can set up
        // a bogus manifest text without block signatures.
        cy.createCollection(adminUser.token, {
            name: colName,
            owner_uuid: activeUser.user.uuid,
            preserve_version: true,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n"
        }).then(function() {
            cy.loginAs(activeUser);

            cy.doSearch(colName);

            cy.get('[data-cy=search-results]').should('contain', colName);

            cy.get('[data-cy=search-results]').contains(colName).closest('tr').click();

            cy.get('[data-cy=element-path]').should('contain', `/ Projects / ${colName}`);
        });
    });

    it('can search items using quotes', function() {
        const random = Math.floor(Math.random() * Math.floor(999999));
        const colName = `Collection ${random}`;
        const colName2 = `Collection test ${random}`;

        // Creates the collection using the admin token so we can set up
        // a bogus manifest text without block signatures.
        cy.createCollection(adminUser.token, {
            name: colName,
            owner_uuid: activeUser.user.uuid,
            preserve_version: true,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n"
        }).as('collection1');

        cy.createCollection(adminUser.token, {
            name: colName2,
            owner_uuid: activeUser.user.uuid,
            preserve_version: true,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n"
        }).as('collection2');

        cy.getAll('@collection1', '@collection2')
            .then(function() {
                cy.loginAs(activeUser);

                cy.doSearch(colName);
                cy.get('[data-cy=search-results] table tbody tr').should('have.length', 2);

                cy.doSearch(`"${colName}"`);
                cy.get('[data-cy=search-results] table tbody tr').should('have.length', 1);
            });
    });

    it('can display owner of the item', function() {
        const colName = `Collection ${Math.floor(Math.random() * Math.floor(999999))}`;

        cy.createCollection(adminUser.token, {
            name: colName,
            owner_uuid: activeUser.user.uuid,
            preserve_version: true,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n"
        }).then(function() {
            cy.loginAs(activeUser);

            cy.doSearch(colName);

            cy.get('[data-cy=search-results]').should('contain', colName);

            cy.get('[data-cy=search-results]').contains(colName).closest('tr')
                .within(() => {
                    cy.get('p').contains(activeUser.user.uuid).should('contain', activeUser.user.full_name);
                });
        });
    });

    it('shows search context menu', function() {
        const colName = `Collection ${Math.floor(Math.random() * Math.floor(999999))}`;
        const federatedColName = `Collection ${Math.floor(Math.random() * Math.floor(999999))}`;
        const federatedColUuid = "xxxxx-4zz18-000000000000000";

        // Intercept config to insert remote cluster
        cy.intercept({method: 'GET', hostname: 'localhost', url: '**/arvados/v1/config?nocache=*'}, (req) => {
            req.reply((res) => {
                res.body.RemoteClusters = {
                    "*": res.body.RemoteClusters["*"],
                    "xxxxx": {
                        "ActivateUsers": true,
                        "Host": "xxxxx.fakecluster.tld",
                        "Insecure": false,
                        "Proxy": true,
                        "Scheme": ""
                    }
                };
            });
        });

        // Fake remote cluster config
        cy.intercept(
          {
            method: "GET",
            hostname: "xxxxx.fakecluster.tld",
            url: "**/arvados/v1/config",
          },
          {
            statusCode: 200,
            body: {
              API: {},
              ClusterID: "xxxxx",
              Collections: {},
              Containers: {},
              InstanceTypes: {},
              Login: {},
              Mail: { SupportEmailAddress: "arvados@example.com" },
              RemoteClusters: {
                "*": {
                  ActivateUsers: false,
                  Host: "",
                  Insecure: false,
                  Proxy: false,
                  Scheme: "https",
                },
              },
              Services: {
                Composer: { ExternalURL: "" },
                Controller: { ExternalURL: "https://xxxxx.fakecluster.tld:34763/" },
                DispatchCloud: { ExternalURL: "" },
                DispatchLSF: { ExternalURL: "" },
                DispatchSLURM: { ExternalURL: "" },
                GitHTTP: { ExternalURL: "https://xxxxx.fakecluster.tld:39105/" },
                GitSSH: { ExternalURL: "" },
                Health: { ExternalURL: "https://xxxxx.fakecluster.tld:42915/" },
                Keepbalance: { ExternalURL: "" },
                Keepproxy: { ExternalURL: "https://xxxxx.fakecluster.tld:46773/" },
                Keepstore: { ExternalURL: "" },
                RailsAPI: { ExternalURL: "" },
                WebDAV: { ExternalURL: "https://xxxxx.fakecluster.tld:36041/" },
                WebDAVDownload: { ExternalURL: "https://xxxxx.fakecluster.tld:42957/" },
                WebShell: { ExternalURL: "" },
                Websocket: { ExternalURL: "wss://xxxxx.fakecluster.tld:37121/websocket" },
                Workbench1: { ExternalURL: "https://wb1.xxxxx.fakecluster.tld/" },
                Workbench2: { ExternalURL: "https://wb2.xxxxx.fakecluster.tld/" },
              },
              StorageClasses: {
                default: { Default: true, Priority: 0 },
              },
              Users: {},
              Volumes: {},
              Workbench: {},
            },
          }
        );

        cy.createCollection(adminUser.token, {
            name: colName,
            owner_uuid: activeUser.user.uuid,
            preserve_version: true,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n"
        }).then(function(testCollection) {
            cy.loginAs(activeUser);

            // Intercept search results to add federated result
            cy.intercept({method: 'GET', url: '**/arvados/v1/groups/contents?*'}, (req) => {
                req.reply((res) => {
                    res.body.items = [
                        res.body.items[0],
                        {
                            ...res.body.items[0],
                            uuid: federatedColUuid,
                            portable_data_hash: "00000000000000000000000000000000+0",
                            name: federatedColName,
                            href: res.body.items[0].href.replace(testCollection.uuid, federatedColUuid),
                        }
                    ];
                    res.body.items_available += 1;
                });
            });

            cy.doSearch(colName);

            // Stub new window
            cy.window().then(win => {
                cy.stub(win, 'open').as('Open')
            });

            // Check copy to clipboard
            cy.get('[data-cy=search-results]').contains(colName).rightclick();
            cy.get('[data-cy=context-menu]').within((ctx) => {
                // Check that there are 4 items in the menu
                cy.get(ctx).children().should('have.length', 4);
                cy.contains('API Details');
                cy.contains('Copy to clipboard');
                cy.contains('Open in new tab');
                cy.contains('View details');

                cy.contains('Copy to clipboard').click();
                cy.window().then((win) => (
                    win.navigator.clipboard.readText().then((text) => {
                        expect(text).to.match(new RegExp(`/collections/${testCollection.uuid}$`));
                    })
                ));
            });

            // Check open in new tab
            cy.get('[data-cy=search-results]').contains(colName).rightclick();
            cy.get('[data-cy=context-menu]').within(() => {
                cy.contains('Open in new tab').click();
                cy.get('@Open').should('have.been.calledOnceWith', `${window.location.origin}/collections/${testCollection.uuid}`)
            });

            // Check federated result copy to clipboard
            cy.get('[data-cy=search-results]').contains(federatedColName).rightclick();
            cy.get('[data-cy=context-menu]').within(() => {
                cy.contains('Copy to clipboard').click();
                cy.window().then((win) => (
                    win.navigator.clipboard.readText().then((text) => {
                        expect(text).to.equal(`https://wb2.xxxxx.fakecluster.tld/collections/${federatedColUuid}`);
                    })
                ));
            });
            // Check open in new tab
            cy.get('[data-cy=search-results]').contains(federatedColName).rightclick();
            cy.get('[data-cy=context-menu]').within(() => {
                cy.contains('Open in new tab').click();
                cy.get('@Open').should('have.been.calledWith', `https://wb2.xxxxx.fakecluster.tld/collections/${federatedColUuid}`)
            });

        });
    });
});
