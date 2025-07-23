/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.test.utils;

import org.arvados.client.config.FileConfigProvider;
import org.arvados.client.facade.ArvadosFacade;
import org.junit.BeforeClass;

import static org.junit.Assert.assertTrue;

public class ArvadosClientIntegrationTest {

    protected static final FileConfigProvider CONFIG = new FileConfigProvider("integration-tests-application.conf");
    protected static final ArvadosFacade FACADE = new ArvadosFacade(CONFIG);
    protected static final String PROJECT_UUID = CONFIG.getIntegrationTestProjectUuid();

    @BeforeClass
    public static void validateConfiguration(){
        String msg = " info must be provided in configuration";
        CONFIG.getConfig().entrySet()
                .forEach(e -> assertTrue("Parameter " + e.getKey() + msg, !e.getValue().render().equals("\"\"")));
    }
}
