// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { isEqual } from "lodash";
import { createServices } from "services/services";
import { configureStore } from "../store";
import { createBrowserHistory } from "history";
import { mockConfig } from 'common/config';
import Axios from "axios";
import MockAdapter from "axios-mock-adapter";
import { openRunProcess } from './workflow-panel-actions';
import { runProcessPanelActions } from 'store/run-process-panel/run-process-panel-actions';
import { initialize } from 'redux-form';
import { RUN_PROCESS_INPUTS_FORM } from 'views/run-process-panel/run-process-inputs-form';
import { ResourceKind } from 'models/resource';

describe('workflow-panel-actions', () => {
    const axiosInst = Axios.create({ headers: {} });
    const axiosMock = new MockAdapter(axiosInst);

    let store;
    let services;
    const config = {};
    const actions = {
        progressFn: (id, working) => { },
        errorFn: (id, message) => { }
    };
    let importMocks;

    beforeEach(() => {
        axiosMock.reset();
        services = createServices(mockConfig({}), actions, axiosInst);
        store = configureStore(createBrowserHistory(), services, config);
        localStorage.clear();
        importMocks = [];
    });

    afterEach(() => {
        importMocks.map(m => m.restore());
    });

    it('opens the run process panel', async () => {
        const wflist = [{
            uuid: "zzzzz-7fd4e-0123456789abcde",
            name: "foo",
            description: "",
            definition: "$graph: []",
            kind: ResourceKind.WORKFLOW,
            ownerUuid: "",
            createdAt: "",
            modifiedByClientUuid: "",
            modifiedByUserUuid: "",
            modifiedAt: "",
            href: "",
            etag: ""
        }];
        axiosMock
            .onGet("/workflows")
            .reply(200, {
                items: wflist
            }).onGet("/links")
            .reply(200, {
                items: []
            });

        const dispatchMock = cy.spy();
        const dispatchWrapper = (action ) => {
            dispatchMock(action);
            return store.dispatch(action);
        };

        await openRunProcess("zzzzz-7fd4e-0123456789abcde", "zzzzz-tpzed-0123456789abcde", "testing", { inputparm: "value" })(dispatchWrapper, store.getState, services);
        expect(dispatchMock).to.be.calledWith(runProcessPanelActions.SET_WORKFLOWS(wflist));
        expect(dispatchMock).to.be.calledWith(runProcessPanelActions.SET_SELECTED_WORKFLOW(wflist[0]));
        expect(arrayDeeplyIncludesObject(dispatchMock.args, initialize(RUN_PROCESS_INPUTS_FORM, { inputparm: "value" }))).to.be.true;
        expect(dispatchMock).to.be.calledWith(initialize(RUN_PROCESS_INPUTS_FORM, { inputparm: "value" }));
    });
});

const arrayDeeplyIncludesObject = (array, object) => {
    return array.some((item) => {
        if (isEqual(item, object)) {
            return true;
        }
        if (typeof item === 'object') {
            return arrayDeeplyIncludesObject(Object.values(item), object);
        }
        return false;
    });
};

