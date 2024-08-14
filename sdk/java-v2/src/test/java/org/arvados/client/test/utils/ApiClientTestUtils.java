/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.test.utils;

import org.arvados.client.config.FileConfigProvider;
import okhttp3.mockwebserver.MockResponse;
import okhttp3.mockwebserver.RecordedRequest;
import org.apache.commons.io.FileUtils;

import java.io.File;
import java.io.IOException;
import java.nio.charset.Charset;

import static org.assertj.core.api.Assertions.assertThat;

public final class ApiClientTestUtils {

    static final String BASE_URL = "/arvados/v1/";

    private ApiClientTestUtils() {}

    public static MockResponse getResponse(String filename) throws IOException {
        String filePath = String.format("src/test/resources/org/arvados/client/api/client/%s.json", filename);
        File jsonFile = new File(filePath);
        String json = FileUtils.readFileToString(jsonFile, Charset.defaultCharset());
        return new MockResponse().setBody(json);
    }

    public static void assertAuthorizationHeader(RecordedRequest request) {
        assertThat(request.getHeader("authorization")).isEqualTo("Bearer " + new FileConfigProvider().getApiToken());
    }

    public static void assertRequestPath(RecordedRequest request, String subPath) {
        assertThat(request.getPath()).isEqualTo(BASE_URL + subPath);
    }

    public static void assertRequestMethod(RecordedRequest request, RequestMethod requestMethod) {
        assertThat(request.getMethod()).isEqualTo(requestMethod.name());
    }
}
