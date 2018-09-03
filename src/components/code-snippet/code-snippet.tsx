// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { StyleRulesCallback, WithStyles, Typography, withStyles, Theme } from '@material-ui/core';
import { ArvadosTheme } from '~/common/custom-theme';

type CssRules = 'root';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        boxSizing: 'border-box',
        width: '100%',
        height: 'auto',
        maxHeight: '550px',
        overflow: 'scroll',
        padding: theme.spacing.unit
    }
});

export interface CodeSnippetDataProps {
    lines: string[];
}

type CodeSnippetProps = CodeSnippetDataProps & WithStyles<CssRules>;

export const CodeSnippet = withStyles(styles)(
    ({ classes, lines }: CodeSnippetProps) =>
        <Typography component="div" className={classes.root}>
            {
                lines.map((line: string, index: number) => {
                    return <Typography key={index} component="div">{line}</Typography>;
                })
            }
        </Typography>
    );