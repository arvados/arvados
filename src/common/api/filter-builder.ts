// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as _ from "lodash";

export class FilterBuilder {
    static create(resourcePrefix = "") {
        return new FilterBuilder(resourcePrefix);
    }

    constructor(
        private resourcePrefix = "",
        private filters = "") { }

    public addEqual(field: string, value?: string) {
        return this.addCondition(field, "=", value);
    }

    public addLike(field: string, value?: string) {
        return this.addCondition(field, "like", value, "%", "%");
    }

    public addILike(field: string, value?: string) {
        return this.addCondition(field, "ilike", value, "%", "%");
    }

    public addIsA(field: string, value?: string | string[]) {
        return this.addCondition(field, "is_a", value);
    }

    public addIn(field: string, value?: string | string[]) {
        return this.addCondition(field, "in", value);
    }

    public concat(filterBuilder: FilterBuilder) {
        return new FilterBuilder(this.resourcePrefix, this.filters + (this.filters && filterBuilder.filters ? "," : "") + filterBuilder.getFilters());
    }

    public getFilters() {
        return this.filters;
    }

    public serialize() {
        return "[" + this.filters + "]";
    }

    private addCondition(field: string, cond: string, value?: string | string[], prefix: string = "", postfix: string = "") {
        if (value) {
            value = typeof value === "string"
                ? `"${prefix}${value}${postfix}"`
                : `["${value.join(`","`)}"]`;

            const resourcePrefix = this.resourcePrefix
                ? _.snakeCase(this.resourcePrefix) + "."
                : "";

            this.filters += `${this.filters ? "," : ""}["${resourcePrefix}${_.snakeCase(field.toString())}","${cond}",${value}]`;
        }
        return this;
    }
}
