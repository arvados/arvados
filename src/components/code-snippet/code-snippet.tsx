// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { StyleRulesCallback, WithStyles, Typography, withStyles, Theme } from '@material-ui/core';
import { ArvadosTheme } from '~/common/custom-theme';
import * as classNames from 'classnames';

type CssRules = 'root';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        boxSizing: 'border-box',
        overflow: 'auto',
        padding: theme.spacing.unit
    }
});

export interface CodeSnippetDataProps {
    lines: string[];
    className?: string;
}

type CodeSnippetProps = CodeSnippetDataProps & WithStyles<CssRules>;

export const CodeSnippet = withStyles(styles)(
    ({ classes, lines, className }: CodeSnippetProps) =>
        <Typography 
        component="div" 
        className={classNames(classes.root, className)}>
            {
                lines.map((line: string, index: number) => {
                    return <Typography key={index} component="pre">{line}</Typography>;
                })
            }
        </Typography>
    );