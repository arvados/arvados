// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0


export default class OrderBuilder {
    private order: string[] = [];

    addAsc(attribute: string) {
        this.order.push(`${attribute} asc`);
        return this;
    }

    addDesc(attribute: string) {
        this.order.push(`${attribute} desc`);
        return this;
    }

    get() {
        return `["${this.order.join(`","`)}"]`;
    }
}
