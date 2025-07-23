/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.common;

public final class Patterns {

    public static final String HINT_PATTERN = "^[A-Z][A-Za-z0-9@_-]+$";
    public static final String FILE_TOKEN_PATTERN = "(\\d+:\\d+:\\S+)";
    public static final String LOCATOR_PATTERN = "([0-9a-f]{32})\\+([0-9]+)(\\+[A-Z][-A-Za-z0-9@_]*)*";
    public static final String GROUP_UUID_PATTERN = "[a-z0-9]{5}-j7d0g-[a-z0-9]{15}";
    public static final String USER_UUID_PATTERN = "[a-z0-9]{5}-tpzed-[a-z0-9]{15}";

    private Patterns() {}
}
