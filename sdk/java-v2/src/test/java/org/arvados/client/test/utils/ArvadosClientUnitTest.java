/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.test.utils;

import org.arvados.client.config.FileConfigProvider;
import org.junit.BeforeClass;

import static org.junit.Assert.assertTrue;

public class ArvadosClientUnitTest {

    protected static final FileConfigProvider CONFIG = new FileConfigProvider("application.conf");

    @BeforeClass
    public static void validateConfiguration(){
        String msg = " info must be provided in configuration";
        CONFIG.getConfig().entrySet().forEach(e -> assertTrue("Parameter " + e.getKey() + msg, !e.getValue().render().equals("\"\"")));
    }
}
