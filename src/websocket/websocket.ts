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
import { loadContainers } from '~/store/processes/processes-actions';
import { LogEventType } from '~/models/log';
import { addProcessLogsPanelItem } from '../store/process-logs-panel/process-logs-panel-actions';
import { FilterBuilder } from "~/services/api/filter-builder";
import { snackbarActions, SnackbarKind } from "~/store/snackbar/snackbar-actions";

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
                return store.dispatch(loadProcess(message.objectUuid));
            case ResourceKind.CONTAINER:
                return store.dispatch(loadContainers(
                    new FilterBuilder().addIn('uuid', [message.objectUuid]).getFilters()
                ));
            default:
                return;
        }
    } else {
        return store.dispatch(addProcessLogsPanelItem(message as ResourceEventMessage<{text: string}>));
    }
};
