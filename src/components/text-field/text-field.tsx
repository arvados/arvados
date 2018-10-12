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

export const TextField = withStyles(styles)((props: WrappedFieldProps & WithStyles<CssRules> & { label?: string, autoFocus?: boolean, required?: boolean }) =>
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
        {...props.input}
    />);


interface RichEditorTextFieldData {
    label?: string;
}

type RichEditorTextFieldProps = RichEditorTextFieldData & WrappedFieldProps & WithStyles<CssRules>;

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

type DateTextFieldProps = WrappedFieldProps & WithStyles<CssRules>;

export const DateTextField = withStyles(styles)
    ((props: DateTextFieldProps) => 
        <MaterialTextField
            disabled={props.meta.submitting}
            error={props.meta.touched && !!props.meta.error}
            type="date"
            fullWidth={true}
            name={props.input.name}
            InputLabelProps={{
                shrink: true
            }}
            onChange={props.input.onChange}
            value={props.input.value}
        />    
    );