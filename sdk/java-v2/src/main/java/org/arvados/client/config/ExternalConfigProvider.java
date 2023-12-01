/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.config;

import java.io.File;

public class ExternalConfigProvider implements ConfigProvider {

    private static final int DEFAULT_CONNECTION_TIMEOUT = 60000;
    private static final int DEFAULT_READ_TIMEOUT = 60000;
    private static final int DEFAULT_WRITE_TIMEOUT = 60000;

    private boolean apiHostInsecure;
    private String keepWebHost;
    private int keepWebPort;
    private String apiHost;
    private int apiPort;
    private String apiToken;
    private String apiProtocol;
    private int fileSplitSize;
    private File fileSplitDirectory;
    private int numberOfCopies;
    private int numberOfRetries;
    private int connectTimeout;
    private int readTimeout;
    private int writeTimeout;

    ExternalConfigProvider(boolean apiHostInsecure, String keepWebHost, int keepWebPort, String apiHost, int apiPort,
			   String apiToken, String apiProtocol, int fileSplitSize, File fileSplitDirectory,
			   int numberOfCopies, int numberOfRetries)
    {
        this.apiHostInsecure = apiHostInsecure;
        this.keepWebHost = keepWebHost;
        this.keepWebPort = keepWebPort;
        this.apiHost = apiHost;
        this.apiPort = apiPort;
        this.apiToken = apiToken;
        this.apiProtocol = apiProtocol;
        this.fileSplitSize = fileSplitSize;
        this.fileSplitDirectory = fileSplitDirectory;
        this.numberOfCopies = numberOfCopies;
        this.numberOfRetries = numberOfRetries;
	this.connectTimeout = DEFAULT_CONNECTION_TIMEOUT;
	this.readTimeout = DEFAULT_READ_TIMEOUT;
	this.writeTimeout = DEFAULT_WRITE_TIMEOUT;
    }

    ExternalConfigProvider(boolean apiHostInsecure, String keepWebHost, int keepWebPort, String apiHost, int apiPort,
			   String apiToken, String apiProtocol, int fileSplitSize, File fileSplitDirectory,
			   int numberOfCopies, int numberOfRetries,
			   int connectTimeout, int readTimeout, int writeTimeout)
    {
        this.apiHostInsecure = apiHostInsecure;
        this.keepWebHost = keepWebHost;
        this.keepWebPort = keepWebPort;
        this.apiHost = apiHost;
        this.apiPort = apiPort;
        this.apiToken = apiToken;
        this.apiProtocol = apiProtocol;
        this.fileSplitSize = fileSplitSize;
        this.fileSplitDirectory = fileSplitDirectory;
        this.numberOfCopies = numberOfCopies;
        this.numberOfRetries = numberOfRetries;
	this.connectTimeout = connectTimeout;
	this.readTimeout = readTimeout;
	this.writeTimeout = writeTimeout;
    }

    public static ExternalConfigProviderBuilder builder() {
        return new ExternalConfigProviderBuilder();
    }

    @Override
    public String toString() {
        return "ExternalConfigProvider{" +
                "apiHostInsecure=" + apiHostInsecure +
                ", keepWebHost='" + keepWebHost + '\'' +
                ", keepWebPort=" + keepWebPort +
                ", apiHost='" + apiHost + '\'' +
                ", apiPort=" + apiPort +
                ", apiToken='" + apiToken + '\'' +
                ", apiProtocol='" + apiProtocol + '\'' +
                ", fileSplitSize=" + fileSplitSize +
                ", fileSplitDirectory=" + fileSplitDirectory +
                ", numberOfCopies=" + numberOfCopies +
                ", numberOfRetries=" + numberOfRetries +
                '}';
    }

    public boolean isApiHostInsecure() {
        return this.apiHostInsecure;
    }

    public String getKeepWebHost() {
        return this.keepWebHost;
    }

    public int getKeepWebPort() {
        return this.keepWebPort;
    }

    public String getApiHost() {
        return this.apiHost;
    }

    public int getApiPort() {
        return this.apiPort;
    }

    public String getApiToken() {
        return this.apiToken;
    }

    public String getApiProtocol() {
        return this.apiProtocol;
    }

    public int getFileSplitSize() {
        return this.fileSplitSize;
    }

    public File getFileSplitDirectory() {
        return this.fileSplitDirectory;
    }

    public int getNumberOfCopies() {
        return this.numberOfCopies;
    }

    public int getNumberOfRetries() {
        return this.numberOfRetries;
    }

    public int getConnectTimeout() {
        return this.connectTimeout;
    }

    public int getReadTimeout() {
        return this.readTimeout;
    }

    public int getWriteTimeout() {
        return this.writeTimeout;
    }

    public static class ExternalConfigProviderBuilder {
        private boolean apiHostInsecure;
        private String keepWebHost;
        private int keepWebPort;
        private String apiHost;
        private int apiPort;
        private String apiToken;
        private String apiProtocol;
        private int fileSplitSize;
        private File fileSplitDirectory;
        private int numberOfCopies;
        private int numberOfRetries;
        private int connectTimeout = DEFAULT_CONNECTION_TIMEOUT;
        private int readTimeout = DEFAULT_READ_TIMEOUT;
        private int writeTimeout = DEFAULT_WRITE_TIMEOUT;

        ExternalConfigProviderBuilder() {
        }

        public ExternalConfigProvider.ExternalConfigProviderBuilder apiHostInsecure(boolean apiHostInsecure) {
            this.apiHostInsecure = apiHostInsecure;
            return this;
        }

        public ExternalConfigProvider.ExternalConfigProviderBuilder keepWebHost(String keepWebHost) {
            this.keepWebHost = keepWebHost;
            return this;
        }

        public ExternalConfigProvider.ExternalConfigProviderBuilder keepWebPort(int keepWebPort) {
            this.keepWebPort = keepWebPort;
            return this;
        }

        public ExternalConfigProvider.ExternalConfigProviderBuilder apiHost(String apiHost) {
            this.apiHost = apiHost;
            return this;
        }

        public ExternalConfigProvider.ExternalConfigProviderBuilder apiPort(int apiPort) {
            this.apiPort = apiPort;
            return this;
        }

        public ExternalConfigProvider.ExternalConfigProviderBuilder apiToken(String apiToken) {
            this.apiToken = apiToken;
            return this;
        }

        public ExternalConfigProvider.ExternalConfigProviderBuilder apiProtocol(String apiProtocol) {
            this.apiProtocol = apiProtocol;
            return this;
        }

        public ExternalConfigProvider.ExternalConfigProviderBuilder fileSplitSize(int fileSplitSize) {
            this.fileSplitSize = fileSplitSize;
            return this;
        }

        public ExternalConfigProvider.ExternalConfigProviderBuilder fileSplitDirectory(File fileSplitDirectory) {
            this.fileSplitDirectory = fileSplitDirectory;
            return this;
        }

        public ExternalConfigProvider.ExternalConfigProviderBuilder numberOfCopies(int numberOfCopies) {
            this.numberOfCopies = numberOfCopies;
            return this;
        }

        public ExternalConfigProvider.ExternalConfigProviderBuilder numberOfRetries(int numberOfRetries) {
            this.numberOfRetries = numberOfRetries;
            return this;
        }

        public ExternalConfigProvider.ExternalConfigProviderBuilder connectTimeout(int connectTimeout) {
            this.connectTimeout = connectTimeout;
            return this;
        }

        public ExternalConfigProvider.ExternalConfigProviderBuilder readTimeout(int readTimeout) {
            this.readTimeout = readTimeout;
            return this;
        }

        public ExternalConfigProvider.ExternalConfigProviderBuilder writeTimeout(int writeTimeout) {
            this.writeTimeout = writeTimeout;
            return this;
        }

        public ExternalConfigProvider build() {
            return new ExternalConfigProvider(apiHostInsecure, keepWebHost, keepWebPort, apiHost, apiPort, apiToken, apiProtocol, fileSplitSize, fileSplitDirectory, numberOfCopies, numberOfRetries, connectTimeout, readTimeout, writeTimeout);
        }

    }
}
