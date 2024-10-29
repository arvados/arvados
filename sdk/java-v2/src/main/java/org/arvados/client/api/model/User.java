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

import java.util.List;

@JsonInclude(JsonInclude.Include.NON_NULL)
@JsonIgnoreProperties(ignoreUnknown = true)
@JsonPropertyOrder({ "email", "username", "full_name", "first_name", "last_name", "identity_url", "is_active", "is_admin", "is_invited", 
    "prefs", "writable_by" })
public class User extends Item {

    @JsonProperty("email")
    private String email;
    @JsonProperty("username")
    private String username;
    @JsonProperty("full_name")
    private String fullName;
    @JsonProperty("first_name")
    private String firstName;
    @JsonProperty("last_name")
    private String lastName;
    @JsonProperty("identity_url")
    private String identityUrl;
    @JsonProperty("is_active")
    private Boolean isActive;
    @JsonProperty("is_admin")
    private Boolean isAdmin;
    @JsonProperty("is_invited")
    private Boolean isInvited;
    @JsonProperty("prefs")
    private Object prefs;
    @JsonProperty("writable_by")
    private List<String> writableBy;

    public String getEmail() {
        return this.email;
    }

    public String getUsername() {
        return this.username;
    }

    public String getFullName() {
        return this.fullName;
    }

    public String getFirstName() {
        return this.firstName;
    }

    public String getLastName() {
        return this.lastName;
    }

    public String getIdentityUrl() {
        return this.identityUrl;
    }

    public Boolean getIsActive() {
        return this.isActive;
    }

    public Boolean getIsAdmin() {
        return this.isAdmin;
    }

    public Boolean getIsInvited() {
        return this.isInvited;
    }

    public Object getPrefs() {
        return this.prefs;
    }

    public List<String> getWritableBy() {
        return this.writableBy;
    }

    public void setEmail(String email) {
        this.email = email;
    }

    public void setUsername(String username) {
        this.username = username;
    }

    public void setFullName(String fullName) {
        this.fullName = fullName;
    }

    public void setFirstName(String firstName) {
        this.firstName = firstName;
    }

    public void setLastName(String lastName) {
        this.lastName = lastName;
    }

    public void setIdentityUrl(String identityUrl) {
        this.identityUrl = identityUrl;
    }

    public void setIsActive(Boolean isActive) {
        this.isActive = isActive;
    }

    public void setIsAdmin(Boolean isAdmin) {
        this.isAdmin = isAdmin;
    }

    public void setIsInvited(Boolean isInvited) {
        this.isInvited = isInvited;
    }

    public void setPrefs(Object prefs) {
        this.prefs = prefs;
    }

    public void setWritableBy(List<String> writableBy) {
        this.writableBy = writableBy;
    }

    public String toString() {
        return "User(email=" + this.getEmail() + ", username=" + this.getUsername() + ", fullName=" + this.getFullName() + ", firstName=" + this.getFirstName() + ", lastName=" + this.getLastName() + ", identityUrl=" + this.getIdentityUrl() + ", isActive=" + this.getIsActive() + ", isAdmin=" + this.getIsAdmin() + ", isInvited=" + this.getIsInvited() + ", prefs=" + this.getPrefs() + ", writableBy=" + this.getWritableBy() + ")";
    }
}
