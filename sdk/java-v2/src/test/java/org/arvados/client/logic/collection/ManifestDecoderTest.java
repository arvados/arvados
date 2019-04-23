/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.logic.collection;

import org.arvados.client.exception.ArvadosClientException;
import org.junit.Assert;
import org.junit.Test;

import java.util.List;

import static junit.framework.TestCase.fail;

public class ManifestDecoderTest {

    private ManifestDecoder manifestDecoder = new ManifestDecoder();

    private static final String ONE_LINE_MANIFEST_TEXT = ". " +
            "eff999f3b5158331eb44a9a93e3b36e1+67108864+Aad3839bea88bce22cbfe71cf4943de7dab3ea52a@5826180f " +
            "db141bfd11f7da60dce9e5ee85a988b8+34038725+Ae8f48913fed782cbe463e0499ab37697ee06a2f8@5826180f " +
            "0:101147589:rna.SRR948778.bam" +
            "\\n";

    private static final String MULTIPLE_LINES_MANIFEST_TEXT  = ". " +
            "930625b054ce894ac40596c3f5a0d947+33 " +
            "0:0:a 0:0:b 0:33:output.txt\n" +
            "./c d41d8cd98f00b204e9800998ecf8427e+0 0:0:d";

    private static final String MANIFEST_TEXT_WITH_INVALID_FIRST_PATH_COMPONENT = "a" + ONE_LINE_MANIFEST_TEXT;


    @Test
    public void allLocatorsAndFileTokensAreExtractedFromSimpleManifest() {

        List<ManifestStream> actual = manifestDecoder.decode(ONE_LINE_MANIFEST_TEXT);

        // one manifest stream
        Assert.assertEquals(1, actual.size());

        ManifestStream manifest = actual.get(0);
        // two locators
        Assert.assertEquals(2, manifest.getKeepLocators().size());
        // one file token
        Assert.assertEquals(1, manifest.getFileTokens().size());

    }

    @Test
    public void allLocatorsAndFileTokensAreExtractedFromComplexManifest() {

        List<ManifestStream> actual = manifestDecoder.decode(MULTIPLE_LINES_MANIFEST_TEXT);

        // two manifest streams
        Assert.assertEquals(2, actual.size());

        // first stream - 1 locator and 3 file tokens
        ManifestStream firstManifestStream = actual.get(0);
        Assert.assertEquals(1, firstManifestStream.getKeepLocators().size());
        Assert.assertEquals(3, firstManifestStream.getFileTokens().size());

        // second stream - 1 locator and 1 file token
        ManifestStream secondManifestStream = actual.get(1);
        Assert.assertEquals(1, secondManifestStream.getKeepLocators().size());
        Assert.assertEquals(1, secondManifestStream.getFileTokens().size());
    }

    @Test
    public void manifestTextWithInvalidStreamNameThrowsException() {

        try {
            List<ManifestStream> actual = manifestDecoder.decode(MANIFEST_TEXT_WITH_INVALID_FIRST_PATH_COMPONENT);
            fail();
        } catch (ArvadosClientException e) {
            Assert.assertEquals("Invalid first path component (expecting \".\")", e.getMessage());
        }

    }

    @Test
    public void emptyManifestTextThrowsException() {
        String emptyManifestText = null;

        try {
            List<ManifestStream> actual = manifestDecoder.decode(emptyManifestText);
            fail();
        } catch (ArvadosClientException e) {
            Assert.assertEquals("Manifest text cannot be empty.", e.getMessage());
        }

        emptyManifestText = "";
        try {
            List<ManifestStream> actual = manifestDecoder.decode(emptyManifestText);
            fail();
        } catch (ArvadosClientException e) {
            Assert.assertEquals("Manifest text cannot be empty.", e.getMessage());
        }

    }





}
