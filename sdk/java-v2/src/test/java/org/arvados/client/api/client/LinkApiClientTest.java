/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.api.client;

import okhttp3.mockwebserver.RecordedRequest;
import org.arvados.client.api.model.Link;
import org.arvados.client.api.model.LinkList;
import org.arvados.client.test.utils.ArvadosClientMockedWebServerTest;
import org.arvados.client.test.utils.RequestMethod;
import org.junit.Test;

import static org.arvados.client.test.utils.ApiClientTestUtils.assertAuthorizationHeader;
import static org.arvados.client.test.utils.ApiClientTestUtils.assertRequestMethod;
import static org.arvados.client.test.utils.ApiClientTestUtils.assertRequestPath;
import static org.arvados.client.test.utils.ApiClientTestUtils.getResponse;
import static org.assertj.core.api.Assertions.assertThat;
import static org.junit.Assert.assertEquals;

public class LinkApiClientTest extends ArvadosClientMockedWebServerTest {

    private static final String RESOURCE = "links";

    private final LinksApiClient client = new LinksApiClient(CONFIG);

    @Test
    public void listLinks() throws Exception {
        // given
        server.enqueue(getResponse("links-list"));

        // when
        LinkList actual = client.list();

        // then
        RecordedRequest request = server.takeRequest();
        assertAuthorizationHeader(request);
        assertRequestPath(request, RESOURCE);
        assertRequestMethod(request, RequestMethod.GET);
        assertThat(actual.getItemsAvailable()).isEqualTo(2);
    }

    @Test
    public void getLink() throws Exception {
        // given
        server.enqueue(getResponse("links-get"));

        String uuid = "arkau-o0j2j-huxuaxbi46s1yml";

        // when
        Link actual = client.get(uuid);

        // then
        RecordedRequest request = server.takeRequest();
        assertAuthorizationHeader(request);
        assertRequestPath(request, RESOURCE + "/" + uuid);
        assertRequestMethod(request, RequestMethod.GET);
        assertEquals(actual.getUuid(), uuid);
        assertEquals(actual.getName(), "can_read");
        assertEquals(actual.getHeadKind(), "arvados#group");
        assertEquals(actual.getHeadUuid(), "arkau-j7d0g-fcedae2076pw56h");
        assertEquals(actual.getTailUuid(), "ardev-tpzed-n3kzq4fvoks3uw4");
        assertEquals(actual.getTailKind(), "arvados#user");
        assertEquals(actual.getLinkClass(), "permission");
    }

    @Test
    public void createLink() throws Exception {
        // given
        server.enqueue(getResponse("links-create"));

        String name = "Star Link";

        Link collection = new Link();
        collection.setName(name);

        // when
        Link actual = client.create(collection);

        // then
        RecordedRequest request = server.takeRequest();
        assertAuthorizationHeader(request);
        assertRequestPath(request, RESOURCE);
        assertRequestMethod(request, RequestMethod.POST);
        assertThat(actual.getName()).isEqualTo(name);
        assertEquals(actual.getName(), name);
        assertEquals(actual.getUuid(), "arkau-o0j2j-huxuaxbi46s1yml");
        assertEquals(actual.getHeadKind(), "arvados#group");
        assertEquals(actual.getHeadUuid(), "arkau-j7d0g-fcedae2076pw56h");
        assertEquals(actual.getTailUuid(), "ardev-tpzed-n3kzq4fvoks3uw4");
        assertEquals(actual.getTailKind(), "arvados#user");
        assertEquals(actual.getLinkClass(), "star");
    }
}
