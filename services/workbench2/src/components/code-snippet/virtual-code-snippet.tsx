// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { StyleRulesCallback, WithStyles, Typography, withStyles } from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';
import classNames from 'classnames';
import { connect, DispatchProp } from 'react-redux';
import { RootState } from 'store/store';
import { FederationConfig } from 'routes/routes';
import { renderLinks } from './code-snippet';
import { FixedSizeList } from 'react-window';
import AutoSizer from "react-virtualized-auto-sizer";

type CssRules = 'root' | 'space' | 'content' ;

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        boxSizing: 'border-box',
        height: '100%',
        padding: theme.spacing.unit,
    },
    space: {
        marginLeft: '15px',
    },
    content: {
        maxHeight: '100%',
        height: '100vh',
    },
});

export interface CodeSnippetDataProps {
    lines: string[];
    lineTransformer?: (line: string) => string;
    className?: string;
    apiResponse?: boolean;
    linked?: boolean;
}

interface CodeSnippetAuthProps {
    auth: FederationConfig;
}

type CodeSnippetProps = CodeSnippetDataProps & WithStyles<CssRules>;

const mapStateToProps = (state: RootState): CodeSnippetAuthProps => ({
    auth: state.auth,
});

export const VirtualCodeSnippet = withStyles(styles)(connect(mapStateToProps)(
    ({ classes, lines, linked, className, apiResponse, dispatch, auth }: CodeSnippetProps & CodeSnippetAuthProps & DispatchProp) => {
        const RenderRow = ({index, style}) => (
            <span style={style}>{linked ? renderLinks(auth, dispatch)(lines[index]) : lines[index]}</span>
        );

        return <Typography
            component="div"
            className={classNames([classes.root, className])}>
            <Typography className={classNames(classes.content, apiResponse ? classes.space : className)} component="pre">
                <AutoSizer>
                    {({ height, width }) =>
                        <FixedSizeList
                            height={height}
                            width={width}
                            itemSize={21}
                            itemCount={lines.length}
                        >
                            {RenderRow}
                        </FixedSizeList>
                    }
                </AutoSizer>
            </Typography>
        </Typography>;
}));
