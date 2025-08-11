/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.test.utils;

import org.apache.commons.io.FileUtils;
import org.assertj.core.util.Lists;

import java.io.File;
import java.io.IOException;
import java.io.RandomAccessFile;
import java.util.List;

public class FileTestUtils {

    public static final String FILE_SPLIT_TEST_DIR = "/tmp/file-split";
    public static final String FILE_DOWNLOAD_TEST_DIR = "/tmp/arvados-downloaded";
    public static final String TEST_FILE = FILE_SPLIT_TEST_DIR + "/test-file";
    public static long ONE_FOURTH_GB = FileUtils.ONE_GB / 4;
    public static long ONE_EIGTH_GB = FileUtils.ONE_GB / 8;
    public static long HALF_GB = FileUtils.ONE_GB / 2;
    public static int FILE_SPLIT_SIZE = 64;

    public static void createDirectory(String path) throws Exception {
        new File(path).mkdirs();
    }

    public static void cleanDirectory(String directory) throws Exception {
        FileUtils.cleanDirectory(new File(directory));
    }
    
    public static File generateFile(String path, long length) throws IOException {
        RandomAccessFile testFile = new RandomAccessFile(path, "rwd");
        testFile.setLength(length);
        testFile.close();
        return new File(path);
    }
    
    public static List<File> generatePredefinedFiles() throws IOException {
        return Lists.newArrayList(
                generateFile(TEST_FILE + 1, FileUtils.ONE_KB),
                generateFile(TEST_FILE + 2, FileUtils.ONE_KB * 20),
                generateFile(TEST_FILE + " " + 3, FileUtils.ONE_MB)
            );
    }
}
