// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useEffect, useState } from 'react';
import { WrappedFieldProps } from 'redux-form';
import { Typography, FormControl } from '@mui/material';
import { ArvadosTheme } from 'common/custom-theme';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { TextField as MaterialTextField, FormControlOwnProps } from '@mui/material';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import RichTextEditor from 'react-rte';
import classNames from 'classnames';

type CssRules = 'textField' | 'rte' | 'errorMessage' | 'redBorder';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    textField: {
        marginBottom: theme.spacing(1)
    },
    rte: {
        fontFamily: 'Arial',
        '& a': {
            textDecoration: 'none',
            color: theme.palette.primary.main,
            '&:hover': {
                cursor: 'pointer',
                textDecoration: 'underline'
            }
        }
    },
    errorMessage: {
        color: theme.palette.error.main,
        fontSize: '0.75rem',
        marginTop: '0.25rem',
    },
    redBorder: {
        border: `1px solid ${theme.palette.error.main}`,
    },
});

type TextFieldProps = WrappedFieldProps & WithStyles<CssRules>;

export const TextField = withStyles(styles)((props: TextFieldProps & {
    label?: string, autoFocus?: boolean, required?: boolean, select?: boolean, disabled?: boolean, children: React.ReactNode, margin?: FormControlOwnProps["margin"], placeholder?: string,
    helperText?: string, type?: string, autoComplete?: string,
}) =>
    <MaterialTextField
        variant="standard"
        helperText={(props.meta.touched && props.meta.error) || props.helperText}
        className={props.classes.textField}
        label={props.label}
        disabled={props.disabled || props.meta.submitting}
        error={props.meta.touched && !!props.meta.error}
        autoComplete={props.autoComplete || 'off'}
        autoFocus={props.autoFocus}
        fullWidth={true}
        required={props.required}
        select={props.select}
        children={props.children}
        margin={props.margin}
        placeholder={props.placeholder}
        type={props.type}
        {...props.input} />);


interface RichEditorTextFieldData {
    label?: string;
}

type RichEditorTextFieldProps = RichEditorTextFieldData & TextFieldProps;

export const RichEditorTextField = withStyles(styles)(
    class RichEditorTextField extends React.Component<RichEditorTextFieldProps> {
        state = {
            value: RichTextEditor.createValueFromString(this.props.input.value, 'html'),
            hasBlurred: false,
            isFocused: false,
        };

        onChange = (value: any) => {
            this.setState({ value });
            this.props.input.onChange(
                !!value.getEditorState().getCurrentContent().getPlainText().trim()
                ? value.toString('html')
                : null
            );
        }

        onFocus = () => {
            this.setState({ isFocused: true });
        }

        onBlur = () => {
            this.setState({ hasBlurred: true });
        }

        fieldRequiredError = () => this.props.meta.error === "This field is required.";
        showError = () => this.fieldRequiredError()
                ? this.state.hasBlurred
                : this.state.isFocused && this.props.meta.error;

        render() {
            return <div>
                <RichTextEditor
                    className={classNames(this.props.classes.rte, this.showError() && this.props.classes.redBorder)}
                    value={this.state.value}
                    onChange={this.onChange}
                    onBlur={this.onBlur}
                    onFocus={this.onFocus}
                    placeholder={this.props.label} />
                    {this.showError() &&
                        <Typography className={this.props.classes.errorMessage}>
                            {this.props.meta.error}
                        </Typography>}
                </div>;
        }
    }
);

export const DateTextField = withStyles(styles)
    ((props: TextFieldProps) =>
        <MaterialTextField
            variant="standard"
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
            value={props.input.value} />
    );

interface TextFieldWithStartValueProps extends WrappedFieldProps {
    startValue: string;
    label?: string;
    children?: React.ReactNode;
}

export const TextFieldWithStartValue = (props: TextFieldWithStartValueProps) => {
    const [hasBeenTouched, setHasBeenTouched] = useState(false);

    useEffect(() => {
        props.input.onChange(props.startValue);
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    return (
        <FormControl variant='standard' fullWidth>
            <TextField
                {...props}
                input={{
                    ...props.input,
                    onFocus: () => setHasBeenTouched(true),
                    value: hasBeenTouched ? props.input.value : props.startValue,
                }}
                label={props.label}
                children={props.children}
            />
        </FormControl>
    );
};