// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { OrderBuilder } from "./order-builder";

describe("OrderBuilder", () => {
    it("should build correct order query", () => {
        const order = new OrderBuilder()
            .addAsc("kind")
            .addDesc("createdAt")
            .getOrder();
        expect(order).toEqual("kind asc,created_at desc");
    });
});
