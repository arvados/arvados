// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Dispatch } from 'redux';
import { connect } from 'react-redux';
import { withStyles, StyleRulesCallback, WithStyles, Typography } from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';
import { openRichTextEditorDialog } from 'store/rich-text-editor-dialog/rich-text-editor-dialog-actions';

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

interface RichTextEditorLinkActions {
    onClick: (title: string, content: string) => void;
}

type RichTextEditorLinkProps = RichTextEditorLinkData & RichTextEditorLinkActions & WithStyles<CssRules>;

const mapDispatchToProps = (dispatch: Dispatch) => ({
    onClick: (title: string, content: string) => dispatch<any>(openRichTextEditorDialog(title, content))
});

export const RichTextEditorLink = connect(undefined, mapDispatchToProps)(
    withStyles(styles)(({ classes, title, content, label, onClick }: RichTextEditorLinkProps) =>
        <Typography component='span' className={classes.root}
            onClick={() => onClick(title, content) }>
            {label}
        </Typography>
    ));