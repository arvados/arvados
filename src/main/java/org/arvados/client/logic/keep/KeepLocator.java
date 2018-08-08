/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.logic.keep;

import org.arvados.client.exception.ArvadosClientException;

import java.time.Instant;
import java.time.LocalDateTime;
import java.time.ZoneOffset;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.LinkedList;
import java.util.List;
import java.util.Objects;
import java.util.stream.Collectors;
import java.util.stream.Stream;

import static org.arvados.client.common.Patterns.HINT_PATTERN;

public class KeepLocator {

    private final List<String> hints = new ArrayList<>();
    private String permSig;
    private LocalDateTime permExpiry;
    private final String md5sum;
    private final Integer size;

    public KeepLocator(String locatorString) {
        LinkedList<String> pieces = new LinkedList<>(Arrays.asList(locatorString.split("\\+")));

        md5sum = pieces.poll();
        size = Integer.valueOf(Objects.requireNonNull(pieces.poll()));

        for (String hint : pieces) {
            if (!hint.matches(HINT_PATTERN)) {
                throw new ArvadosClientException(String.format("invalid hint format: %s", hint));
            } else if (hint.startsWith("A")) {
                parsePermissionHint(hint);
            } else {
                hints.add(hint);
            }
        }
    }

    public List<String> getHints() {
        return hints;
    }

    public String getMd5sum() {
        return md5sum;
    }

    @Override
    public String toString() {
        return Stream.concat(Stream.of(md5sum, size.toString(), permissionHint()), hints.stream())
                .filter(Objects::nonNull)
                .collect(Collectors.joining("+"));
    }

    public String stripped() {
        return size != null ? String.format("%s+%d", md5sum, size) : md5sum;
    }

    public String permissionHint() {
        if (permSig == null || permExpiry == null) {
            return null;
        }

        long timestamp = permExpiry.toEpochSecond(ZoneOffset.UTC);
        String signTimestamp = Long.toHexString(timestamp);
        return String.format("A%s@%s", permSig, signTimestamp);
    }

    private void parsePermissionHint(String hint) {
        String[] hintSplit = hint.substring(1).split("@", 2);
        permSig = hintSplit[0];

        int permExpiryDecimal = Integer.parseInt(hintSplit[1], 16);
        permExpiry = LocalDateTime.ofInstant(Instant.ofEpochSecond(permExpiryDecimal), ZoneOffset.UTC);
    }
}
