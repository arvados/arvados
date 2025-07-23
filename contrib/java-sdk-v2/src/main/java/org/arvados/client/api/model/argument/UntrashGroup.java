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

@JsonInclude(JsonInclude.Include.NON_NULL)
@JsonPropertyOrder({ "ensure_unique_name" })
public class UntrashGroup extends Argument {

    @JsonProperty("ensure_unique_name")
    private Boolean ensureUniqueName;

    public Boolean getEnsureUniqueName() {
        return this.ensureUniqueName;
    }

    public void setEnsureUniqueName(Boolean ensureUniqueName) {
        this.ensureUniqueName = ensureUniqueName;
    }
}
