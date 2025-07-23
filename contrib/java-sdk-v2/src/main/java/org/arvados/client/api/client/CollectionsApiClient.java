/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.api.client;

import org.arvados.client.api.model.Collection;
import org.arvados.client.api.model.CollectionList;
import org.arvados.client.api.model.CollectionReplaceFiles;
import org.arvados.client.config.ConfigProvider;
import org.slf4j.Logger;

import okhttp3.HttpUrl;
import okhttp3.Request;
import okhttp3.RequestBody;

public class CollectionsApiClient extends BaseStandardApiClient<Collection, CollectionList> {

    private static final String RESOURCE = "collections";

    private final Logger log = org.slf4j.LoggerFactory.getLogger(CollectionsApiClient.class);

    public CollectionsApiClient(ConfigProvider config) {
        super(config);
    }
    
    @Override
    public Collection create(Collection type) {
        Collection newCollection = super.create(type);
        log.debug(String.format("New collection '%s' with UUID %s has been created", newCollection.getName(), newCollection.getUuid()));
        return newCollection;
    }

    public Collection update(String collectionUUID, CollectionReplaceFiles replaceFilesRequest) {
        String json = mapToJson(replaceFilesRequest);
        RequestBody body = RequestBody.create(JSON, json);
        HttpUrl url = getUrlBuilder().addPathSegment(collectionUUID).build();
        Request request = getRequestBuilder().put(body).url(url).build();
        return callForType(request);
    }

    @Override
    String getResource() {
        return RESOURCE;
    }

    @Override
    Class<Collection> getType() {
        return Collection.class;
    }

    @Override
    Class<CollectionList> getListType() {
        return CollectionList.class;
    }
}
