// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { MuiThemeProvider, createMuiTheme, StyleRulesCallback, withStyles, WithStyles } from '@material-ui/core/styles';
import { CodeSnippet } from '~/components/code-snippet/code-snippet';
import grey from '@material-ui/core/colors/grey';

type CssRules = 'codeSnippet';

const styles: StyleRulesCallback<CssRules> = () => ({
    codeSnippet: {
        maxHeight: '550px',
    }
});

const theme = createMuiTheme({
    overrides: {
        MuiTypography: {
            body1: {
                color: grey["200"]
            },
            root: {
                backgroundColor: '#000'
            }
        }
    },
    typography: {
        fontFamily: 'monospace',
        useNextVariants: true,
    }
});

interface ProcessLogCodeSnippetProps {
    lines: string[];
}

export const ProcessLogCodeSnippet = withStyles(styles)(
    (props: ProcessLogCodeSnippetProps & WithStyles<CssRules>) =>
        <MuiThemeProvider theme={theme}>
            <CodeSnippet lines={props.lines} className={props.classes.codeSnippet} />
        </MuiThemeProvider>);