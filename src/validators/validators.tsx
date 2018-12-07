// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { require } from './require';
import { maxLength } from './max-length';
import { isRsaKey } from './is-rsa-key';

export const TAG_KEY_VALIDATION = [require, maxLength(255)];
export const TAG_VALUE_VALIDATION = [require, maxLength(255)];

export const PROJECT_NAME_VALIDATION = [require, maxLength(255)];

export const COLLECTION_NAME_VALIDATION = [require, maxLength(255)];
export const COLLECTION_DESCRIPTION_VALIDATION = [maxLength(255)];
export const COLLECTION_PROJECT_VALIDATION = [require];

export const COPY_NAME_VALIDATION = [require, maxLength(255)];
export const COPY_FILE_VALIDATION = [require];

export const MOVE_TO_VALIDATION = [require];

export const PROCESS_NAME_VALIDATION = [require, maxLength(255)];

export const REPOSITORY_NAME_VALIDATION = [require, maxLength(255)];

export const USER_EMAIL_VALIDATION = [require, maxLength(255)];
export const USER_LENGTH_VALIDATION = [maxLength(255)];

export const SSH_KEY_PUBLIC_VALIDATION = [require, isRsaKey, maxLength(1024)];
export const SSH_KEY_NAME_VALIDATION = [require, maxLength(255)];

export const MY_ACCOUNT_VALIDATION = [require];
