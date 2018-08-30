// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export enum ResourceEventMessageType {
    CREATE = 'create',
    UPDATE = 'update',
    HOTSTAT = 'hotstat',
    CRUNCH_RUN = 'crunch-run',
    NODE_INFO = 'node-info',
}

export interface ResourceEventMessage {
    eventAt: string;
    eventType: ResourceEventMessageType;
    id: string;
    msgID: string;
    objectKind: string;
    objectOwnerUuid: string;
    objectUuid: string;
    properties: {};
    uuid: string;
}
