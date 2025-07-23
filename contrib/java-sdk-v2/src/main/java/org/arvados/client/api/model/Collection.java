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
@JsonPropertyOrder({ "portable_data_hash", "replication_desired", "replication_confirmed_at", "replication_confirmed", "manifest_text", 
    "name", "description", "properties", "delete_at", "trash_at", "is_trashed" })
public class Collection extends Item {

    @JsonProperty("portable_data_hash")
    private String portableDataHash;
    @JsonProperty("replication_desired")
    private Integer replicationDesired;
    @JsonProperty("replication_confirmed_at")
    private LocalDateTime replicationConfirmedAt;
    @JsonProperty("replication_confirmed")
    private Integer replicationConfirmed;
    @JsonProperty("manifest_text")
    private String manifestText;
    @JsonProperty("name")
    private String name;
    @JsonProperty("description")
    private String description;
    @JsonProperty("properties")
    private Object properties;
    @JsonProperty("delete_at")
    private LocalDateTime deleteAt;
    @JsonProperty("trash_at")
    private LocalDateTime trashAt;
    @JsonProperty("is_trashed")
    private Boolean trashed;

    public String getPortableDataHash() {
        return this.portableDataHash;
    }

    public Integer getReplicationDesired() {
        return this.replicationDesired;
    }

    public LocalDateTime getReplicationConfirmedAt() {
        return this.replicationConfirmedAt;
    }

    public Integer getReplicationConfirmed() {
        return this.replicationConfirmed;
    }

    public String getManifestText() {
        return this.manifestText;
    }

    public String getName() {
        return this.name;
    }

    public String getDescription() {
        return this.description;
    }

    public Object getProperties() {
        return this.properties;
    }

    public LocalDateTime getDeleteAt() {
        return this.deleteAt;
    }

    public LocalDateTime getTrashAt() {
        return this.trashAt;
    }

    public Boolean getTrashed() {
        return this.trashed;
    }

    public void setPortableDataHash(String portableDataHash) {
        this.portableDataHash = portableDataHash;
    }

    public void setReplicationDesired(Integer replicationDesired) {
        this.replicationDesired = replicationDesired;
    }

    public void setReplicationConfirmedAt(LocalDateTime replicationConfirmedAt) {
        this.replicationConfirmedAt = replicationConfirmedAt;
    }

    public void setReplicationConfirmed(Integer replicationConfirmed) {
        this.replicationConfirmed = replicationConfirmed;
    }

    public void setManifestText(String manifestText) {
        this.manifestText = manifestText;
    }

    public void setName(String name) {
        this.name = name;
    }

    public void setDescription(String description) {
        this.description = description;
    }

    public void setProperties(Object properties) {
        this.properties = properties;
    }

    public void setDeleteAt(LocalDateTime deleteAt) {
        this.deleteAt = deleteAt;
    }

    public void setTrashAt(LocalDateTime trashAt) {
        this.trashAt = trashAt;
    }

    public void setTrashed(Boolean trashed) {
        this.trashed = trashed;
    }

    public String toString() {
        return "Collection(portableDataHash=" + this.getPortableDataHash() + ", replicationDesired=" + this.getReplicationDesired() + ", replicationConfirmedAt=" + this.getReplicationConfirmedAt() + ", replicationConfirmed=" + this.getReplicationConfirmed() + ", manifestText=" + this.getManifestText() + ", name=" + this.getName() + ", description=" + this.getDescription() + ", properties=" + this.getProperties() + ", deleteAt=" + this.getDeleteAt() + ", trashAt=" + this.getTrashAt() + ", trashed=" + this.getTrashed() + ")";
    }
}