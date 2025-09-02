/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.api.model;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;

@JsonIgnoreProperties(ignoreUnknown = true)
public class ArvadosConfig {

    @JsonProperty("Services")
    private Services services;

    public Services getServices() {
        return services;
    }

    public void setServices(Services services) {
        this.services = services;
    }

    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class Services {

        @JsonProperty("WebDAVDownload")
        private WebDAVDownload webDAVDownload;

        public WebDAVDownload getWebDAVDownload() {
            return webDAVDownload;
        }

        public void setWebDAVDownload(WebDAVDownload webDAVDownload) {
            this.webDAVDownload = webDAVDownload;
        }
    }

    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class WebDAVDownload {

        @JsonProperty("ExternalURL")
        private String externalURL;

        public String getExternalURL() {
            return externalURL;
        }

        public void setExternalURL(String externalURL) {
            this.externalURL = externalURL;
        }
    }
}