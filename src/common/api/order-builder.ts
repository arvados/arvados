// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as _ from "lodash";
import { Resource } from "../../models/resource";

export class OrderBuilder<T extends Resource = Resource> {

    static create<T extends Resource = Resource>(prefix?: string){
        return new OrderBuilder<T>([], prefix);
    }

    private constructor(
        private order: string[] = [],
        private prefix = ""){}

    private addRule (direction: string, attribute: keyof T) {
        const prefix = this.prefix ? this.prefix + "." : "";
        const order = [...this.order, `${prefix}${_.snakeCase(attribute.toString())} ${direction}`];
        return new OrderBuilder<T>(order, prefix);
    }

    addAsc(attribute: keyof T) {
        return this.addRule("asc", attribute);
    }

    addDesc(attribute: keyof T) {
        return this.addRule("desc", attribute);
    }

    concat(orderBuilder: OrderBuilder){
        return new OrderBuilder<T>(
            this.order.concat(orderBuilder.getOrder()),
            this.prefix
        );
    }

    getOrder() {
        return this.order.slice();
    }
}
