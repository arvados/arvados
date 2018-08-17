// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as _ from "lodash";

export function joinFilters(filters0?: string, filters1?: string) {
    return [filters0, filters1].filter(s => s).join(",");
}

export class FilterBuilder {
    constructor(private filters = "") { }

    public addEqual(field: string, value?: string, resourcePrefix?: string) {
        return this.addCondition(field, "=", value, "", "", resourcePrefix );
    }

    public addLike(field: string, value?: string, resourcePrefix?: string) {
        return this.addCondition(field, "like", value, "%", "%", resourcePrefix);
    }

    public addILike(field: string, value?: string, resourcePrefix?: string) {
        return this.addCondition(field, "ilike", value, "%", "%", resourcePrefix);
    }

    public addIsA(field: string, value?: string | string[], resourcePrefix?: string) {
        return this.addCondition(field, "is_a", value, "", "", resourcePrefix);
    }

    public addIn(field: string, value?: string | string[], resourcePrefix?: string) {
        return this.addCondition(field, "in", value, "", "", resourcePrefix);
    }

    public getFilters() {
        return this.filters;
    }

    private addCondition(field: string, cond: string, value?: string | string[], prefix: string = "", postfix: string = "", resourcePrefix?: string) {
        if (value) {
            value = typeof value === "string"
                ? `"${prefix}${value}${postfix}"`
                : `["${value.join(`","`)}"]`;

            const resPrefix = resourcePrefix
                ? _.snakeCase(resourcePrefix) + "."
                : "";

            this.filters += `${this.filters ? "," : ""}["${resPrefix}${_.snakeCase(field)}","${cond}",${value}]`;
        }
        return this;
    }
}
