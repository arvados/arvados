// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { LinkService } from "../link-service/link-service";
import { GroupsService, GroupContentsResource } from "../groups-service/groups-service";
import { FavoriteService } from "./favorite-service";
import { LinkClass, LinkResource } from "../../models/link";
import { mockResourceService } from "../../common/api/common-resource-service.test";
import { FilterBuilder } from "../../common/api/filter-builder";

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

        const newFavorite = await favoriteService.create({ userUuid: "userUuid", resourceUuid: "resourceUuid" });

        expect(linkService.create).toHaveBeenCalledWith({
            ownerUuid: "userUuid",
            tailUuid: "userUuid",
            headUuid: "resourceUuid",
            linkClass: LinkClass.STAR,
            name: "resourceUuid"
        });
        expect(newFavorite.uuid).toEqual("newUuid");

    });

    it("unmarks resource as favorite", async () => {
        const list = jest.fn().mockReturnValue(Promise.resolve({ items: [{ uuid: "linkUuid" }] }));
        const filters = FilterBuilder
            .create<LinkResource>()
            .addEqual('tailUuid', "userUuid")
            .addEqual('headUuid', "resourceUuid")
            .addEqual('linkClass', LinkClass.STAR);
        linkService.list = list;
        linkService.delete = jest.fn().mockReturnValue(Promise.resolve({ uuid: "linkUuid" }));
        const favoriteService = new FavoriteService(linkService, groupService);

        const newFavorite = await favoriteService.delete({ userUuid: "userUuid", resourceUuid: "resourceUuid" });

        expect(list.mock.calls[0][0].filters.getFilters()).toEqual(filters.getFilters());
        expect(linkService.delete).toHaveBeenCalledWith("linkUuid");
        expect(newFavorite[0].uuid).toEqual("linkUuid");
    });

    it("lists favorite resources", async () => {
        const list = jest.fn().mockReturnValue(Promise.resolve({ items: [{ headUuid: "headUuid" }] }));
        const listFilters = FilterBuilder
            .create<LinkResource>()
            .addEqual('tailUuid', "userUuid")
            .addEqual('linkClass', LinkClass.STAR);
        const contents = jest.fn().mockReturnValue(Promise.resolve({ items: [{ uuid: "resourceUuid" }] }));
        const contentFilters = FilterBuilder.create<GroupContentsResource>().addIn('uuid', ["headUuid"]);
        linkService.list = list;
        groupService.contents = contents;
        const favoriteService = new FavoriteService(linkService, groupService);

        const favorites = await favoriteService.list("userUuid");

        expect(list.mock.calls[0][0].filters.getFilters()).toEqual(listFilters.getFilters());
        expect(contents.mock.calls[0][0]).toEqual("userUuid");
        expect(contents.mock.calls[0][1].filters.getFilters()).toEqual(contentFilters.getFilters());
        expect(favorites).toEqual({ items: [{ uuid: "resourceUuid" }] });
    });

});
