/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.logic.collection;

import org.arvados.client.common.Characters;
import org.arvados.client.exception.ArvadosClientException;
import org.arvados.client.logic.keep.KeepLocator;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.LinkedList;
import java.util.List;
import java.util.Objects;

import static java.util.stream.Collectors.toList;
import static org.arvados.client.common.Patterns.FILE_TOKEN_PATTERN;
import static org.arvados.client.common.Patterns.LOCATOR_PATTERN;

public class ManifestDecoder {

    public List<ManifestStream> decode(String manifestText) {

        if (manifestText == null || manifestText.isEmpty()) {
            throw new ArvadosClientException("Manifest text cannot be empty.");
        }

        List<String> manifestStreams = new ArrayList<>(Arrays.asList(manifestText.split("\\n")));
        if (!manifestStreams.get(0).startsWith(". ")) {
            throw new ArvadosClientException("Invalid first path component (expecting \".\")");
        }

        return manifestStreams.stream()
                .map(this::decodeSingleManifestStream)
                .collect(toList());
    }

    private ManifestStream decodeSingleManifestStream(String manifestStream) {
        Objects.requireNonNull(manifestStream, "Manifest stream cannot be empty.");

        LinkedList<String> manifestPieces = new LinkedList<>(Arrays.asList(manifestStream.split("\\s+")));
        String streamName = manifestPieces.poll();
        String path = ".".equals(streamName) ? "" : streamName.substring(2).concat(Characters.SLASH);

        List<KeepLocator> keepLocators = manifestPieces
                .stream()
                .filter(p -> p.matches(LOCATOR_PATTERN))
                .map(this::getKeepLocator)
                .collect(toList());


        List<FileToken> fileTokens = manifestPieces.stream()
                .skip(keepLocators.size())
                .filter(p -> p.matches(FILE_TOKEN_PATTERN))
                .map(p -> new FileToken(p, path))
                .collect(toList());

        return new ManifestStream(streamName, keepLocators, fileTokens);

    }

    private KeepLocator getKeepLocator(String locatorString ) {
        try {
            return new KeepLocator(locatorString);
        } catch (Exception e) {
            throw new RuntimeException(e);
        }
    }

}
