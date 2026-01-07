// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { fieldRequire } from './require';
import { maxLength } from './max-length';
import { isRsaKey } from './is-rsa-key';
import { isRemoteHost } from "./is-remote-host";
import { validFileName, validFilePath, validName, validNameAllowSlash } from "./valid-name";
import { isZipFilename } from './is-zip-filename';
import { isValidFutureDate } from './is-valid-future-date';

export type Validator = (value: string) => string | undefined;

// pass in fieldName for better debugging messages
export const getFieldErrors = (value: string, validationArray: Validator[], fieldName?: string): string[] => {
  const errMessages: string[] = [];
  for (const validation of validationArray) {
    const result = validation(value);
    const errorMsg = result ? (fieldName ? `${fieldName}: ${result}` : result) : null;
    if (errorMsg) {
      errMessages.push(errorMsg);
    }
  }
  return errMessages;
}

export const TAG_KEY_VALIDATION: Validator[] = [maxLength(255)];
export const TAG_VALUE_VALIDATION: Validator[] = [maxLength(255)];

export const PROJECT_NAME_VALIDATION: Validator[] = [fieldRequire, validName, maxLength(255)];
export const PROJECT_NAME_VALIDATION_ALLOW_SLASH: Validator[] = [fieldRequire, validNameAllowSlash, maxLength(255)];
export const PROJECT_DESCRIPTION_VALIDATION: Validator[] = [maxLength(524_288)];

export const COLLECTION_NAME_VALIDATION: Validator[] = [fieldRequire, validName, maxLength(255)];
export const COLLECTION_NAME_VALIDATION_ALLOW_SLASH: Validator[] = [fieldRequire, validNameAllowSlash, maxLength(255)];
export const COLLECTION_DESCRIPTION_VALIDATION: Validator[] = [maxLength(524_288)];
export const COLLECTION_PROJECT_VALIDATION: Validator[] = [fieldRequire];

export const COPY_NAME_VALIDATION: Validator[] = [fieldRequire, maxLength(255)];
export const COPY_FILE_VALIDATION: Validator[] = [fieldRequire];
export const RENAME_FILE_VALIDATION: Validator[] = [fieldRequire, validFilePath];
export const DOWNLOAD_ZIP_VALIDATION: Validator[] = [fieldRequire, isZipFilename, validFileName];

export const MOVE_TO_VALIDATION: Validator[] = [fieldRequire];

export const PROCESS_NAME_VALIDATION: Validator[] = [fieldRequire, maxLength(255)];
export const PROCESS_DESCRIPTION_VALIDATION: Validator[] = [maxLength(255)];

export const REPOSITORY_NAME_VALIDATION: Validator[] = [fieldRequire, maxLength(255)];

export const USER_EMAIL_VALIDATION: Validator[] = [fieldRequire, maxLength(255)];
export const PROFILE_EMAIL_VALIDATION: Validator[] = [maxLength(255)];
export const PROFILE_URL_VALIDATION: Validator[] = [maxLength(255)];
export const USER_LENGTH_VALIDATION: Validator[] = [maxLength(255)];

export const SSH_KEY_PUBLIC_VALIDATION: Validator[] = [fieldRequire, isRsaKey, maxLength(1024)];
export const SSH_KEY_NAME_VALIDATION: Validator[] = [fieldRequire, maxLength(255)];

export const SITE_MANAGER_REMOTE_HOST_VALIDATION: Validator[] = [fieldRequire, isRemoteHost, maxLength(255)];

export const MY_ACCOUNT_VALIDATION: Validator[] = [fieldRequire];
export const CHOOSE_VM_VALIDATION: Validator[] = [fieldRequire];

export const REQUIRED_VALIDATION: Validator[] = [fieldRequire];

export const LENGTH255_VALIDATION: Validator[] = [maxLength(255)];
export const REQUIRED_LENGTH255_VALIDATION: Validator[] = [fieldRequire, maxLength(255)];
export const REQUIRED_VALIDNAME_LENGTH255_VALIDATION: Validator[] = [fieldRequire, validName, maxLength(255)];
export const MAXLENGTH_524288_VALIDATION: Validator[] = [maxLength(524_288)];

export const DATE_VALIDATION: Validator[] = [isValidFutureDate];