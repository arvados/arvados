/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.api.client;

import okhttp3.Request;
import org.arvados.client.api.model.User;
import org.arvados.client.api.model.UserList;
import org.arvados.client.config.ConfigProvider;
import org.slf4j.Logger;

public class UsersApiClient extends BaseStandardApiClient<User, UserList> {

    private static final String RESOURCE = "users";
    private final Logger log = org.slf4j.LoggerFactory.getLogger(UsersApiClient.class);

    public UsersApiClient(ConfigProvider config) {
        super(config);
    }

    public User current() {
        log.debug("Get current {}", getType().getSimpleName());
        Request request = getNoArgumentMethodRequest("current");
        return callForType(request);
    }

    public User system() {
        log.debug("Get system {}", getType().getSimpleName());
        Request request = getNoArgumentMethodRequest("system");
        return callForType(request);
    }

    @Override
    String getResource() {
        return RESOURCE;
    }

    @Override
    Class<User> getType() {
        return User.class;
    }

    @Override
    Class<UserList> getListType() {
        return UserList.class;
    }
}
