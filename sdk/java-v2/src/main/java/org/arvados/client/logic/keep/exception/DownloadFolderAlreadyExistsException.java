/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.logic.keep.exception;

import org.arvados.client.exception.ArvadosClientException;

/**
 * Exception indicating that directory with given name was already created in specified location.
 *
 * <p> This exception will be thrown during an attempt to download all files from certain
 * collection to a location that already contains folder named by this collection's UUID.</p>
 */
public class DownloadFolderAlreadyExistsException extends ArvadosClientException {

    public DownloadFolderAlreadyExistsException(String message) {
        super(message);
    }

}
