/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.api.client;

import okhttp3.MediaType;
import okhttp3.RequestBody;
import okio.BufferedSink;
import okio.Okio;
import okio.Source;
import org.slf4j.Logger;

import java.io.File;

/**
 * Based on:
 * {@link} https://gist.github.com/eduardb/dd2dc530afd37108e1ac
 */
public class CountingFileRequestBody extends RequestBody {

    private static final int SEGMENT_SIZE = 2048; // okio.Segment.SIZE
    private static final MediaType CONTENT_BINARY = MediaType.parse(com.google.common.net.MediaType.OCTET_STREAM.toString());

    private final File file;
    private final ProgressListener listener;

    CountingFileRequestBody(final File file, final ProgressListener listener) {
        this.file = file;
        this.listener = listener;
    }

    @Override
    public long contentLength() {
        return file.length();
    }

    @Override
    public MediaType contentType() {
        return CONTENT_BINARY;
    }

    @Override
    public void writeTo(BufferedSink sink) {
        try (Source source = Okio.source(file)) {
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

    static class TransferData {

        private final Logger log = org.slf4j.LoggerFactory.getLogger(TransferData.class);
        private int progressValue;
        private long totalSize;

        TransferData(long totalSize) {
            this.progressValue = 0;
            this.totalSize = totalSize;
        }

        void updateTransferProgress(long transferred) {
            float progress = (transferred / (float) totalSize) * 100;
            if (progressValue != (int) progress) {
                progressValue = (int) progress;
                log.debug("{} / {} / {}%", transferred, totalSize, progressValue);
            }
        }
    }
}