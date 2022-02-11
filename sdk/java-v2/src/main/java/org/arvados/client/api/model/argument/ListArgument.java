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
@JsonPropertyOrder({ "limit", "offset", "filters", "order", "select", "distinct", "count", "exclude_home_project" })
public class ListArgument extends Argument {

    @JsonProperty("limit")
    private Integer limit;

    @JsonProperty("offset")
    private Integer offset;
    
    @JsonProperty("filters")
    private List<Filter> filters;

    @JsonProperty("order")
    private List<String> order;

    @JsonProperty("select")
    private List<String> select;

    @JsonProperty("distinct")
    private Boolean distinct;

    @JsonProperty("count")
    private Count count;

    @JsonProperty("exclude_home_project")
    private Boolean excludeHomeProject;

    ListArgument(Integer limit, Integer offset, List<Filter> filters, List<String> order, List<String> select, Boolean distinct, Count count, Boolean excludeHomeProject) {
        this.limit = limit;
        this.offset = offset;
        this.filters = filters;
        this.order = order;
        this.select = select;
        this.distinct = distinct;
        this.count = count;
        this.excludeHomeProject = excludeHomeProject;
    }

    public static ListArgumentBuilder builder() {
        return new ListArgumentBuilder();
    }

    public enum Count {
        
        @JsonProperty("exact")
        EXACT,
        
        @JsonProperty("none")
        NONE
    }

    public static class ListArgumentBuilder {
        private Integer limit;
        private Integer offset;
        private List<Filter> filters;
        private List<String> order;
        private List<String> select;
        private Boolean distinct;
        private Count count;
        private Boolean excludeHomeProject;

        ListArgumentBuilder() {
        }

        public ListArgumentBuilder limit(Integer limit) {
            this.limit = limit;
            return this;
        }

        public ListArgumentBuilder offset(Integer offset) {
            this.offset = offset;
            return this;
        }

        public ListArgumentBuilder filters(List<Filter> filters) {
            this.filters = filters;
            return this;
        }

        public ListArgumentBuilder order(List<String> order) {
            this.order = order;
            return this;
        }

        public ListArgumentBuilder select(List<String> select) {
            this.select = select;
            return this;
        }

        public ListArgumentBuilder distinct(Boolean distinct) {
            this.distinct = distinct;
            return this;
        }

        public ListArgumentBuilder count(Count count) {
            this.count = count;
            return this;
        }

        public ListArgument.ListArgumentBuilder excludeHomeProject(Boolean excludeHomeProject) {
            this.excludeHomeProject = excludeHomeProject;
            return this;
        }

        public ListArgument build() {
            return new ListArgument(limit, offset, filters, order, select, distinct, count, excludeHomeProject);
        }

        public String toString() {
            return "ListArgument.ListArgumentBuilder(limit=" + this.limit +
                    ", offset=" + this.offset + ", filters=" + this.filters +
                    ", order=" + this.order + ", select=" + this.select +
                    ", distinct=" + this.distinct + ", count=" + this.count +
                    ", excludeHomeProject=" + this.excludeHomeProject + ")";
        }
    }
}
