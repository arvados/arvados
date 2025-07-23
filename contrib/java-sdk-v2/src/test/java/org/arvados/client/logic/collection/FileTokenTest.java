/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.logic.collection;

import org.arvados.client.common.Characters;
import org.junit.Assert;
import org.junit.Test;

public class FileTokenTest {

    public static final String FILE_TOKEN_INFO = "0:1024:test-file1";
    public static final int FILE_POSITION = 0;
    public static final long FILE_LENGTH = 1024L;
    public static final String FILE_NAME = "test-file1";
    public static final String FILE_PATH = "c" + Characters.SLASH;

    private static FileToken fileToken = new FileToken(FILE_TOKEN_INFO);
    private static FileToken fileTokenWithPath = new FileToken(FILE_TOKEN_INFO, FILE_PATH);

    @Test
    public void tokenInfoIsDividedCorrectly(){
        Assert.assertEquals(FILE_NAME, fileToken.getFileName());
        Assert.assertEquals(FILE_POSITION, fileToken.getFilePosition());
        Assert.assertEquals(FILE_LENGTH, fileToken.getFileSize());
    }

    @Test
    public void toStringReturnsOriginalFileTokenInfo(){
        Assert.assertEquals(FILE_TOKEN_INFO, fileToken.toString());
    }

    @Test
    public void fullPathIsReturnedProperly(){
        Assert.assertEquals(FILE_NAME, fileToken.getFullPath());
        Assert.assertEquals(FILE_PATH + FILE_NAME, fileTokenWithPath.getFullPath());
    }
}
