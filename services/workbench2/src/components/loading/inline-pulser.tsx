// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { ThreeDots } from './three-dots';
import withTheme from '@mui/styles/withTheme';
import { ArvadosTheme } from 'common/custom-theme';

type ThemeProps = {
    theme: ArvadosTheme;
};

type Props = {
    color?: string;
    height?: number;
    width?: number;
    radius?: number;
};

export const InlinePulser = withTheme((props: Props & ThemeProps) => (
    <ThreeDots
        visible={true}
        height={props.height || "30"}
        width={props.width || "30"}
        color={props.color || props.theme.customs.colors.greyL}
        radius={props.radius || "10"}
        ariaLabel="three-dots-loading"
    />
));
