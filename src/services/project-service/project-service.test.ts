// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import axios from "axios";
import { ProjectService } from "./project-service";
import { FilterBuilder } from "~/services/api/filter-builder";
import { ApiActions } from "~/services/api/api-actions";

describe("CommonResourceService", () => {
    const axiosInstance = axios.create();
    const actions: ApiActions = {
        progressFn: (id: string, working: boolean) => {},
        errorFn: (id: string, message: string) => {}
    };

    it(`#create has groupClass set to "project"`, async () => {
        axiosInstance.post = jest.fn(() => Promise.resolve({ data: {} }));
        const projectService = new ProjectService(axiosInstance, actions);
        const resource = await projectService.create({ name: "nameValue" });
        expect(axiosInstance.post).toHaveBeenCalledWith("/groups", {
            group: {
                name: "nameValue",
                group_class: "project"
            }
        });
    });

    it("#list has groupClass filter set by default", async () => {
        axiosInstance.get = jest.fn(() => Promise.resolve({ data: {} }));
        const projectService = new ProjectService(axiosInstance, actions);
        const resource = await projectService.list();
        expect(axiosInstance.get).toHaveBeenCalledWith("/groups", {
            params: {
                filters: "[" + new FilterBuilder()
                    .addIn("group_class", ["project", "filter"])
                    .getFilters() + "]",
                order: undefined
            }
        });
    });
});
