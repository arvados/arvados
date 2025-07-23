/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.api.client;

import okhttp3.HttpUrl;
import okhttp3.HttpUrl.Builder;
import okhttp3.Request;
import okhttp3.RequestBody;
import org.arvados.client.api.model.Group;
import org.arvados.client.api.model.GroupList;
import org.arvados.client.api.model.argument.ContentsGroup;
import org.arvados.client.api.model.argument.ListArgument;
import org.arvados.client.api.model.argument.UntrashGroup;
import org.arvados.client.config.ConfigProvider;
import org.slf4j.Logger;

public class GroupsApiClient extends BaseStandardApiClient<Group, GroupList> {

    private static final String RESOURCE = "groups";
    private final Logger log = org.slf4j.LoggerFactory.getLogger(GroupsApiClient.class);

    public GroupsApiClient(ConfigProvider config) {
        super(config);
    }

    public GroupList contents(ContentsGroup contentsGroup) {
        log.debug("Get {} contents", getType().getSimpleName());
        Builder urlBuilder = getUrlBuilder().addPathSegment("contents");
        addQueryParameters(urlBuilder, contentsGroup);
        HttpUrl url = urlBuilder.build();
        Request request = getRequestBuilder().url(url).build();
        return callForList(request);
    }

    public GroupList contents(ListArgument listArguments) {
        this.log.debug("Get {} contents", this.getType().getSimpleName());
        HttpUrl.Builder urlBuilder = this.getUrlBuilder().addPathSegment("contents");
        this.addQueryParameters(urlBuilder, listArguments);
        HttpUrl url = urlBuilder.build();
        Request request = this.getRequestBuilder().url(url).build();
        return callForList(request);
    }

    public Group untrash(UntrashGroup untrashGroup) {
        log.debug("Untrash {} by UUID {}", getType().getSimpleName(), untrashGroup.getUuid());
        HttpUrl url = getUrlBuilder().addPathSegment(untrashGroup.getUuid()).addPathSegment("untrash").build();
        RequestBody requestBody = getJsonRequestBody(untrashGroup);
        Request request = getRequestBuilder().post(requestBody).url(url).build();
        return callForType(request);
    }

    @Override
    String getResource() {
        return RESOURCE;
    }

    @Override
    Class<Group> getType() {
        return Group.class;
    }

    @Override
    Class<GroupList> getListType() {
        return GroupList.class;
    }
}
