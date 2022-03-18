// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
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
    containerClassName?: string;
    fontSize?: number;
}

type CodeSnippetProps = CodeSnippetDataProps & WithStyles<CssRules>;

export const CodeSnippet = withStyles(styles)(
    ({ classes, lines, className, containerClassName, apiResponse, fontSize }: CodeSnippetProps) =>
        <Typography
            component="div"
            className={classNames(classes.root, containerClassName, className)}>
                { lines.map((line: string, index: number) => {
                    return <Typography key={index} style={{ fontSize: fontSize }} className={apiResponse ? classes.space : className} component="pre">{line}</Typography>;
                }) }
        </Typography>
    );