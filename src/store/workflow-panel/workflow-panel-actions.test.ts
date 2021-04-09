// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { getNewExtraToken, initAuth } from "./auth-action";
import { API_TOKEN_KEY } from "~/services/auth-service/auth-service";

import 'jest-localstorage-mock';
import { ServiceRepository, createServices } from "~/services/services";
import { configureStore, RootStore } from "../store";
import { createBrowserHistory } from "history";
import { Config, mockConfig } from '~/common/config';
import { ApiActions } from "~/services/api/api-actions";
import { ACCOUNT_LINK_STATUS_KEY } from '~/services/link-account-service/link-account-service';
import Axios from "axios";
import MockAdapter from "axios-mock-adapter";
import { ImportMock } from 'ts-mock-imports';
import * as servicesModule from "~/services/services";
import { SessionStatus } from "~/models/session";
import { openRunProcess } from './workflow-panel-actions';
import { runProcessPanelActions } from '~/store/run-process-panel/run-process-panel-actions';
import { initialize } from 'redux-form';
import { RUN_PROCESS_BASIC_FORM } from '~/views/run-process-panel/run-process-basic-form';
import { RUN_PROCESS_INPUTS_FORM } from '~/views/run-process-panel/run-process-inputs-form';

describe('workflow-panel-actions', () => {
    const axiosInst = Axios.create({ headers: {} });
    const axiosMock = new MockAdapter(axiosInst);

    let store: RootStore;
    let services: ServiceRepository;
    const config: any = {};
    const actions: ApiActions = {
        progressFn: (id: string, working: boolean) => { },
        errorFn: (id: string, message: string) => { }
    };
    let importMocks: any[];

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
            definition: "$graph: []"
        }];
        axiosMock
            .onGet("/workflows")
            .reply(200, {
                items: wflist
            }).onGet("/links")
            .reply(200, {
                items: []
            });

        const dispatchMock = jest.fn();
        const dispatchWrapper = (action: any) => {
            dispatchMock(action);
            return store.dispatch(action);
        };

        await openRunProcess("zzzzz-7fd4e-0123456789abcde", "zzzzz-tpzed-0123456789abcde", "testing", { inputparm: "value" })(dispatchWrapper, store.getState, services);
        expect(dispatchMock).toHaveBeenCalledWith(runProcessPanelActions.SET_WORKFLOWS(wflist));
        expect(dispatchMock).toHaveBeenCalledWith(runProcessPanelActions.SET_SELECTED_WORKFLOW(wflist[0]));
        expect(dispatchMock).toHaveBeenCalledWith(runProcessPanelActions.SET_PROCESS_OWNER_UUID("zzzzz-tpzed-0123456789abcde"));
        expect(dispatchMock).toHaveBeenCalledWith(initialize(RUN_PROCESS_BASIC_FORM, { name: "testing" }));
        expect(dispatchMock).toHaveBeenCalledWith(initialize(RUN_PROCESS_INPUTS_FORM, { inputparm: "value" }));
    });
});
