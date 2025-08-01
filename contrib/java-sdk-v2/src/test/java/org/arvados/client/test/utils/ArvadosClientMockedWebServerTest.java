/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.test.utils;

import okhttp3.mockwebserver.MockWebServer;
import org.junit.After;
import org.junit.Before;

public class ArvadosClientMockedWebServerTest extends ArvadosClientUnitTest {
    private static final int PORT = CONFIG.getApiPort();
    protected MockWebServer server = new MockWebServer();

    @Before
    public void setUpServer() throws Exception {
        server.start(PORT);
    }
    
    @After
    public void tearDownServer() throws Exception {
        server.shutdown();
    }
}
