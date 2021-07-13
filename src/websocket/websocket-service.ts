// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { AuthService } from 'services/auth-service/auth-service';
import { ResourceEventMessage } from './resource-event-message';
import { camelCase } from 'lodash';
import { CommonResourceService } from "services/common-service/common-resource-service";

type MessageListener = (message: ResourceEventMessage) => void;

export class WebSocketService {
    private ws: WebSocket;
    private messageListener: MessageListener;

    constructor(private url: string, private authService: AuthService) { }

    connect() {
        if (this.ws) {
            this.ws.close();
        }
        this.ws = new WebSocket(this.getUrl());
        this.ws.addEventListener('message', this.handleMessage);
        this.ws.addEventListener('open', this.handleOpen);
    }

    setMessageListener = (listener: MessageListener) => {
        this.messageListener = listener;
    }

    private getUrl() {
        return `${this.url}?api_token=${this.authService.getApiToken()}`;
    }

    private handleMessage = (event: MessageEvent) => {
        if (this.messageListener) {
            const data = JSON.parse(event.data);
            const message = CommonResourceService.mapKeys(camelCase)(data);
            this.messageListener(message);
        }
    }

    private handleOpen = () => {
        this.ws.send('{"method":"subscribe"}');
    }

}
