/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.config;

import com.typesafe.config.Config;
import com.typesafe.config.ConfigFactory;

import java.io.File;

public class FileConfigProvider implements ConfigProvider {

    private static final String DEFAULT_PATH = "arvados";
    private final Config config;

    public FileConfigProvider() {
        config = ConfigFactory.load().getConfig(DEFAULT_PATH);
    }

    public FileConfigProvider(final String configFile) {
        config = (configFile != null) ?
                ConfigFactory.load(configFile).getConfig(DEFAULT_PATH) : ConfigFactory.load().getConfig(DEFAULT_PATH);
    }

    public Config getConfig() {
        return config;
    }

    private File getFile(String path) {
        return new File(config.getString(path));
    }

    private int getInt(String path) {
        return config.getInt(path);
    }

    private boolean getBoolean(String path) {
        return config.getBoolean(path);
    }

    private String getString(String path) {
        return config.getString(path);
    }

    @Override
    public boolean isApiHostInsecure() {
        return this.getBoolean("api.host-insecure");
    }

    @Override
    public String getKeepWebHost() {
        return this.getString("api.keepweb-host");
    }

    @Override
    public int getKeepWebPort() {
        return this.getInt("api.keepweb-port");
    }

    @Override
    public String getApiHost() {
        return this.getString("api.host");
    }

    @Override
    public int getApiPort() {
        return this.getInt("api.port");
    }

    @Override
    public String getApiToken() {
        return this.getString("api.token");
    }

    @Override
    public String getApiProtocol() {
        return this.getString("api.protocol");
    }

    @Override
    public int getFileSplitSize() {
        return this.getInt("split-size");
    }

    @Override
    public File getFileSplitDirectory() {
        return this.getFile("temp-dir");
    }

    @Override
    public int getNumberOfCopies() {
        return this.getInt("copies");
    }

    @Override
    public int getNumberOfRetries() {
        return this.getInt("retries");
    }

    public String getIntegrationTestProjectUuid() {
        return this.getString("integration-tests.project-uuid");
    }

    @Override
    public int getConnectTimeout() {
        return this.getInt("connectTimeout");
    }

    @Override
    public int getReadTimeout() {
        return this.getInt("readTimeout");
    }

    @Override
    public int getWriteTimeout() {
        return this.getInt("writeTimeout");
    }
}
