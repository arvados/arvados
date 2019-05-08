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

public class FileMergeTest {

    @Before
    public void setUp() throws Exception {
        FileTestUtils.createDirectory(FILE_SPLIT_TEST_DIR);
    }

    @Test
    public void fileChunksAreMergedIntoOneFile() throws Exception {

        // given
        FileTestUtils.generateFile(TEST_FILE, FileTestUtils.ONE_EIGTH_GB);

        List<File> files = FileSplit.split(new File(TEST_FILE), new File(FILE_SPLIT_TEST_DIR), FILE_SPLIT_SIZE);
        File targetFile = new File(TEST_FILE);

        // when
        FileMerge.merge(files, targetFile);

        // then
        assertThat(targetFile.length()).isEqualTo(FileTestUtils.ONE_EIGTH_GB);
    }

    @After
    public void tearDown() throws Exception {
        FileTestUtils.cleanDirectory(FILE_SPLIT_TEST_DIR);
    }
}
