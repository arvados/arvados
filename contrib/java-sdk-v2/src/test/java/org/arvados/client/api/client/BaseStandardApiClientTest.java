/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.api.client;

import okhttp3.HttpUrl;
import org.arvados.client.api.model.Item;
import org.arvados.client.api.model.ItemList;
import org.arvados.client.test.utils.ArvadosClientUnitTest;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.mockito.Spy;
import org.mockito.junit.MockitoJUnitRunner;

import static org.assertj.core.api.Assertions.assertThat;

@RunWith(MockitoJUnitRunner.class)
public class BaseStandardApiClientTest extends ArvadosClientUnitTest {

    @Spy
    private BaseStandardApiClient<?, ?> client = new BaseStandardApiClient<Item, ItemList>(CONFIG) {
        @Override
        String getResource() {
            return "resource";
        }

        @Override
        Class<Item> getType() {
            return null;
        }

        @Override
        Class<ItemList> getListType() {
            return null;
        }
    };

    @Test
    public void urlBuilderBuildsExpectedUrlFormat() {
        // when
        HttpUrl.Builder actual = client.getUrlBuilder();

        // then
        assertThat(actual.build().toString()).isEqualTo("http://localhost:9000/arvados/v1/resource");
    }
}
