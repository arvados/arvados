// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import axios from "axios";
import { ProjectService } from "./project-service";
import { FilterBuilder } from "services/api/filter-builder";

describe("CommonResourceService", () => {
    const axiosInstance = axios.create();
    const actions = {
        progressFn: (id, working) => {},
        errorFn: (id, message) => {}
    };

    it(`#create has groupClass set to "project"`, async () => {
        axiosInstance.post = cy.stub().returns(Promise.resolve({ data: {} })).as("post");
        const projectService = new ProjectService(axiosInstance, actions);

        await projectService.create({ name: "nameValue" });

        cy.get("@post").should("be.calledWith", "/groups", {
            group: {
                name: "nameValue",
                group_class: "project"
            }
        });
    });

    it("#list has groupClass filter set by default", async () => {
        axiosInstance.get = cy.stub().returns(Promise.resolve({ data: {} })).as("get");
        const projectService = new ProjectService(axiosInstance, actions);

        await projectService.list();

        cy.get("@get").should("be.calledWith", "/groups", {
            params: {
                filters: "[" + new FilterBuilder()
                    .addEqual("group_class", "project")
                    .getFilters() + "]",
                order: undefined
            }
        });
    });
});
