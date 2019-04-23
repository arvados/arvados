/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.exception;

public class ArvadosApiException extends ArvadosClientException {

    private static final long serialVersionUID = 1L;

    public ArvadosApiException(String message) {
        super(message);
    }
    
    public ArvadosApiException(String message, Throwable cause) {
        super(message, cause);
    }
    
    public ArvadosApiException(Throwable cause) {
        super(cause);
    }
}
