// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { fieldRequire } from './require';
import { maxLength } from './max-length';
import { isRsaKey } from './is-rsa-key';
import { isRemoteHost } from "./is-remote-host";
import { validFilePath, validName, validNameAllowSlash } from "./valid-name";

export const TAG_KEY_VALIDATION = [maxLength(255)];
export const TAG_VALUE_VALIDATION = [maxLength(255)];

export const PROJECT_NAME_VALIDATION = [fieldRequire, validName, maxLength(255)];
export const PROJECT_NAME_VALIDATION_ALLOW_SLASH = [fieldRequire, validNameAllowSlash, maxLength(255)];

export const COLLECTION_NAME_VALIDATION = [fieldRequire, validName, maxLength(255)];
export const COLLECTION_NAME_VALIDATION_ALLOW_SLASH = [fieldRequire, validNameAllowSlash, maxLength(255)];
export const COLLECTION_DESCRIPTION_VALIDATION = [maxLength(255)];
export const COLLECTION_PROJECT_VALIDATION = [fieldRequire];

export const COPY_NAME_VALIDATION = [fieldRequire, maxLength(255)];
export const COPY_FILE_VALIDATION = [fieldRequire];
export const RENAME_FILE_VALIDATION = [fieldRequire, validFilePath];

export const MOVE_TO_VALIDATION = [fieldRequire];

export const PROCESS_NAME_VALIDATION = [fieldRequire, maxLength(255)];
export const PROCESS_DESCRIPTION_VALIDATION = [maxLength(255)];

export const REPOSITORY_NAME_VALIDATION = [fieldRequire, maxLength(255)];

export const USER_EMAIL_VALIDATION = [fieldRequire, maxLength(255)];
export const PROFILE_EMAIL_VALIDATION = [maxLength(255)];
export const PROFILE_URL_VALIDATION = [maxLength(255)];
export const USER_LENGTH_VALIDATION = [maxLength(255)];

export const SSH_KEY_PUBLIC_VALIDATION = [fieldRequire, isRsaKey, maxLength(1024)];
export const SSH_KEY_NAME_VALIDATION = [fieldRequire, maxLength(255)];

export const SITE_MANAGER_REMOTE_HOST_VALIDATION = [fieldRequire, isRemoteHost, maxLength(255)];

export const MY_ACCOUNT_VALIDATION = [fieldRequire];

export const CHOOSE_VM_VALIDATION = [fieldRequire];
