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
@JsonPropertyOrder({ "errors", "error_token" })
public class ApiError {

    @JsonProperty("errors")
    private List<String> errors;
    @JsonProperty("error_token")
    private String errorToken;

    public List<String> getErrors() {
        return this.errors;
    }

    public String getErrorToken() {
        return this.errorToken;
    }

    public void setErrors(List<String> errors) {
        this.errors = errors;
    }

    public void setErrorToken(String errorToken) {
        this.errorToken = errorToken;
    }
}
