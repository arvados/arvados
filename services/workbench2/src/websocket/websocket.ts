// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { RootStore } from "store/store";
import { AuthService } from "services/auth-service/auth-service";
import { Config } from "common/config";
import { WebSocketService } from "./websocket-service";
import { ResourceEventMessage } from "./resource-event-message";
import { ResourceKind } from "models/resource";
import { loadProcess } from "store/process-panel/process-panel-actions";
import { getProcess, getSubprocesses } from "store/processes/process";
import { LogEventType } from "models/log";
import { subprocessPanelActions } from "store/subprocess-panel/subprocess-panel-actions";
import { projectPanelDataActions } from "store/project-panel/project-panel-action-bind";
import { getProjectPanelCurrentUuid } from "store/project-panel/project-panel-action";
import { allProcessesPanelActions } from "store/all-processes-panel/all-processes-panel-action";
import { loadCollection } from "store/workbench/workbench-actions";
import { matchAllProcessesRoute, matchProjectRoute, matchProcessRoute } from "routes/routes";

export const initWebSocket = (config: Config, authService: AuthService, store: RootStore) => {
    if (config.websocketUrl) {
        const webSocketService = new WebSocketService(config.websocketUrl, authService);
        webSocketService.setMessageListener(messageListener(store));
        webSocketService.connect();
    } else {
        console.warn("WARNING: Websocket ExternalURL is not set on the cluster config.");
    }
};

const messageListener = (store: RootStore) => (message: ResourceEventMessage) => {
    if (message.eventType === LogEventType.CREATE || message.eventType === LogEventType.UPDATE) {
        const state = store.getState();
        const location = state.router.location ? state.router.location.pathname : "";
        switch (message.objectKind) {
            case ResourceKind.COLLECTION:
                const currentCollection = state.collectionPanel.item;
                if (currentCollection && currentCollection.uuid === message.objectUuid) {
                    store.dispatch(loadCollection(message.objectUuid));
                }
                return;
            case ResourceKind.CONTAINER_REQUEST:
                if (matchProcessRoute(location)) {
                    if (state.processPanel.containerRequestUuid === message.objectUuid) {
                        store.dispatch(loadProcess(message.objectUuid));
                    }
                    const proc = getProcess(state.processPanel.containerRequestUuid)(state.resources);
                    if (proc && proc.container && proc.container.uuid === message.properties["new_attributes"]["requesting_container_uuid"]) {
                        store.dispatch(subprocessPanelActions.REQUEST_ITEMS(false, true));
                        return;
                    }
                }
            // fall through, this will happen for container requests as well.
            case ResourceKind.CONTAINER:
                if (matchProcessRoute(location)) {
                    // refresh only if this is a subprocess of the currently displayed process.
                    const subproc = getSubprocesses(state.processPanel.containerRequestUuid)(state.resources);
                    for (const sb of subproc) {
                        if (sb.containerRequest.uuid === message.objectUuid || (sb.container && sb.container.uuid === message.objectUuid)) {
                            store.dispatch(subprocessPanelActions.REQUEST_ITEMS(false, true));
                            break;
                        }
                    }
                }
                if (matchAllProcessesRoute(location)) {
                    store.dispatch(allProcessesPanelActions.REQUEST_ITEMS(false, true));
                }
                if (matchProjectRoute(location) && message.objectOwnerUuid === getProjectPanelCurrentUuid(state)) {
                    store.dispatch(projectPanelDataActions.REQUEST_ITEMS(false, true));
                }
                return;
            default:
                return;
        }
    }
};
