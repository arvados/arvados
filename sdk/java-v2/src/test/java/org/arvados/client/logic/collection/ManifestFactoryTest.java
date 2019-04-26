/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.logic.collection;

import org.arvados.client.test.utils.FileTestUtils;
import org.assertj.core.util.Lists;
import org.junit.Test;
import org.junit.Ignore;

import java.io.File;
import java.util.List;

import static org.assertj.core.api.Assertions.assertThat;

public class ManifestFactoryTest {

    @Test
    @Ignore("Failing test #15041")
    public void manifestIsCreatedAsExpected() throws Exception {

        // given
        List<File> files = FileTestUtils.generatePredefinedFiles();
        List<String> locators = Lists.newArrayList("a", "b", "c");
        ManifestFactory factory = ManifestFactory.builder()
                .files(files)
                .locators(locators)
                .build();

        // when
        String actual = factory.create();

        // then
        assertThat(actual).isEqualTo(". a b c 0:1024:test-file1 1024:20480:test-file2 21504:1048576:test-file\\0403\n");
    }
}
