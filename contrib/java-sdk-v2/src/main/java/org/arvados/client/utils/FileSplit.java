/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.utils;

import org.apache.commons.io.FileUtils;

import java.io.*;
import java.util.ArrayList;
import java.util.List;

/**
 * Based on:
 * {@link} https://stackoverflow.com/questions/10864317/how-to-break-a-file-into-pieces-using-java
 */
public class FileSplit {

    public static List<File> split(File f, File dir, int splitSize) throws IOException {
        int partCounter = 1;

        long sizeOfFiles = splitSize * FileUtils.ONE_MB;
        byte[] buffer = new byte[(int) sizeOfFiles];

        List<File> files = new ArrayList<>();
        String fileName = f.getName();

        try (FileInputStream fis = new FileInputStream(f); BufferedInputStream bis = new BufferedInputStream(fis)) {
            int bytesAmount = 0;
            while ((bytesAmount = bis.read(buffer)) > 0) {
                String filePartName = String.format("%s.%03d", fileName, partCounter++);
                File newFile = new File(dir, filePartName);
                try (FileOutputStream out = new FileOutputStream(newFile)) {
                    out.write(buffer, 0, bytesAmount);
                }
                files.add(newFile);
            }
        }
        return files;
    }
}