// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { DispatchProp } from 'react-redux';
import { withStyles, StyleRulesCallback, WithStyles, Typography } from '@material-ui/core';
import { ArvadosTheme } from '~/common/custom-theme';
import { openRichTextEditorDialog } from '~/store/rich-text-editor-dialog/rich-text-editor-dialog-actions';

type CssRules = "root";

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        color: theme.palette.primary.main,
        cursor: 'pointer'
    }
});

interface RichTextEditorLinkData {
    title: string;
    label: string;
    content: string;
}

type RichTextEditorLinkProps = RichTextEditorLinkData & WithStyles<CssRules>;

export const RichTextEditorLink = withStyles(styles)(
    ({ classes, title, content, label }: RichTextEditorLinkProps) =>
        <Typography component='span' className={classes.root} 
            // onClick={() => dispatch<any>(openRichTextEditorDialog(title, content))}
            >
            {label}
        </Typography>
);