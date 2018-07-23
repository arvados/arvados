// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import require from '../require';
import maxLength from '../max-length';

export const NAME = [require, maxLength(255)];
export const DESCRIPTION = [maxLength(255)];
