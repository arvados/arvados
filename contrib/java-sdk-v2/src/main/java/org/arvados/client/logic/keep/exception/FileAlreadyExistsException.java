/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.logic.keep.exception;

import org.arvados.client.exception.ArvadosClientException;

/**
 * Signals that an attempt to download a file with given name has failed for a specified
 * download location.
 *
 * <p> This exception will be thrown during an attempt to download single file to a location
 * that already contains file with given name</p>
 */
public class FileAlreadyExistsException extends ArvadosClientException {

    public FileAlreadyExistsException(String message) { super(message); }

}
