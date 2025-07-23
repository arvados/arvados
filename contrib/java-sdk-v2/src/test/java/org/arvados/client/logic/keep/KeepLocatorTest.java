/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.logic.keep;

import org.junit.Test;

import static org.assertj.core.api.Assertions.assertThat;

public class KeepLocatorTest {

    private KeepLocator locator;

    @Test
    public void md5sumIsExtracted() throws Exception {

        // given
        locator = new KeepLocator("7df44272090cee6c0732382bba415ee9+70");

        // when
        String actual = locator.getMd5sum();

        // then
        assertThat(actual).isEqualTo("7df44272090cee6c0732382bba415ee9");
    }

    @Test
    public void locatorIsStrippedWithMd5sumAndSize() throws Exception {

        // given
        locator = new KeepLocator("7df44272090cee6c0732382bba415ee9+70");

        // when
        String actual = locator.stripped();

        // then
        assertThat(actual).isEqualTo("7df44272090cee6c0732382bba415ee9+70");
    }


    @Test
    public void locatorToStringProperlyShowing() throws Exception {

        // given
        locator = new KeepLocator("7df44272090cee6c0732382bba415ee9+70+Ae8f48913fed782cbe463e0499ab37697ee06a2f8@5826180f");

        // when
        String actual = locator.toString();

        // then
        assertThat(actual).isEqualTo("7df44272090cee6c0732382bba415ee9+70+Ae8f48913fed782cbe463e0499ab37697ee06a2f8@5826180f");
    }
}
