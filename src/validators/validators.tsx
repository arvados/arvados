// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { require } from './require';
import { maxLength } from './max-length';

export const TAG_KEY_VALIDATION = [require, maxLength(255)];
export const TAG_VALUE_VALIDATION = [require, maxLength(255)];

export const PROJECT_NAME_VALIDATION = [require, maxLength(255)];
export const PROJECT_DESCRIPTION_VALIDATION = [maxLength(255)];

export const COLLECTION_NAME_VALIDATION = [require, maxLength(255)];
export const COLLECTION_DESCRIPTION_VALIDATION = [maxLength(255)];
export const COLLECTION_PROJECT_VALIDATION = [require];

export const COPY_NAME_VALIDATION = [require, maxLength(255)];
export const MAKE_A_COPY_VALIDATION = [require, maxLength(255)];

export const MOVE_TO_VALIDATION = [require];
