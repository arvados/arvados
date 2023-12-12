/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.api.client;

import okhttp3.MediaType;
import okhttp3.RequestBody;
import org.slf4j.Logger;

abstract class CountingRequestBody<T> extends RequestBody {

    protected static final int SEGMENT_SIZE = 2048; // okio.Segment.SIZE
    protected static final MediaType CONTENT_BINARY = MediaType.parse(com.google.common.net.MediaType.OCTET_STREAM.toString());

    protected final ProgressListener listener;

    protected final T requestBodyData;

    CountingRequestBody(T file, final ProgressListener listener) {
        this.requestBodyData = file;
        this.listener = listener;
    }

    @Override
    public MediaType contentType() {
        return CONTENT_BINARY;
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