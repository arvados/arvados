/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.api.client;

import org.arvados.client.api.model.Link;
import org.arvados.client.api.model.LinkList;
import org.arvados.client.config.ConfigProvider;

public class LinksApiClient extends BaseStandardApiClient<Link, LinkList> {

    private static final String RESOURCE = "links";

    public LinksApiClient(ConfigProvider config) {
        super(config);
    }

    @Override
    String getResource() {
        return RESOURCE;
    }

    @Override
    Class<Link> getType() {
        return Link.class;
    }

    @Override
    Class<LinkList> getListType() {
        return LinkList.class;
    }
}
