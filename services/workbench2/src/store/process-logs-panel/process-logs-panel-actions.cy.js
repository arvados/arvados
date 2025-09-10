// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import {
    pollProcessLogs,
    processLogsPanelActions,
} from "./process-logs-panel-actions";
import { ContainerRequestState } from "models/container-request";
import { ContainerState } from "models/container";

describe("pollProcessLogs", () => {
    const processUuid = "xxxxx-xvhdp-000000000000000";
    const containerUuid = "xxxxx-dz642-000000000000000";
    const dummyProcess = {
        containerRequest: {
            uuid: processUuid,
            logUuid: "xxxxx-4zz18-000000000000000",
            state: ContainerRequestState.COMMITTED,
            containerUuid: containerUuid,
        },
        container: {
            uuid: containerUuid,
            state: ContainerState.QUEUED,
        },
    };

    let dispatch;
    let getState;
    let logServiceStub;

    // Fake Container Request Service that returns dummy scheduling statuses
    const createCrServiceStub = (schedulingStatus) => ({
        containerStatus: cy
            .stub()
            .resolves({ schedulingStatus }),
    });


    beforeEach(() => {
        dispatch = cy.stub();

        getState = () => ({
            resources: {
                [processUuid]: dummyProcess.containerRequest,
                [containerUuid]: dummyProcess.container,
            },
            processLogsPanel: { logs: {}, filters: [] },
        });

        // Log service stubs
        logServiceStub = {
            listLogFiles: cy.stub().resolves([]),
            getLogFileContents: cy.stub().resolves([]),
        };

    });

    it("Dispatches ADD_PROCESS_LOGS_PANEL_ITEM for valid SCHEDULING logs", async () => {
        const schedulingStatus = "some status";

        const thunk = pollProcessLogs(processUuid);

        // Execute the thunk
        const res = await thunk(dispatch, getState, {
            logService: logServiceStub,
            containerRequestService: createCrServiceStub(schedulingStatus),
        });

        // We expect add process logs panel to be dispatched with the sceduling log in relevant sections
        expect(dispatch).to.have.been.calledWithMatch(processLogsPanelActions.ADD_PROCESS_LOGS_PANEL_ITEM({
            "Main logs": {lastByte: undefined, contents: [ Cypress.sinon.match.string ]},
            "All logs": {lastByte: undefined, contents: [ Cypress.sinon.match.string ]},
            "scheduling": {lastByte: undefined, contents: [ Cypress.sinon.match.string ]},
        }));

        // Expect no calls
        expect(dispatch).to.have.callCount(1);
    });

    it("Does not dispatch ADD_PROCESS_LOGS_PANEL_ITEM for whitespace-only SCHEDULING logs", async () => {
        // Tests tab, CR, LF, nbsp
        const schedulingStatus = "\t \r \n \xa0";

        const thunk = pollProcessLogs(processUuid);

        // Execute the thunk
        await thunk(dispatch, getState, {
            logService: logServiceStub,
            containerRequestService: createCrServiceStub(schedulingStatus),
        });

        // Since the scheduling status was only whitespace, no logs should be added
        // pollProcessLogs only dispatches ADD_PROCESS_LOGS_PANEL_ITEM when logFragments.length > 0
        expect(dispatch).to.not.have.been.called;

        // Expect no calls
        expect(dispatch).to.have.callCount(0);
    });
});
