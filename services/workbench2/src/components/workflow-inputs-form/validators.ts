// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { CommandInputParameter } from 'models/workflow';
import { require } from 'validators/require';
import { CWLType } from '../../models/workflow';


const alwaysValid = () => undefined;

export const required = ({ type }: CommandInputParameter) => {
    if (type instanceof Array) {
        for (const t of type) {
            if (t === CWLType.NULL) {
                return alwaysValid;
            }
        }
    }
    return require;
};
