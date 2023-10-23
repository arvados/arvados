// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';

export interface PickerIdProp {
    pickerId: string;
}

export const pickerId =
    (id: string) =>
        <P extends PickerIdProp>(Component: React.ComponentType<P>) =>
            (props: P) =>
                <Component {...props} pickerId={id} />;
                