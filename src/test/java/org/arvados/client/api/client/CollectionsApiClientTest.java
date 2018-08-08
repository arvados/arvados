/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.api.client;

import okhttp3.mockwebserver.RecordedRequest;
import org.arvados.client.api.model.Collection;
import org.arvados.client.api.model.CollectionList;
import org.arvados.client.test.utils.RequestMethod;
import org.arvados.client.test.utils.ArvadosClientMockedWebServerTest;
import org.junit.Test;

import static org.arvados.client.test.utils.ApiClientTestUtils.*;
import static org.assertj.core.api.Assertions.assertThat;

public class CollectionsApiClientTest extends ArvadosClientMockedWebServerTest {

    private static final String RESOURCE = "collections";

    private CollectionsApiClient client = new CollectionsApiClient(CONFIG);

    @Test
    public void listCollections() throws Exception {

        // given
        server.enqueue(getResponse("collections-list"));

        // when
        CollectionList actual = client.list();

        // then
        RecordedRequest request = server.takeRequest();
        assertAuthorizationHeader(request);
        assertRequestPath(request, RESOURCE);
        assertRequestMethod(request, RequestMethod.GET);
        assertThat(actual.getItemsAvailable()).isEqualTo(41);
    }

    @Test
    public void getCollection() throws Exception {

        // given
        server.enqueue(getResponse("collections-get"));

        String uuid = "112ci-4zz18-p51w7z3fpopo6sm";

        // when
        Collection actual = client.get(uuid);

        // then
        RecordedRequest request = server.takeRequest();
        assertAuthorizationHeader(request);
        assertRequestPath(request, RESOURCE + "/" + uuid);
        assertRequestMethod(request, RequestMethod.GET);
        assertThat(actual.getUuid()).isEqualTo(uuid);
        assertThat(actual.getPortableDataHash()).isEqualTo("6c4106229b08fe25f48b3a7a8289dd46+143");
    }

    @Test
    public void createCollection() throws Exception {

        // given
        server.enqueue(getResponse("collections-create-simple"));

        String name = "Super Collection";
        
        Collection collection = new Collection();
        collection.setName(name);

        // when
        Collection actual = client.create(collection);

        // then
        RecordedRequest request = server.takeRequest();
        assertAuthorizationHeader(request);
        assertRequestPath(request, RESOURCE);
        assertRequestMethod(request, RequestMethod.POST);
        assertThat(actual.getName()).isEqualTo(name);
        assertThat(actual.getPortableDataHash()).isEqualTo("d41d8cd98f00b204e9800998ecf8427e+0");
        assertThat(actual.getManifestText()).isEmpty();
    }

    @Test
    public void createCollectionWithManifest() throws Exception {

        // given
        server.enqueue(getResponse("collections-create-manifest"));

        String name = "Super Collection";
        String manifestText = ". 7df44272090cee6c0732382bba415ee9+70+Aa5ece4560e3329315165b36c239b8ab79c888f8a@5a1d5708 0:70:README.md\n";
        
        Collection collection = new Collection();
        collection.setName(name);
        collection.setManifestText(manifestText);

        // when
        Collection actual = client.create(collection);

        // then
        RecordedRequest request = server.takeRequest();
        assertAuthorizationHeader(request);
        assertRequestPath(request, RESOURCE);
        assertRequestMethod(request, RequestMethod.POST);
        assertThat(actual.getName()).isEqualTo(name);
        assertThat(actual.getPortableDataHash()).isEqualTo("d41d8cd98f00b204e9800998ecf8427e+0");
        assertThat(actual.getManifestText()).isEqualTo(manifestText);
    }
}
