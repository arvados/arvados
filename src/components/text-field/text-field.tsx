// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { WrappedFieldProps } from 'redux-form';
import { ArvadosTheme } from '~/common/custom-theme';
import { TextField as MaterialTextField, StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core';
import RichTextEditor from 'react-rte';

type CssRules = 'textField';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    textField: {
        marginBottom: theme.spacing.unit * 3
    },
});

type TextFieldProps = WrappedFieldProps & WithStyles<CssRules>;

export const TextField = withStyles(styles)((props: TextFieldProps & { 
    label?: string, autoFocus?: boolean, required?: boolean, select?: boolean, children: React.ReactNode
}) =>
    <MaterialTextField
        helperText={props.meta.touched && props.meta.error}
        className={props.classes.textField}
        label={props.label}
        disabled={props.meta.submitting}
        error={props.meta.touched && !!props.meta.error}
        autoComplete='off'
        autoFocus={props.autoFocus}
        fullWidth={true}
        required={props.required}
        select={props.select}
        children={props.children}
        {...props.input}
    />);


interface RichEditorTextFieldData {
    label?: string;
}

type RichEditorTextFieldProps = RichEditorTextFieldData & TextFieldProps;

export const RichEditorTextField = withStyles(styles)(
    class RichEditorTextField extends React.Component<RichEditorTextFieldProps> {
        state = {
            value: RichTextEditor.createValueFromString(this.props.input.value, 'html')
        };

        onChange = (value: any) => {
            this.setState({ value });
            this.props.input.onChange(value.toString('html'));
        }

        render() {
            return <RichTextEditor
                value={this.state.value}
                onChange={this.onChange}
                placeholder={this.props.label} />;
        }
    }
);

export const DateTextField = withStyles(styles)
    ((props: TextFieldProps) =>
        <MaterialTextField
            type="date"
            disabled={props.meta.submitting}
            helperText={props.meta.error}
            error={!!props.meta.error}
            fullWidth={true}
            InputLabelProps={{
                shrink: true
            }}
            name={props.input.name}
            onChange={props.input.onChange}
            value={props.input.value}
        />
    );