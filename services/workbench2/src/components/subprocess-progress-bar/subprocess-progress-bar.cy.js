// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { createServices } from "services/services";
import { configureStore } from "store/store";
import { createBrowserHistory } from "history";
import { mockConfig } from 'common/config';
import Axios from "axios";
import { ContainerState } from "models/container";
import { SubprocessProgressBar } from "./subprocess-progress-bar";
import { Provider } from "react-redux";
import { FilterBuilder } from 'services/api/filter-builder';
import { ProcessStatusFilter, buildProcessStatusFilters } from 'store/resource-type-filters/resource-type-filters';

describe("<SubprocessProgressBar />", () => {
    const axiosInst = Axios.create({ headers: {} });

    let store;
    let services;
    const config = {};
    const actions = {
        progressFn: (id, working) => { },
        errorFn: (id, message) => { }
    };
    let statusResponse = {
        [ProcessStatusFilter.COMPLETED]: 0,
        [ProcessStatusFilter.RUNNING]: 0,
        [ProcessStatusFilter.FAILED]: 0,
        [ProcessStatusFilter.QUEUED]: 0,
    };

    const createMockListFunc = (uuid) => (async (args) => {
        const baseFilter = new FilterBuilder().addEqual('requesting_container_uuid', uuid).getFilters();

        const filterResponses = Object.keys(statusResponse)
            .map(status => ({filters: buildProcessStatusFilters(new FilterBuilder(baseFilter), status).getFilters(), value: statusResponse[status]}));

        const matchedFilter = filterResponses.find(response => response.filters === args.filters);
        if (matchedFilter) {
            return { itemsAvailable: matchedFilter.value };
        } else {
            return { itemsAvailable: 0 };
        }
    });

    beforeEach(() => {
        services = createServices(mockConfig({}), actions, axiosInst);
        store = configureStore(createBrowserHistory(), services, config);
    });

    it("requests subprocess progress stats for stopped processes and displays progress", async () => {
        // when
        const process = {
            container: {
                state: ContainerState.COMPLETE,
            },
            containerRequest: {
                containerUuid: 'zzzzz-dz642-000000000000000',
            },
        };

        statusResponse = {
            [ProcessStatusFilter.COMPLETED]: 100,
            [ProcessStatusFilter.RUNNING]: 200,

            // Combined into failed segment
            [ProcessStatusFilter.FAILED]: 200,
            [ProcessStatusFilter.CANCELLED]: 100,

            // Combined into queued segment
            [ProcessStatusFilter.QUEUED]: 300,
            [ProcessStatusFilter.ONHOLD]: 100,
        };

        services.containerRequestService.list = createMockListFunc(process.containerRequest.containerUuid);

        cy.mount(
            <Provider store={store}>
                <SubprocessProgressBar process={process} />
            </Provider>);

        // expects 6 subprocess status list requests
        const expectedFilters = [
            ProcessStatusFilter.COMPLETED,
            ProcessStatusFilter.RUNNING,
            ProcessStatusFilter.FAILED,
            ProcessStatusFilter.CANCELLED,
            ProcessStatusFilter.QUEUED,
            ProcessStatusFilter.ONHOLD,
        ].map((state) =>
            buildProcessStatusFilters(
                new FilterBuilder().addEqual(
                    "requesting_container_uuid",
                    process.containerRequest.containerUuid
                ),
                state
            ).getFilters()
        );

        cy.spy(services.containerRequestService, 'list').as('list');
        
        expectedFilters.forEach((filter) => {
            cy.get('@list').should('have.been.calledWith', {limit: 0, offset: 0, filters: filter});
        });

        // Verify progress bar with correct degment widths
        ['10%', '20%', '30%', '40%'].forEach((value, i) => {
            cy.get('div[class=progress]').eq(i).should('have.attr', 'style').and('include', `width: ${value};`);

        });
    });

    it("dislays correct progress bar widths with different values", async () => {
        const process = {
            container: {
                state: ContainerState.COMPLETE,
            },
            containerRequest: {
                containerUuid: 'zzzzz-dz642-000000000000001',
            },
        };

        statusResponse = {
            [ProcessStatusFilter.COMPLETED]: 50,
            [ProcessStatusFilter.RUNNING]: 55,

            [ProcessStatusFilter.FAILED]: 30,
            [ProcessStatusFilter.CANCELLED]: 30,

            [ProcessStatusFilter.QUEUED]: 235,
            [ProcessStatusFilter.ONHOLD]: 100,
        };

        services.containerRequestService.list = createMockListFunc(process.containerRequest.containerUuid);

        cy.mount(
            <Provider store={store}>
                <SubprocessProgressBar process={process} />
            </Provider>);

        // Verify progress bar with correct degment widths
        ['10%', '11%', '12%', '67%'].forEach((value, i) => {
            cy.get('div[class=progress]').eq(i).should('have.attr', 'style').and('include', `width: ${value};`);
        });
    });

});
