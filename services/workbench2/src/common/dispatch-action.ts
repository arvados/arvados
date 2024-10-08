// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { store } from '../index';

export function dispatchAction<T extends (...args: any[]) => any>(callback: T, ...args: Parameters<T>): ReturnType<T> {
    const dispatch = store.dispatch as Dispatch<any>;
    return dispatch<any>(callback(...args));
}
