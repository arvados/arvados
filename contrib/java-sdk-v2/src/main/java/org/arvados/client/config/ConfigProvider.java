/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.config;

import java.io.File;

public interface ConfigProvider {

    //API
    boolean isApiHostInsecure();

    String getKeepWebHost();

    int getKeepWebPort();

    String getApiHost();

    int getApiPort();

    String getApiToken();

    String getApiProtocol();

    int getConnectTimeout();

    int getReadTimeout();

    int getWriteTimeout();

    //FILE UPLOAD
    int getFileSplitSize();

    File getFileSplitDirectory();

    int getNumberOfCopies();

    int getNumberOfRetries();


}
