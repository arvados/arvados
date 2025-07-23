/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.exception;

/**
 * Parent exception for all exceptions in library.
 * More specific exceptions like ArvadosApiException extend this class.
 */
public class ArvadosClientException extends RuntimeException {

    public ArvadosClientException(String message) {
        super(message);
    }

    public ArvadosClientException(String message, Throwable cause) {
        super(message, cause);
    }

    public ArvadosClientException(Throwable cause) {
        super(cause);
    }
}
