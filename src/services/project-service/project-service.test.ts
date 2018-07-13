// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import axios from "axios";
import MockAdapter from "axios-mock-adapter/types";
import ProjectService from "./project-service";
import FilterBuilder from "../../common/api/filter-builder";
import { ProjectResource } from "../../models/project";

describe("CommonResourceService", () => {
    const axiosInstance = axios.create();

    it(`#create has groupClass set to "project"`, async () => {
        axiosInstance.post = jest.fn(() => Promise.resolve({ data: {} }));
        const projectService = new ProjectService(axiosInstance);
        const resource = await projectService.create({ name: "nameValue" });
        expect(axiosInstance.post).toHaveBeenCalledWith("/groups/", {
            name: "nameValue",
            group_class: "project"
        });
    });


    it("#list has groupClass filter set by default", async () => {
        axiosInstance.get = jest.fn(() => Promise.resolve({ data: {} }));
        const projectService = new ProjectService(axiosInstance);
        const resource = await projectService.list();
        expect(axiosInstance.get).toHaveBeenCalledWith("/groups/", {
            params: {
                filters: FilterBuilder
                    .create<ProjectResource>()
                    .addEqual("groupClass", "project")
                    .serialize()
            }
        });
    });
    
});
