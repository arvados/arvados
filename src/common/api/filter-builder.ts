// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export enum FilterField {
    UUID = "uuid"
}

export default class FilterBuilder {
    private filters = "";

    private addCondition(field: FilterField, cond: string, value?: string) {
        if (value) {
            this.filters += `["${field}","${cond}","${value}"]`;
        }
        return this;
    }

    public addEqual(field: FilterField, value?: string) {
        return this.addCondition(field, "=", value);
    }

    public addLike(field: FilterField, value?: string) {
        return this.addCondition(field, "like", value);
    }

    public addILike(field: FilterField, value?: string) {
        return this.addCondition(field, "ilike", value);
    }

    public get() {
        return "[" + this.filters + "]";
    }
}
