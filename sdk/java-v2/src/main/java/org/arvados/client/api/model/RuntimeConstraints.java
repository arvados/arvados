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
@JsonPropertyOrder({ "API", "vcpus", "ram", "keep_cache_ram" })
public class RuntimeConstraints {

    @JsonProperty("API")
    private Boolean api;
    @JsonProperty("vcpus")
    private Integer vcpus;
    @JsonProperty("ram")
    private Long ram;
    @JsonProperty("keep_cache_ram")
    private Long keepCacheRam;

    public Boolean getApi() {
        return this.api;
    }

    public Integer getVcpus() {
        return this.vcpus;
    }

    public Long getRam() {
        return this.ram;
    }

    public Long getKeepCacheRam() {
        return this.keepCacheRam;
    }

    public void setApi(Boolean api) {
        this.api = api;
    }

    public void setVcpus(Integer vcpus) {
        this.vcpus = vcpus;
    }

    public void setRam(Long ram) {
        this.ram = ram;
    }

    public void setKeepCacheRam(Long keepCacheRam) {
        this.keepCacheRam = keepCacheRam;
    }
}
