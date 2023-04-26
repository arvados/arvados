// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export default function arraysAreCongruent<T>(arr1: T[], arr2: T[]): boolean {
    if (!arr1.length || !arr2.length) return false;
    if (arr1.length !== arr2.length) return false;

    const sortedArr1 = [...arr1].sort();
    const sortedArr2 = [...arr2].sort();

    for (let i = 0; i < sortedArr1.length; i++) {
        if (sortedArr1[i] !== sortedArr2[i]) return false;
    }

    return true;
}
