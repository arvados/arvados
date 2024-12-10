// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { ThemeProvider, Theme, StyledEngineProvider, createTheme } from '@mui/material/styles';
import { VirtualCodeSnippet, CodeSnippetDataProps } from 'components/code-snippet/virtual-code-snippet';
import { themeOptions } from 'common/custom-theme';
import { grey } from '@mui/material/colors';


declare module '@mui/styles/defaultTheme' {
  // eslint-disable-next-line @typescript-eslint/no-empty-interface
  interface DefaultTheme extends Theme {}
}


const theme = createTheme(Object.assign({}, themeOptions, {
    components: {
        MuiTypography: {
            styleOverrides: {
                body1: {
                    color: grey["900"]
                },
                root: {
                    backgroundColor: grey["200"]
                },
            }
        }
    },
    typography: {
        fontFamily: 'monospace',
    }
}));

export const DefaultVirtualCodeSnippet = (props: CodeSnippetDataProps) =>
    <StyledEngineProvider injectFirst>
        <ThemeProvider theme={theme}>
            <VirtualCodeSnippet {...props} />
        </ThemeProvider>
    </StyledEngineProvider>;
