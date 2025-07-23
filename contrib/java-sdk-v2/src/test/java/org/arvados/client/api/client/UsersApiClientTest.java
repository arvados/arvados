/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.api.client;

import okhttp3.mockwebserver.RecordedRequest;
import org.arvados.client.api.model.User;
import org.arvados.client.api.model.UserList;
import org.arvados.client.test.utils.RequestMethod;
import org.arvados.client.test.utils.ArvadosClientMockedWebServerTest;
import org.junit.Test;

import static org.arvados.client.common.Characters.SLASH;
import static org.arvados.client.test.utils.ApiClientTestUtils.*;
import static org.assertj.core.api.Assertions.assertThat;

public class UsersApiClientTest extends ArvadosClientMockedWebServerTest {

    private static final String RESOURCE = "users";
    private static final String USER_UUID = "ardev-tpzed-q6dvn7sby55up1b";

    private UsersApiClient client = new UsersApiClient(CONFIG);

    @Test
    public void listUsers() throws Exception {

        // given
        server.enqueue(getResponse("users-list"));

        // when
        UserList actual = client.list();

        // then
        RecordedRequest request = server.takeRequest();
        assertAuthorizationHeader(request);
        assertRequestPath(request, RESOURCE);
        assertRequestMethod(request, RequestMethod.GET);
        assertThat(actual.getItemsAvailable()).isEqualTo(13);
    }

    @Test
    public void getUser() throws Exception {

        // given
        server.enqueue(getResponse("users-get"));

        // when
        User actual = client.get(USER_UUID);

        // then
        RecordedRequest request = server.takeRequest();
        assertAuthorizationHeader(request);
        assertRequestPath(request, RESOURCE + SLASH + USER_UUID);
        assertRequestMethod(request, RequestMethod.GET);
        assertThat(actual.getUuid()).isEqualTo(USER_UUID);
    }

    @Test
    public void getCurrentUser() throws Exception {

        // given
        server.enqueue(getResponse("users-get"));

        // when
        User actual = client.current();

        // then
        RecordedRequest request = server.takeRequest();
        assertAuthorizationHeader(request);
        assertRequestPath(request, RESOURCE + SLASH + "current");
        assertRequestMethod(request, RequestMethod.GET);
        assertThat(actual.getUuid()).isEqualTo(USER_UUID);
    }

    @Test
    public void getSystemUser() throws Exception {

        // given
        server.enqueue(getResponse("users-system"));

        // when
        User actual = client.system();

        // then
        RecordedRequest request = server.takeRequest();
        assertAuthorizationHeader(request);
        assertRequestPath(request, RESOURCE + SLASH + "system");
        assertRequestMethod(request, RequestMethod.GET);
        assertThat(actual.getUuid()).isEqualTo("ardev-tpzed-000000000000000");
    }

    @Test
    public void createUser() throws Exception {

        // given
        server.enqueue(getResponse("users-create"));

        String firstName = "John";
        String lastName = "Wayne";
        String fullName = String.format("%s %s", firstName, lastName);
        String username = String.format("%s%s", firstName, lastName).toLowerCase();

        User user = new User();
        user.setFirstName(firstName);
        user.setLastName(lastName);
        user.setFullName(fullName);
        user.setUsername(username);

        // when
        User actual = client.create(user);

        // then
        RecordedRequest request = server.takeRequest();
        assertAuthorizationHeader(request);
        assertRequestPath(request, RESOURCE);
        assertRequestMethod(request, RequestMethod.POST);
        assertThat(actual.getFirstName()).isEqualTo(firstName);
        assertThat(actual.getLastName()).isEqualTo(lastName);
        assertThat(actual.getFullName()).isEqualTo(fullName);
        assertThat(actual.getUsername()).isEqualTo(username);
    }
}
