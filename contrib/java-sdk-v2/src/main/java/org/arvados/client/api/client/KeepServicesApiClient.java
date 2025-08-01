/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.api.client;

import org.arvados.client.api.model.KeepService;
import org.arvados.client.api.model.KeepServiceList;
import org.arvados.client.config.ConfigProvider;
import org.slf4j.Logger;

public class KeepServicesApiClient extends BaseStandardApiClient<KeepService, KeepServiceList> {

    private static final String RESOURCE = "keep_services";
    private final Logger log = org.slf4j.LoggerFactory.getLogger(KeepServicesApiClient.class);

    public KeepServicesApiClient(ConfigProvider config) {
        super(config);
    }

    public KeepServiceList accessible() {
        log.debug("Get list of accessible {}", getType().getSimpleName());
        return callForList(getNoArgumentMethodRequest("accessible"));
    }

    @Override
    String getResource() {
        return RESOURCE;
    }

    @Override
    Class<KeepService> getType() {
        return KeepService.class;
    }

    @Override
    Class<KeepServiceList> getListType() {
        return KeepServiceList.class;
    }
}
