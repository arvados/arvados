// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { LinkService } from "../link-service/link-service";
import { GroupsService } from "../groups-service/groups-service";
import { FavoriteService } from "./favorite-service";
import { LinkClass } from "models/link";
import { mockResourceService } from "services/common-service/common-resource-service.test";
import { FilterBuilder } from "services/api/filter-builder";

describe("FavoriteService", () => {

    let linkService: LinkService;
    let groupService: GroupsService;

    beforeEach(() => {
        linkService = mockResourceService(LinkService);
        groupService = mockResourceService(GroupsService);
    });

    it("marks resource as favorite", async () => {
        linkService.create = jest.fn().mockReturnValue(Promise.resolve({ uuid: "newUuid" }));
        const favoriteService = new FavoriteService(linkService, groupService);

        const newFavorite = await favoriteService.create({ userUuid: "userUuid", resource: { uuid: "resourceUuid", name: "resource" } });

        expect(linkService.create).toHaveBeenCalledWith({
            ownerUuid: "userUuid",
            tailUuid: "userUuid",
            headUuid: "resourceUuid",
            linkClass: LinkClass.STAR,
            name: "resource"
        });
        expect(newFavorite.uuid).toEqual("newUuid");

    });

    it("unmarks resource as favorite", async () => {
        const list = jest.fn().mockReturnValue(Promise.resolve({ items: [{ uuid: "linkUuid" }] }));
        const filters = new FilterBuilder()
            .addEqual('owner_uuid', "userUuid")
            .addEqual('head_uuid', "resourceUuid")
            .addEqual('link_class', LinkClass.STAR);
        linkService.list = list;
        linkService.delete = jest.fn().mockReturnValue(Promise.resolve({ uuid: "linkUuid" }));
        const favoriteService = new FavoriteService(linkService, groupService);

        const newFavorite = await favoriteService.delete({ userUuid: "userUuid", resourceUuid: "resourceUuid" });

        expect(list.mock.calls[0][0].filters).toEqual(filters.getFilters());
        expect(linkService.delete).toHaveBeenCalledWith("linkUuid");
        expect(newFavorite[0].uuid).toEqual("linkUuid");
    });

    it("lists favorite resources", async () => {
        const list = jest.fn().mockReturnValue(Promise.resolve({ items: [{ headUuid: "headUuid" }] }));
        const listFilters = new FilterBuilder()
            .addEqual('owner_uuid', "userUuid")
            .addEqual('link_class', LinkClass.STAR);
        const contents = jest.fn().mockReturnValue(Promise.resolve({ items: [{ uuid: "resourceUuid" }] }));
        const contentFilters = new FilterBuilder().addIn('uuid', ["headUuid"]);
        linkService.list = list;
        groupService.contents = contents;
        const favoriteService = new FavoriteService(linkService, groupService);

        const favorites = await favoriteService.list("userUuid");

        expect(list.mock.calls[0][0].filters).toEqual(listFilters.getFilters());
        expect(contents.mock.calls[0][0]).toEqual("userUuid");
        expect(contents.mock.calls[0][1].filters).toEqual(contentFilters.getFilters());
        expect(favorites).toEqual({ items: [{ uuid: "resourceUuid" }] });
    });

    it("checks if resources are present in favorites", async () => {
        const list = jest.fn().mockReturnValue(Promise.resolve({ items: [{ headUuid: "foo" }] }));
        const listFilters = new FilterBuilder()
            .addIn("head_uuid", ["foo", "oof"])
            .addEqual("owner_uuid", "userUuid")
            .addEqual("link_class", LinkClass.STAR);
        linkService.list = list;
        const favoriteService = new FavoriteService(linkService, groupService);

        const favorites = await favoriteService.checkPresenceInFavorites("userUuid", ["foo", "oof"]);

        expect(list.mock.calls[0][0].filters).toEqual(listFilters.getFilters());
        expect(favorites).toEqual({ foo: true, oof: false });
    });

});
