/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.logic.keep;

import org.arvados.client.api.client.KeepServerApiClient;
import org.arvados.client.exception.ArvadosApiException;
import org.arvados.client.config.ConfigProvider;
import org.slf4j.Logger;

import java.io.File;
import java.util.Map;

public class FileTransferHandler {

    private final String host;
    private final KeepServerApiClient keepServerApiClient;
    private final Map<String, String> headers;
    private final Logger log = org.slf4j.LoggerFactory.getLogger(FileTransferHandler.class);

    public FileTransferHandler(String host, Map<String, String> headers, ConfigProvider config) {
        this.host = host;
        this.headers = headers;
        this.keepServerApiClient = new KeepServerApiClient(config);
    }

    public String put(String hashString, File body) {
        String url = host + hashString;
        String locator = null;
        try {
            locator = keepServerApiClient.upload(url, headers, body);
        } catch (ArvadosApiException e) {
            log.error("Cannot upload file to Keep server.", e);
        }
        return locator;
    }

    public byte[] get(KeepLocator locator) {
        return get(locator.stripped(), locator.permissionHint());
    }

    public byte[] get(String blockLocator, String authToken) {
        String url = host + blockLocator + "+" + authToken;
        try {
            return keepServerApiClient.download(url);
        } catch (ArvadosApiException e) {
            log.error("Cannot download file from Keep server.", e);
            return  null;
        }
    }
}
