// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { snakeCase } from "lodash";
import { Resource } from "src/models/resource";

export enum OrderDirection { ASC, DESC }

export class OrderBuilder<T extends Resource = Resource> {

    constructor(private order: string[] = []) {}

    addOrder(direction: OrderDirection, attribute: keyof T, prefix?: string) {
        this.order.push(`${prefix ? prefix + "." : ""}${snakeCase(attribute.toString())} ${direction === OrderDirection.ASC ? "asc" : "desc"}`);
        return this;
    }

    addAsc(attribute: keyof T, prefix?: string) {
        return this.addOrder(OrderDirection.ASC, attribute, prefix);
    }

    addDesc(attribute: keyof T, prefix?: string) {
        return this.addOrder(OrderDirection.DESC, attribute, prefix);
    }

    getOrder() {
        return this.order.join(",");
    }
}
