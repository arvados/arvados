/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.api.model;

import com.fasterxml.jackson.annotation.*;

@JsonInclude(JsonInclude.Include.NON_NULL)
@JsonIgnoreProperties(ignoreUnknown = true)
@JsonPropertyOrder({ "service_host", "service_port", "service_ssl_flag", "service_type", "read_only" })
public class KeepService extends Item {

    @JsonProperty("service_host")
    private String serviceHost;
    @JsonProperty("service_port")
    private Integer servicePort;
    @JsonProperty("service_ssl_flag")
    private Boolean serviceSslFlag;
    @JsonProperty("service_type")
    private String serviceType;
    @JsonProperty("read_only")
    private Boolean readOnly;
    @JsonIgnore
    private String serviceRoot;

    public String getServiceHost() {
        return this.serviceHost;
    }

    public Integer getServicePort() {
        return this.servicePort;
    }

    public Boolean getServiceSslFlag() {
        return this.serviceSslFlag;
    }

    public String getServiceType() {
        return this.serviceType;
    }

    public Boolean getReadOnly() {
        return this.readOnly;
    }

    public String getServiceRoot() {
        return this.serviceRoot;
    }

    public void setServiceHost(String serviceHost) {
        this.serviceHost = serviceHost;
    }

    public void setServicePort(Integer servicePort) {
        this.servicePort = servicePort;
    }

    public void setServiceSslFlag(Boolean serviceSslFlag) {
        this.serviceSslFlag = serviceSslFlag;
    }

    public void setServiceType(String serviceType) {
        this.serviceType = serviceType;
    }

    public void setReadOnly(Boolean readOnly) {
        this.readOnly = readOnly;
    }

    public void setServiceRoot(String serviceRoot) {
        this.serviceRoot = serviceRoot;
    }
}