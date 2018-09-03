import { LogEventType } from '../models/log';
// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export interface ResourceEventMessage<Properties = {}> {
    eventAt: string;
    eventType: LogEventType;
    id: string;
    msgID: string;
    objectKind: string;
    objectOwnerUuid: string;
    objectUuid: string;
    properties: Properties;
    uuid: string;
}
