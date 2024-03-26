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

/**
 * Based on:
 * {@link} https://gist.github.com/eduardb/dd2dc530afd37108e1ac
 */
public class CountingFileRequestBody extends CountingRequestBody<File> {

    CountingFileRequestBody(final File file, final ProgressListener listener) {
        super(file, listener);
    }

    @Override
    public long contentLength() {
        return requestBodyData.length();
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