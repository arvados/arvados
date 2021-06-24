// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { StyleRulesCallback, WithStyles, Typography, withStyles } from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';
import classNames from 'classnames';

type CssRules = 'root' | 'space';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        boxSizing: 'border-box',
        overflow: 'auto',
        padding: theme.spacing.unit
    },
    space: {
        marginLeft: '15px'
    }
});

export interface CodeSnippetDataProps {
    lines: string[];
    className?: string;
    apiResponse?: boolean;
}

type CodeSnippetProps = CodeSnippetDataProps & WithStyles<CssRules>;

export const CodeSnippet = withStyles(styles)(
    ({ classes, lines, className, apiResponse }: CodeSnippetProps) =>
        <Typography
        component="div"
        className={classNames(classes.root, className)}>
            {
                lines.map((line: string, index: number) => {
                    return <Typography key={index} className={apiResponse ? classes.space : className} component="pre">{line}</Typography>;
                })
            }
        </Typography>
    );