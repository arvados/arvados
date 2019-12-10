/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.logic.collection;

import com.google.common.base.Strings;
import org.arvados.client.common.Characters;

public class FileToken {

    private long filePosition;
    private long fileSize;
    private String fileName;
    private String path;

    public FileToken(String fileTokenInfo) {
        splitFileTokenInfo(fileTokenInfo);
    }

    public FileToken(String fileTokenInfo, String path) {
        splitFileTokenInfo(fileTokenInfo);
        this.path = path;
    }

    private void splitFileTokenInfo(String fileTokenInfo) {
        String[] tokenPieces = fileTokenInfo.split(":");
        this.filePosition = Long.parseLong(tokenPieces[0]);
        this.fileSize = Long.parseLong(tokenPieces[1]);
        this.fileName = tokenPieces[2].replace(Characters.SPACE, " ");
    }

    @Override
    public String toString() {
        return filePosition + ":" + fileSize + ":" + fileName;
    }

    public String getFullPath() {
        return Strings.isNullOrEmpty(path) ? fileName : path + fileName;
    }

    public long getFilePosition() {
        return this.filePosition;
    }

    public long getFileSize() {
        return this.fileSize;
    }

    public String getFileName() {
        return this.fileName;
    }

    public String getPath() {
        return this.path;
    }
}
