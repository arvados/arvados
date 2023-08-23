// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as _ from "./resource-properties";
import { omit } from "lodash";

describe("Resource properties lib", () => {

    let properties: any;

    beforeEach(() => {
        properties = {
            animal: 'dog',
            color: ['brown', 'black'],
            name: ['Toby']
        }
    })

    it("should convert a single string value into a list when adding values", () => {
        expect(
            _.addProperty(properties, 'animal', 'cat')
        ).toEqual({
            ...properties, animal: ['dog', 'cat']
        });
    });

    it("should convert a 2 value list into a string when removing values", () => {
        expect(
            _.deleteProperty(properties, 'color', 'brown')
        ).toEqual({
            ...properties, color: 'black'
        });
    });

    it("shouldn't add duplicated key:value items", () => {
        expect(
            _.addProperty(properties, 'animal', 'dog')
        ).toEqual(properties);
    });

    it("should remove the key when deleting from a one value list", () => {
        expect(
            _.deleteProperty(properties, 'name', 'Toby')
        ).toEqual(omit(properties, 'name'));
    });

    it("should return the same when deleting non-existant value", () => {
        expect(
            _.deleteProperty(properties, 'animal', 'dolphin')
        ).toEqual(properties);
    });

    it("should return the same when deleting non-existant key", () => {
        expect(
            _.deleteProperty(properties, 'doesntexist', 'something')
        ).toEqual(properties);
    });
});