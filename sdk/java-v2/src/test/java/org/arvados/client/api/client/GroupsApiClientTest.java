/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.api.client;

import com.google.common.collect.Lists;
import okhttp3.mockwebserver.RecordedRequest;
import org.arvados.client.api.model.Group;
import org.arvados.client.api.model.GroupList;
import org.arvados.client.api.model.argument.Filter;
import org.arvados.client.api.model.argument.ListArgument;
import org.arvados.client.test.utils.RequestMethod;
import org.arvados.client.test.utils.ArvadosClientMockedWebServerTest;
import org.junit.Test;

import java.util.Arrays;

import static java.util.Arrays.asList;
import static java.util.UUID.randomUUID;
import static org.arvados.client.test.utils.ApiClientTestUtils.*;
import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertNotNull;
import static org.junit.Assert.assertNull;

public class GroupsApiClientTest extends ArvadosClientMockedWebServerTest {
    private static final String RESOURCE = "groups";
    private GroupsApiClient client = new GroupsApiClient(CONFIG);

    @Test
    public void listGroups() throws Exception {

        // given
        server.enqueue(getResponse("groups-list"));

        // when
        GroupList actual = client.list();

        // then
        RecordedRequest request = server.takeRequest();
        assertAuthorizationHeader(request);
        assertRequestPath(request, RESOURCE);
        assertRequestMethod(request, RequestMethod.GET);
        assertEquals(20, actual.getItems().size());
    }

    @Test
    public void listProjectsByOwner() throws Exception {

        // given
        server.enqueue(getResponse("groups-list"));
        String ownerUuid = "ardev-tpzed-n3kzq4fvoks3uw4";
        String filterSubPath = "?filters=[%20[%20%22owner_uuid%22,%20%22like%22,%20%22ardev-tpzed-n3kzq4fvoks3uw4%22%20],%20" +
                "[%20%22group_class%22,%20%22in%22,%20[%20%22project%22,%20%22sub-project%22%20]%20]%20]";

        // when
        ListArgument listArgument = ListArgument.builder()
                .filters(Arrays.asList(
                        Filter.of("owner_uuid", Filter.Operator.LIKE, ownerUuid),
                        Filter.of("group_class", Filter.Operator.IN, Lists.newArrayList("project", "sub-project")
                        )))
                .build();
        GroupList actual = client.list(listArgument);

        // then
        RecordedRequest request = server.takeRequest();
        assertAuthorizationHeader(request);
        assertRequestPath(request, RESOURCE + filterSubPath);
        assertRequestMethod(request, RequestMethod.GET);
        assertEquals(20, actual.getItems().size());
    }

    @Test
    public void getGroup() throws Exception {

        // given
        server.enqueue(getResponse("groups-get"));

        String uuid = "ardev-j7d0g-bmg3pfqtx3ivczp";

        // when
        Group actual = client.get(uuid);

        // then
        RecordedRequest request = server.takeRequest();
        assertAuthorizationHeader(request);
        assertRequestPath(request, RESOURCE + "/" + uuid);
        assertRequestMethod(request, RequestMethod.GET);
        assertEquals(uuid, actual.getUuid());
        assertEquals("3hw0vk4mbl0ofvia5k6x4dwrx", actual.getEtag());
        assertEquals("ardev-tpzed-n3kzq4fvoks3uw4", actual.getOwnerUuid());
        assertEquals("TestGroup1", actual.getName());
        assertEquals("project", actual.getGroupClass());

    }

    @Test
    public void shouldClearWritableByPropertyBeforeUpdating() throws Exception {
        // given
        server.enqueue(getResponse("groups-get"));
        Group group = new Group();
        group.setUuid(randomUUID().toString());
        group.setWritableBy(asList(randomUUID().toString(), randomUUID().toString()));

        // when
        Group updatedGroup = client.update(group);

        // then
        assertNull(group.getWritableBy());
        assertNotNull(updatedGroup.getWritableBy());
    }
}
