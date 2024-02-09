/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.api.client;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectWriter;
import okhttp3.MediaType;
import okhttp3.HttpUrl;
import okhttp3.HttpUrl.Builder;
import okhttp3.Request;
import okhttp3.RequestBody;
import org.arvados.client.exception.ArvadosApiException;
import org.arvados.client.api.model.Item;
import org.arvados.client.api.model.ItemList;
import org.arvados.client.api.model.argument.ListArgument;
import org.arvados.client.config.ConfigProvider;
import org.slf4j.Logger;

import java.io.IOException;
import java.util.Map;

public abstract class BaseStandardApiClient<T extends Item, L extends ItemList> extends BaseApiClient {

    protected static final MediaType JSON = MediaType.parse(com.google.common.net.MediaType.JSON_UTF_8.toString());
    private final Logger log = org.slf4j.LoggerFactory.getLogger(BaseStandardApiClient.class);

    BaseStandardApiClient(ConfigProvider config) {
        super(config);
    }

    public L list(ListArgument listArguments) {
        log.debug("Get list of {}", getType().getSimpleName());
        Builder urlBuilder = getUrlBuilder();
        addQueryParameters(urlBuilder, listArguments);
        HttpUrl url = urlBuilder.build();
        Request request = getRequestBuilder().url(url).build();
        return callForList(request);
    }
    
    public L list() {
        return list(ListArgument.builder().build());
    }

    public T get(String uuid) {
        log.debug("Get {} by UUID {}", getType().getSimpleName(), uuid);
        HttpUrl url = getUrlBuilder().addPathSegment(uuid).build();
        Request request = getRequestBuilder().get().url(url).build();
        return callForType(request);
    }

    public T create(T type) {
        log.debug("Create {}", getType().getSimpleName());
        String json = mapToJson(type);
        RequestBody body = RequestBody.create(JSON, json);
        Request request = getRequestBuilder().post(body).build();
        return callForType(request);
    }

    public T delete(String uuid) {
        log.debug("Delete {} by UUID {}", getType().getSimpleName(), uuid);
        HttpUrl url = getUrlBuilder().addPathSegment(uuid).build();
        Request request = getRequestBuilder().delete().url(url).build();
        return callForType(request);
    }

    public T update(T type) {
        String uuid = type.getUuid();
        log.debug("Update {} by UUID {}", getType().getSimpleName(), uuid);
        String json = mapToJson(type);
        RequestBody body = RequestBody.create(JSON, json);
        HttpUrl url = getUrlBuilder().addPathSegment(uuid).build();
        Request request = getRequestBuilder().put(body).url(url).build();
        return callForType(request);
    }

    @Override
    Request.Builder getRequestBuilder() {
        return super.getRequestBuilder().url(getUrlBuilder().build());
    }

    HttpUrl.Builder getUrlBuilder() {
        return new HttpUrl.Builder()
                .scheme(config.getApiProtocol())
                .host(config.getApiHost())
                .port(config.getApiPort())
                .addPathSegment("arvados")
                .addPathSegment("v1")
                .addPathSegment(getResource());
    }

    <TL> TL call(Request request, Class<TL> cls) {
        String bodyAsString = newCall(request);
        try {
            return mapToObject(bodyAsString, cls);
        } catch (IOException e) {
            throw new ArvadosApiException("A problem occurred while parsing JSON data", e);
        }
    }

    private <TL> TL mapToObject(String content, Class<TL> cls) throws IOException {
        return MAPPER.readValue(content, cls);
    }

    protected  <TL> String mapToJson(TL type) {
        ObjectWriter writer = MAPPER.writer().withDefaultPrettyPrinter();
        try {
            return writer.writeValueAsString(type);
        } catch (JsonProcessingException e) {
            log.error(e.getMessage());
            return null;
        }
    }

    T callForType(Request request) {
        return call(request, getType());
    }

    L callForList(Request request) {
        return call(request, getListType());
    }

    abstract String getResource();

    abstract Class<T> getType();

    abstract Class<L> getListType();
    
    Request getNoArgumentMethodRequest(String method) {
        HttpUrl url = getUrlBuilder().addPathSegment(method).build();
        return getRequestBuilder().get().url(url).build();
    }
    
    RequestBody getJsonRequestBody(Object object) {
        return RequestBody.create(JSON, mapToJson(object));
    }
    
    void addQueryParameters(Builder urlBuilder, Object object) {
        Map<String, Object> queryMap = MAPPER.convertValue(object, new TypeReference<Map<String, Object>>() {});
        queryMap.keySet().forEach(key -> {
            Object type = queryMap.get(key);
            if (!(type instanceof String)) {
                type = mapToJson(type);
            }
            urlBuilder.addQueryParameter(key, (String) type);
        });
    }
}
