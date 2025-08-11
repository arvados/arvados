/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.utils;

import org.arvados.client.test.utils.FileTestUtils;
import org.junit.After;
import org.junit.Before;
import org.junit.Test;

import java.io.File;
import java.util.List;

import static org.arvados.client.test.utils.FileTestUtils.*;
import static org.assertj.core.api.Assertions.assertThat;

public class FileSplitTest {

    @Before
    public void setUp() throws Exception {
        FileTestUtils.createDirectory(FILE_SPLIT_TEST_DIR);
    }

    @Test
    public void fileIsDividedIntoSmallerChunks() throws Exception {

        // given
        int expectedSize = 2;
        int expectedFileSizeInBytes = 67108864;
        FileTestUtils.generateFile(TEST_FILE, FileTestUtils.ONE_EIGTH_GB);

        // when
        List<File> actual = FileSplit.split(new File(TEST_FILE), new File(FILE_SPLIT_TEST_DIR), FILE_SPLIT_SIZE);

        // then
        assertThat(actual).hasSize(expectedSize);
        assertThat(actual).allMatch(a -> a.length() == expectedFileSizeInBytes);
    }

    @After
    public void tearDown() throws Exception {
        FileTestUtils.cleanDirectory(FILE_SPLIT_TEST_DIR);
    }
}
