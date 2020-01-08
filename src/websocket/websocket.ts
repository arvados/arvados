// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { RootStore } from '~/store/store';
import { AuthService } from '~/services/auth-service/auth-service';
import { Config } from '~/common/config';
import { WebSocketService } from './websocket-service';
import { ResourceEventMessage } from './resource-event-message';
import { ResourceKind } from '~/models/resource';
import { loadProcess } from '~/store/processes/processes-actions';
// import { loadContainers } from '~/store/processes/processes-actions';
import { LogEventType } from '~/models/log';
import { addProcessLogsPanelItem } from '../store/process-logs-panel/process-logs-panel-actions';
// import { FilterBuilder } from "~/services/api/filter-builder";
import { subprocessPanelActions } from "~/store/subprocess-panel/subprocess-panel-actions";
import { projectPanelActions } from "~/store/project-panel/project-panel-action";

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
        switch (message.objectKind) {
            case ResourceKind.CONTAINER_REQUEST:
                if (store.getState().processPanel.containerRequestUuid === message.objectUuid) {
                    store.dispatch(loadProcess(message.objectUuid));
                }
            case ResourceKind.CONTAINER:
                store.dispatch(subprocessPanelActions.REQUEST_ITEMS());
                store.dispatch(projectPanelActions.REQUEST_ITEMS());
                return;
            default:
                return;
        }
    } else {
        return store.dispatch(addProcessLogsPanelItem(message as ResourceEventMessage<{ text: string }>));
    }
};
