/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.logic.collection;

import org.arvados.client.logic.keep.KeepLocator;

import java.util.List;
import java.util.stream.Collectors;
import java.util.stream.Stream;

public class ManifestStream {

    private String streamName;
    private List<KeepLocator> keepLocators;
    private List<FileToken> fileTokens;

    public ManifestStream(String streamName, List<KeepLocator> keepLocators, List<FileToken> fileTokens) {
        this.streamName = streamName;
        this.keepLocators = keepLocators;
        this.fileTokens = fileTokens;
    }

    @Override
    public String toString() {
        return streamName + " " + Stream.concat(keepLocators.stream().map(KeepLocator::toString), fileTokens.stream().map(FileToken::toString))
                .collect(Collectors.joining(" "));
    }

    public String getStreamName() {
        return this.streamName;
    }

    public List<KeepLocator> getKeepLocators() {
        return this.keepLocators;
    }

    public List<FileToken> getFileTokens() {
        return this.fileTokens;
    }
}
