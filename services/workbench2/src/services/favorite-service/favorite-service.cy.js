// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { LinkService } from "../link-service/link-service";
import { GroupsService } from "../groups-service/groups-service";
import { FavoriteService } from "./favorite-service";
import { LinkClass } from "models/link";
import { FilterBuilder } from "services/api/filter-builder";
import axios from "axios";
import { isEqual } from "lodash";

describe("FavoriteService", () => {

    let linkService;
    let groupService;

    const mockListArgs = {
        filters: [],
        limit: undefined,
        offset: undefined,
        order: undefined,
    };

    const mockContentsArgs = {
        limit: undefined,
        offset: undefined,
        order: undefined,
        filters: [],
        recursive: true
    };

    beforeEach(() => {
        linkService = new LinkService(axios, []);
        groupService = new GroupsService(axios, []);
    });

    it("marks resource as favorite", async () => {
        linkService.create = cy.stub().returns(Promise.resolve({ uuid: "newUuid" })).as("create");
        const favoriteService = new FavoriteService(linkService, groupService);
        const newFavorite = await favoriteService.create({ userUuid: "userUuid", resource: { uuid: "resourceUuid", name: "resource" } });

        cy.get("@create").should("be.calledWith", {
            ownerUuid: "userUuid",
            tailUuid: "userUuid",
            headUuid: "resourceUuid",
            linkClass: LinkClass.STAR,
            name: "resource"
        });
        expect(newFavorite.uuid).to.equal("newUuid");

    });

    it("unmarks resource as favorite", async () => {
        const list = cy.stub().returns(Promise.resolve({ items: [{ uuid: "linkUuid" }] })).as("list");
        const filters = new FilterBuilder()
            .addEqual('owner_uuid', "userUuid")
            .addEqual('head_uuid', "resourceUuid")
            .addEqual('link_class', LinkClass.STAR);
        linkService.list = list;
        linkService.delete = cy.stub().returns(Promise.resolve({ uuid: "linkUuid" })).as("delete");
        const favoriteService = new FavoriteService(linkService, groupService);

        const newFavorite = await favoriteService.delete({ userUuid: "userUuid", resourceUuid: "resourceUuid" });

        cy.get("@list").should("be.calledWith", { filters: filters.getFilters() });
        cy.get("@delete").should("be.calledWith", "linkUuid");
        expect(newFavorite[0].uuid).to.equal("linkUuid");
    });

    it("lists favorite resources", async () => {
        const list = cy.stub().returns(Promise.resolve({ items: [{ headUuid: "headUuid" }] })).as("list");
        const listFilters = new FilterBuilder()
            .addEqual('owner_uuid', "userUuid")
            .addEqual('link_class', LinkClass.STAR);
        const contents = cy.stub().returns(Promise.resolve({ items: [{ uuid: "resourceUuid" }] })).as("contents");
        const contentFilters = new FilterBuilder().addIn('uuid', ["headUuid"]);
        linkService.list = list;
        groupService.contents = contents;
        const favoriteService = new FavoriteService(linkService, groupService);

        const favorites = await favoriteService.list("userUuid");

        cy.get("@list").should("be.calledWith", { ...mockListArgs, filters: listFilters.getFilters() });
        cy.get("@contents").should("be.calledWith", "userUuid", { ...mockContentsArgs,  filters: contentFilters.getFilters() });
        expect(isEqual(favorites, { items: [{ uuid: "resourceUuid" }] })).to.equal(true);
    });

    it("checks if resources are present in favorites", async () => {
        const list = cy.stub().returns(Promise.resolve({ items: [{ headUuid: "foo" }] })).as("list");
        const listFilters = new FilterBuilder()
            .addIn("head_uuid", ["foo", "oof"])
            .addEqual("owner_uuid", "userUuid")
            .addEqual("link_class", LinkClass.STAR);
        linkService.list = list;
        const favoriteService = new FavoriteService(linkService, groupService);

        const favorites = await favoriteService.checkPresenceInFavorites("userUuid", ["foo", "oof"]);

        cy.get("@list").should("be.calledWith", { filters: listFilters.getFilters() });
        expect(isEqual(favorites, { foo: true, oof: false })).to.be.true;
    });

});
