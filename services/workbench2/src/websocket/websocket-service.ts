// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { AuthService } from 'services/auth-service/auth-service';
import { ResourceEventMessage } from './resource-event-message';
import { camelCase } from 'lodash';
import { CommonResourceService } from "services/common-service/common-resource-service";

type MessageListener = (message: ResourceEventMessage) => void;

export class WebSocketService {
    private static instance: WebSocketService;

    private ws: WebSocket;
    private messageListener: MessageListener;
    private url: string;
    private authService: AuthService

    /**
     * Empty constructor so that consumers checking for WS initialization need
     * not pass in configuration
     */
    private constructor() {}

    /**
     * Gets the singleton WebSocketService instance
     * @returns The singleton WebSocketService
     */
    public static getInstance() {
        if (this.instance) {
            return this.instance;
        }
        this.instance = new WebSocketService();
        return this.instance;
    }

    /**
     * Sets connection params, starts WS connection, and attaches handlers
     * @param url WS url
     * @param authService Auth service containing API token
     */
    public connect(url: string, authService: AuthService) {
        if (this.ws) {
            this.ws.close();
        }
        this.url = url;
        this.authService = authService;
        this.ws = new WebSocket(this.getUrl());
        this.ws.addEventListener('message', this.handleMessage);
        this.ws.addEventListener('open', this.handleOpen);
    }

    public setMessageListener = (listener: MessageListener) => {
        this.messageListener = listener;
    }

    /**
     * Returns true if the WS is in any active state, including "CLOSING"
     * Useful to prevent re-initialization before WS is closed
     * Only returns false if the WS is not initialized or fully closed
     * @returns whether the WebSocket is initialized or in transition state
     */
    isInitialized = (): boolean => {
        return !!this.ws && this.ws.readyState !== WebSocket.CLOSED;
    }

    /**
     * Returns true only if the WebSocket connection is active
     * Returns false in any other state, including connecting and closing
     * @returns whether the WebSocket is active
     */
    isActive = (): boolean => {
        return !!this.ws && this.ws.readyState === WebSocket.OPEN;
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
