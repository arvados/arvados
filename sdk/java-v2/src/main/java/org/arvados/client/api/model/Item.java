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

import java.time.LocalDateTime;

@JsonInclude(JsonInclude.Include.NON_NULL)
@JsonIgnoreProperties(ignoreUnknown = true)
@JsonPropertyOrder({ "kind", "etag", "uuid", "owner_uuid", "created_at", "modified_by_client_uuid",
        "modified_by_user_uuid", "modified_at", "updated_at" })
public abstract class Item {

    @JsonProperty("kind")
    private String kind;
    @JsonProperty("etag")
    private String etag;
    @JsonProperty("uuid")
    private String uuid;
    @JsonProperty("owner_uuid")
    private String ownerUuid;
    @JsonProperty("created_at")
    private LocalDateTime createdAt;
    @JsonProperty("modified_by_client_uuid")
    private String modifiedByClientUuid;
    @JsonProperty("modified_by_user_uuid")
    private String modifiedByUserUuid;
    @JsonProperty("modified_at")
    private LocalDateTime modifiedAt;
    @JsonProperty("updated_at")
    private LocalDateTime updatedAt;

    public String getKind() {
        return this.kind;
    }

    public String getEtag() {
        return this.etag;
    }

    public String getUuid() {
        return this.uuid;
    }

    public String getOwnerUuid() {
        return this.ownerUuid;
    }

    public LocalDateTime getCreatedAt() {
        return this.createdAt;
    }

    public String getModifiedByClientUuid() {
        return this.modifiedByClientUuid;
    }

    public String getModifiedByUserUuid() {
        return this.modifiedByUserUuid;
    }

    public LocalDateTime getModifiedAt() {
        return this.modifiedAt;
    }

    public LocalDateTime getUpdatedAt() {
        return this.updatedAt;
    }

    public void setKind(String kind) {
        this.kind = kind;
    }

    public void setEtag(String etag) {
        this.etag = etag;
    }

    public void setUuid(String uuid) {
        this.uuid = uuid;
    }

    public void setOwnerUuid(String ownerUuid) {
        this.ownerUuid = ownerUuid;
    }

    public void setCreatedAt(LocalDateTime createdAt) {
        this.createdAt = createdAt;
    }

    public void setModifiedByClientUuid(String modifiedByClientUuid) {
        this.modifiedByClientUuid = modifiedByClientUuid;
    }

    public void setModifiedByUserUuid(String modifiedByUserUuid) {
        this.modifiedByUserUuid = modifiedByUserUuid;
    }

    public void setModifiedAt(LocalDateTime modifiedAt) {
        this.modifiedAt = modifiedAt;
    }

    public void setUpdatedAt(LocalDateTime updatedAt) {
        this.updatedAt = updatedAt;
    }
}
