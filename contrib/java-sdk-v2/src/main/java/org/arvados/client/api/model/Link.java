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
@JsonPropertyOrder({"name", "head_kind", "head_uuid", "link_class"})
public class Link extends Item {

    @JsonProperty("name")
    private String name;
    @JsonProperty(value = "head_kind", access = JsonProperty.Access.WRITE_ONLY)
    private String headKind;
    @JsonProperty("head_uuid")
    private String headUuid;
    @JsonProperty("tail_uuid")
    private String tailUuid;
    @JsonProperty(value = "tail_kind", access = JsonProperty.Access.WRITE_ONLY)
    private String tailKind;
    @JsonProperty("link_class")
    private String linkClass;

    public String getName() {
        return name;
    }

    public String getHeadKind() {
        return headKind;
    }

    public String getHeadUuid() {
        return headUuid;
    }

    public String getTailUuid() {
        return tailUuid;
    }

    public String getTailKind() {
        return tailKind;
    }

    public String getLinkClass() {
        return linkClass;
    }

    public void setName(String name) {
        this.name = name;
    }

    public void setHeadKind(String headKind) {
        this.headKind = headKind;
    }

    public void setHeadUuid(String headUuid) {
        this.headUuid = headUuid;
    }

    public void setTailUuid(String tailUuid) {
        this.tailUuid = tailUuid;
    }

    public void setTailKind(String tailKind) {
        this.tailKind = tailKind;
    }

    public void setLinkClass(String linkClass) {
        this.linkClass = linkClass;
    }

}
