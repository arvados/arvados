// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as _ from "lodash";
import { Resource } from "./common-resource-service";

export default class OrderBuilder<T extends Resource = Resource> {

    static create<T extends Resource = Resource>(prefix?: string){
        return new OrderBuilder<T>([], prefix);
    }

    private constructor(
        private order: string[] = [], 
        private prefix = ""){}

    private getRule (direction: string, attribute: keyof T) {
        const prefix = this.prefix ? this.prefix + "." : "";
        return `${prefix}${_.snakeCase(attribute.toString())} ${direction}`;
    }

    addAsc(attribute: keyof T) {
        return new OrderBuilder<T>(
            [...this.order, this.getRule("asc", attribute)],
            this.prefix
        );
    }

    addDesc(attribute: keyof T) {
        return new OrderBuilder<T>(
            [...this.order, this.getRule("desc", attribute)],
            this.prefix
        );
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
