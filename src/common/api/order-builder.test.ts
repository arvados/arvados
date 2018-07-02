// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import OrderBuilder from "./order-builder";

describe("OrderBuilder", () => {
    it("should build correct order query", () => {
        const orderBuilder = new OrderBuilder();
        const order = orderBuilder
            .addAsc("name")
            .addDesc("modified_at")
            .get();
        expect(order).toEqual(`["name asc","modified_at desc"]`);
    });
});
