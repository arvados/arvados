/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.api.model;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.JsonPropertyOrder;

@JsonInclude(JsonInclude.Include.NON_NULL)
@JsonIgnoreProperties(ignoreUnknown = true)
@JsonPropertyOrder({ "kind", "etag", "offset", "limit", "items_available" })
public class ItemList {

    @JsonProperty("kind")
    private String kind;
    @JsonProperty("etag")
    private String etag;
    @JsonProperty("offset")
    private Object offset;
    @JsonProperty("limit")
    private Object limit;
    @JsonProperty("items_available")
    private Integer itemsAvailable;

    public String getKind() {
        return this.kind;
    }

    public String getEtag() {
        return this.etag;
    }

    public Object getOffset() {
        return this.offset;
    }

    public Object getLimit() {
        return this.limit;
    }

    public Integer getItemsAvailable() {
        return this.itemsAvailable;
    }

    public void setKind(String kind) {
        this.kind = kind;
    }

    public void setEtag(String etag) {
        this.etag = etag;
    }

    public void setOffset(Object offset) {
        this.offset = offset;
    }

    public void setLimit(Object limit) {
        this.limit = limit;
    }

    public void setItemsAvailable(Integer itemsAvailable) {
        this.itemsAvailable = itemsAvailable;
    }
}