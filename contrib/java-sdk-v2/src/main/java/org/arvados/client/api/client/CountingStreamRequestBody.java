/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.api.client;

import okio.BufferedSink;
import okio.Okio;
import okio.Source;

import java.io.File;
import java.io.IOException;
import java.io.InputStream;

public class CountingStreamRequestBody extends CountingRequestBody<InputStream> {

    CountingStreamRequestBody(final InputStream inputStream, final ProgressListener listener) {
        super(inputStream, listener);
    }

    @Override
    public long contentLength() throws IOException {
        return requestBodyData.available();
    }

    @Override
    public void writeTo(BufferedSink sink) {
        try (Source source = Okio.source(requestBodyData)) {
            long total = 0;
            long read;

            while ((read = source.read(sink.buffer(), SEGMENT_SIZE)) != -1) {
                total += read;
                sink.flush();
                listener.updateProgress(total);

            }
        } catch (RuntimeException rethrown) {
            throw rethrown;
        } catch (Exception ignored) {
            //ignore
        }
    }
}