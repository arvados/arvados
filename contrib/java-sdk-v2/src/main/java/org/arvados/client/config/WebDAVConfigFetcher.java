/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.config;

import org.arvados.client.api.client.ConfigApiClient;
import org.arvados.client.api.model.ArvadosConfig;
import org.arvados.client.exception.ArvadosApiException;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.net.MalformedURLException;
import java.net.URL;

public class WebDAVConfigFetcher {
    
    private static final Logger log = LoggerFactory.getLogger(WebDAVConfigFetcher.class);
    private static final int DEFAULT_HTTPS_PORT = 443;
    private static final int DEFAULT_HTTP_PORT = 80;
    
    private final String apiProtocol;
    private final String apiHost;
    private final int apiPort;
    private final boolean apiHostInsecure;
    
    public WebDAVConfigFetcher(String apiProtocol, String apiHost, int apiPort, boolean apiHostInsecure) {
        this.apiProtocol = apiProtocol != null ? apiProtocol : "https";
        this.apiHost = apiHost;
        this.apiPort = apiPort > 0 ? apiPort : (this.apiProtocol.equals("https") ? DEFAULT_HTTPS_PORT : DEFAULT_HTTP_PORT);
        this.apiHostInsecure = apiHostInsecure;
    }
    
    public WebDAVConfig fetch() {
        if (!isConfigured()) {
            log.debug("API host not configured, skipping WebDAV auto-fetch");
            return null;
        }
        
        try {
            log.info("Attempting to auto-fetch WebDAV configuration from Arvados API");
            
            ArvadosConfig config = fetchArvadosConfig();
            String webDavUrl = extractWebDAVUrl(config);
            
            if (webDavUrl == null) {
                log.debug("No WebDAV URL found in Arvados config");
                return null;
            }
            
            return parseWebDAVUrl(webDavUrl);
            
        } catch (ArvadosApiException e) {
            log.warn("Failed to auto-fetch WebDAV configuration: {}. " +
                "You may need to configure keepWebHost and keepWebPort manually.", 
                e.getMessage());
        } catch (Exception e) {
            log.warn("Unexpected error while auto-fetching WebDAV configuration: {}. " +
                "You may need to configure keepWebHost and keepWebPort manually.", 
                e.getMessage());
        }
        
        return null;
    }
    
    private boolean isConfigured() {
        return apiHost != null && !apiHost.isEmpty();
    }
    
    private ArvadosConfig fetchArvadosConfig() throws ArvadosApiException {
        ConfigApiClient configClient = new ConfigApiClient(
            apiProtocol, apiHost, apiPort, apiHostInsecure
        );
        return configClient.fetchConfig();
    }
    
    private String extractWebDAVUrl(ArvadosConfig config) {
        if (config == null || config.getServices() == null) {
            return null;
        }
        
        ArvadosConfig.WebDAVDownload webDav = config.getServices().getWebDAVDownload();
        if (webDav == null) {
            return null;
        }
        
        return webDav.getExternalURL();
    }
    
    private WebDAVConfig parseWebDAVUrl(String webDavUrl) {
        if (webDavUrl == null || webDavUrl.isEmpty()) {
            return null;
        }
        
        try {
            URL url = new URL(webDavUrl);
            String host = url.getHost();
            int port = url.getPort();
            
            // Use default port based on protocol if not specified
            if (port == -1) {
                port = "https".equals(url.getProtocol()) ? DEFAULT_HTTPS_PORT : DEFAULT_HTTP_PORT;
            }
            
            log.info("Successfully auto-configured WebDAV: host={}, port={}", host, port);
            return new WebDAVConfig(host, port);
            
        } catch (MalformedURLException e) {
            log.warn("Failed to parse WebDAV URL '{}': {}", webDavUrl, e.getMessage());
            return null;
        }
    }
    
    public static class WebDAVConfig {
        private final String host;
        private final int port;
        
        public WebDAVConfig(String host, int port) {
            this.host = host;
            this.port = port;
        }
        
        public String getHost() {
            return host;
        }
        
        public int getPort() {
            return port;
        }
    }
}