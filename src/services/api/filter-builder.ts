// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as _ from "lodash";

export function joinFilters(filters0?: string, filters1?: string) {
    return [filters0, filters1].filter(s => s).join(",");
}

export class FilterBuilder {
    constructor(private filters = "") { }

    public addEqual(field: string, value?: string | boolean, resourcePrefix?: string) {
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

    public addNotIn(field: string, value?: string | string[], resourcePrefix?: string) {
        return this.addCondition(field, "not in", value, "", "", resourcePrefix);
    }

    public addGt(field: string, value?: string, resourcePrefix?: string) {
        return this.addCondition(field, ">", value, "", "", resourcePrefix);
    }

    public addGte(field: string, value?: string, resourcePrefix?: string) {
        return this.addCondition(field, ">=", value, "", "", resourcePrefix);
    }

    public addLt(field: string, value?: string, resourcePrefix?: string) {
        return this.addCondition(field, "<", value, "", "", resourcePrefix);
    }

    public addLte(field: string, value?: string, resourcePrefix?: string) {
        return this.addCondition(field, "<=", value, "", "", resourcePrefix);
    }

    public addExists(value?: string, resourcePrefix?: string) {
        return this.addCondition("properties", "exists", value, "", "", resourcePrefix);
    }

    public getFilters() {
        return this.filters;
    }

    private addCondition(field: string, cond: string, value?: string | string[] | boolean, prefix: string = "", postfix: string = "", resourcePrefix?: string) {
        if (value) {
            if (typeof value === "string") {
                value = `"${prefix}${value}${postfix}"`;
            } else if (Array.isArray(value)) {
                value = `["${value.join(`","`)}"]`;
            } else {
                value = value ? "true" : "false";
            }

            const resPrefix = resourcePrefix
                ? resourcePrefix + "."
                : "";

            const fld = field.indexOf('properties.') < 0 ? _.snakeCase(field) : field;

            this.filters += `${this.filters ? "," : ""}["${resPrefix}${fld}","${cond}",${value}]`;
        }
        return this;
    }
}
