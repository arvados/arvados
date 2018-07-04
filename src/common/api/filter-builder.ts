// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as _ from "lodash";
import { Resource } from "./common-resource-service";

export default class FilterBuilder<T extends Resource = Resource> {
    private filters = "";

    static create<T extends Resource = Resource>() {
        return new FilterBuilder<T>();
    }

    private addCondition(field: keyof T, cond: string, value?: string | string[], prefix: string = "", postfix: string = "") {
        if (value) {
            value = typeof value === "string"
                ? `"${prefix}${value}${postfix}"`
                : `["${value.join(`","`)}"]`;

            this.filters += `["${_.snakeCase(field.toString())}","${cond}",${value}]`;
        }
        return this;
    }

    public addEqual(field: keyof T, value?: string) {
        return this.addCondition(field, "=", value);
    }

    public addLike(field: keyof T, value?: string) {
        return this.addCondition(field, "like", value, "", "%");
    }

    public addILike(field: keyof T, value?: string) {
        return this.addCondition(field, "ilike", value, "", "%");
    }

    public addIsA(field: keyof T, value?: string | string[]) {
        return this.addCondition(field, "is_a", value);
    }

    public get() {
        return "[" + this.filters + "]";
    }
}
