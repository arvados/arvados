// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useState, useEffect } from "react";
import classNames from "classnames";
import RichTextEditor from 'react-rte';
import { TextField, Typography } from "@mui/material";
import { getFieldErrors, Validator } from "validators/validators";
import { CustomStyleRulesCallback } from "common/custom-theme";
import { ArvadosTheme } from "common/custom-theme";
import { WithStyles } from "@mui/styles/withStyles/withStyles";
import withStyles from '@mui/styles/withStyles';

type RichTextCssRules = 'textField' | 'rte' | 'errorMessage' | 'redBorder';

const richTextStyles: CustomStyleRulesCallback<RichTextCssRules> = (theme: ArvadosTheme) => ({
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
        fontSize: '0.78rem',
        marginTop: '0.25rem',
    },
    redBorder: {
        border: `1px solid ${theme.palette.error.main}`,
    },
});

interface DialogTextFieldProps {
    disabled?: boolean;
    label: string;
    defaultValue: string;
    validators: Validator[];
    submitErr?: string;
    setValue: React.Dispatch<React.SetStateAction<string>>;
    setSubmitErr?: (errMsg: string) => void;
}

export const DialogTextField = React.memo(({  disabled, label, defaultValue, validators, submitErr, setValue, setSubmitErr }: DialogTextFieldProps) => {
    const [thisValue, setThisValue] = React.useState(defaultValue);
    const errs = getFieldErrors(thisValue, validators)

    React.useEffect(() => {
            setValue(thisValue.trim())
    }, [thisValue])

    return (
        <TextField
            disabled={disabled}
            value={thisValue}
            onChange={(e) => {
                setThisValue(e.target.value)
                if (submitErr && setSubmitErr) setSubmitErr('')
            }}
            autoFocus
            required
            error={errs.length > 0 || !!submitErr}
            helperText={errs.join(', ') || submitErr || ''}
            margin="dense"
            id="name"
            name="name"
            type="text"
            fullWidth
            variant="standard"
            label={label}
            onBlur={() => setValue(thisValue)}
        />
    )
})

type DialogRichTextFieldProps = {
    label: string;
    defaultValue: string;
    validators: Validator[];
    setValue: React.Dispatch<React.SetStateAction<string>>;
}

export const DialogRichTextField = withStyles(richTextStyles)((props: WithStyles<RichTextCssRules> & DialogRichTextFieldProps) => {
    const [rteValue, setRteValue] = useState(RichTextEditor.createValueFromString(props.defaultValue, 'html'));
    const [isFocused, setIsFocused] = useState(false);
    const [hasChanged, setHasChanged] = useState(false);
    const plainTextValue: string = rteValue.getEditorState().getCurrentContent().getPlainText().trim();
    const htmlValue: string = plainTextValue ? rteValue.toString('html') : '';
    const fieldErrors = getFieldErrors(plainTextValue, props.validators);

        useEffect(() => {
            props.setValue(htmlValue);
        }, []);

        useEffect(() => {
            if (isFocused && hasChanged) {
                props.setValue(htmlValue);
            }
        }, [isFocused, htmlValue]);

        useEffect(() => {
            if (isFocused) setHasChanged(true);
        }, [plainTextValue]);

        const onFocus = () => {
            setIsFocused(true);
        }

        const showError = () => fieldErrors.length > 0

            return <div>
                <RichTextEditor
                    className={classNames(props.classes.rte, showError() && props.classes.redBorder)}
                    value={rteValue}
                    onChange={(value) => {
                        setRteValue(value);
                    }}
                    onBlur={() => setIsFocused(false)}
                    onFocus={onFocus}
                    placeholder={props.label} />
                    {showError() &&
                        <Typography>
                            <span className={props.classes.errorMessage}>
                                {fieldErrors.join(', ')}
                            </span>
                        </Typography>}
                </div>;
        }
);
