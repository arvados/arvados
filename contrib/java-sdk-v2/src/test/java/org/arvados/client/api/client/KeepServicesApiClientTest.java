/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.api.client;

import okhttp3.mockwebserver.RecordedRequest;
import org.arvados.client.api.model.KeepService;
import org.arvados.client.api.model.KeepServiceList;
import org.arvados.client.test.utils.ArvadosClientMockedWebServerTest;
import org.junit.Test;

import static org.arvados.client.test.utils.ApiClientTestUtils.*;
import static org.assertj.core.api.Assertions.assertThat;

public class KeepServicesApiClientTest extends ArvadosClientMockedWebServerTest {

    private static final String RESOURCE = "keep_services";

    private KeepServicesApiClient client = new KeepServicesApiClient(CONFIG);

    @Test
    public void listKeepServices() throws Exception {

        // given
        server.enqueue(getResponse("keep-services-list"));

        // when
        KeepServiceList actual = client.list();

        // then
        RecordedRequest request = server.takeRequest();
        assertAuthorizationHeader(request);
        assertRequestPath(request, RESOURCE);

        assertThat(actual.getItemsAvailable()).isEqualTo(3);

    }

    @Test
    public void listAccessibleKeepServices() throws Exception {

        // given
        server.enqueue(getResponse("keep-services-accessible"));

        // when
        KeepServiceList actual = client.accessible();

        // then
        RecordedRequest request = server.takeRequest();
        assertAuthorizationHeader(request);
        assertRequestPath(request, RESOURCE + "/accessible");
        assertThat(actual.getItemsAvailable()).isEqualTo(2);
    }

    @Test
    public void getKeepService() throws Exception {

        // given
        server.enqueue(getResponse("keep-services-get"));

        String uuid = "112ci-bi6l4-hv02fg8sbti8ykk";

        // whenFs
        KeepService actual = client.get(uuid);

        // then
        RecordedRequest request = server.takeRequest();
        assertAuthorizationHeader(request);
        assertRequestPath(request, RESOURCE + "/" + uuid);
        assertThat(actual.getUuid()).isEqualTo(uuid);
        assertThat(actual.getServiceType()).isEqualTo("disk");
    }

}
