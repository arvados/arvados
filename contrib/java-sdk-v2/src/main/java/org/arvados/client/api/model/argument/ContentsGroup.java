/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.api.model.argument;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.JsonPropertyOrder;

import java.util.List;

@JsonInclude(JsonInclude.Include.NON_NULL)
@JsonPropertyOrder({ "limit", "order", "filters", "recursive" })
public class ContentsGroup extends Argument {

    @JsonProperty("limit")
    private Integer limit;

    @JsonProperty("order")
    private String order;

    @JsonProperty("filters")
    private List<String> filters;

    @JsonProperty("recursive")
    private Boolean recursive;

    public Integer getLimit() {
        return this.limit;
    }

    public String getOrder() {
        return this.order;
    }

    public List<String> getFilters() {
        return this.filters;
    }

    public Boolean getRecursive() {
        return this.recursive;
    }

    public void setLimit(Integer limit) {
        this.limit = limit;
    }

    public void setOrder(String order) {
        this.order = order;
    }

    public void setFilters(List<String> filters) {
        this.filters = filters;
    }

    public void setRecursive(Boolean recursive) {
        this.recursive = recursive;
    }
}
