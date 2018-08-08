/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.logic.collection;


import org.junit.Assert;
import org.junit.Test;

import java.util.List;

public class ManifestStreamTest {

    private ManifestDecoder manifestDecoder = new ManifestDecoder();

    @Test
    public void toStringReturnsProperlyConnectedManifestStream() throws Exception{
        String encodedManifest = ". eff999f3b5158331eb44a9a93e3b36e1+67108864 db141bfd11f7da60dce9e5ee85a988b8+34038725 0:101147589:rna.SRR948778.bam\\n\"";
        List<ManifestStream> manifestStreams = manifestDecoder.decode(encodedManifest);
        Assert.assertEquals(encodedManifest, manifestStreams.get(0).toString());

    }
}
