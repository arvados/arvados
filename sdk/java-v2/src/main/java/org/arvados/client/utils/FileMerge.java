/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.utils;

import java.io.BufferedOutputStream;
import java.io.File;
import java.io.FileOutputStream;
import java.io.IOException;
import java.nio.file.Files;
import java.util.Collection;

public class FileMerge {

    public static void merge(Collection<File> files, File targetFile) throws IOException {
        try (FileOutputStream fos = new FileOutputStream(targetFile); BufferedOutputStream mergingStream = new BufferedOutputStream(fos)) {
            for (File file : files) {
                Files.copy(file.toPath(), mergingStream);
            }
        }
    }
}
