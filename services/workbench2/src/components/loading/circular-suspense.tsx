// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { CircularProgress } from '@mui/material';

type CircularSuspenseProps = {
    element: React.ReactNode;
    showElement: boolean;
}

export const CircularSuspense: React.FC<CircularSuspenseProps> = ({ element, showElement }) => {
    return showElement
        ? <>{element}</>
        : <CircularProgress />;
};