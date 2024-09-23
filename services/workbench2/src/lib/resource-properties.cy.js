// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { addProperty, deleteProperty } from "./resource-properties";
import { omit, isEqual } from "lodash";

describe("Resource properties lib", () => {

    let properties;

    beforeEach(() => {
        properties = {
            animal: 'dog',
            color: ['brown', 'black'],
            name: ['Toby']
        }
    })

    it("should convert a single string value into a list when adding values", () => {
        expect(isEqual(addProperty(properties, 'animal', 'cat'), {...properties, animal: ['dog', 'cat']})).to.equal(true);
    });

    it("should convert a 2 value list into a string when removing values", () => {
        expect(isEqual(deleteProperty(properties, 'color', 'brown'), {...properties, color: 'black'})).to.equal(true);
    });

    it("shouldn't add duplicated key:value items", () => {
        expect(isEqual(addProperty(properties, 'name', 'Toby'), properties)).to.equal(true);
    });

    it("should remove the key when deleting from a one value list", () => {
        expect(isEqual(deleteProperty(properties, 'name', 'Toby'), omit(properties, 'name'))).to.equal(true);
    });

    it("should return the same when deleting non-existant value", () => {
        expect(isEqual(deleteProperty(properties, 'animal', 'dolphin'), properties)).to.equal(true);
    });

    it("should return the same when deleting non-existant key", () => {
        expect(isEqual(deleteProperty(properties, 'doesntexist', 'something'), properties)).to.equal(true);
    });
});