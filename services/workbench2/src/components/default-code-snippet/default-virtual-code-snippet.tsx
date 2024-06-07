// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { MuiThemeProvider, createMuiTheme } from '@material-ui/core/styles';
import { VirtualCodeSnippet, CodeSnippetDataProps } from 'components/code-snippet/virtual-code-snippet';
import grey from '@material-ui/core/colors/grey';
import { themeOptions } from 'common/custom-theme';

const theme = createMuiTheme(Object.assign({}, themeOptions, {
    overrides: {
        MuiTypography: {
            body1: {
                color: grey["900"]
            },
            root: {
                backgroundColor: grey["200"]
            }
        }
    },
    typography: {
        fontFamily: 'monospace',
    }
}));

export const DefaultVirtualCodeSnippet = (props: CodeSnippetDataProps) =>
    <MuiThemeProvider theme={theme}>
        <VirtualCodeSnippet {...props} />
    </MuiThemeProvider>;
