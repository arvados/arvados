// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { reduxForm, Field, reset } from 'redux-form';
import { compose, Dispatch } from 'redux';
import { ArvadosTheme } from '../../common/custom-theme';
import { StyleRulesCallback, withStyles, WithStyles, TextField, Button, CircularProgress } from '@material-ui/core';
import { TagProperty } from '../../models/tag';
import { createCollectionTag, COLLECTION_TAG_FORM_NAME } from '../../store/collection-panel/collection-panel-action';
import { TAG_VALUE_VALIDATION, TAG_KEY_VALIDATION } from '../../validators/validators';

type CssRules = 'form' | 'textField' | 'buttonWrapper' | 'saveButton' | 'circularProgress';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    form: {
        marginBottom: theme.spacing.unit * 4 
    },
    textField: {
        marginRight: theme.spacing.unit
    },
    buttonWrapper: {
        position: 'relative',
        display: 'inline-block'
    },
    saveButton: {
        boxShadow: 'none'
    },
    circularProgress: {
        position: 'absolute',
        top: 0,
        bottom: 0,
        left: 0,
        right: 0,
        margin: 'auto'
    }
});

interface CollectionTagFormDataProps {
    submitting: boolean;
    invalid: boolean;
    pristine: boolean;
}

interface CollectionTagFormActionProps {
    handleSubmit: any;
}

interface TextFieldProps {
    label: string;
    floatinglabeltext: string;
    className?: string;
    input?: string;
    meta?: any;
}

type CollectionTagFormProps = CollectionTagFormDataProps & CollectionTagFormActionProps & WithStyles<CssRules>;

export const CollectionTagForm = compose(
    reduxForm({ 
        form: COLLECTION_TAG_FORM_NAME, 
        onSubmit: (data: TagProperty, dispatch: Dispatch) => {
            dispatch<any>(createCollectionTag(data));
            dispatch(reset(COLLECTION_TAG_FORM_NAME));
        } 
    }),
    withStyles(styles))(
        
    class CollectionTagForm extends React.Component<CollectionTagFormProps> {

            render() {
                const { classes, submitting, pristine, invalid, handleSubmit } = this.props;
                return (
                    <form className={classes.form} onSubmit={handleSubmit}>
                        <Field name="key"
                            disabled={submitting}
                            component={this.renderTextField}
                            floatinglabeltext="Key"
                            validate={TAG_KEY_VALIDATION}
                            className={classes.textField}
                            label="Key" />
                        <Field name="value"
                            disabled={submitting}
                            component={this.renderTextField}
                            floatinglabeltext="Value"
                            validate={TAG_VALUE_VALIDATION}
                            className={classes.textField}
                            label="Value" />
                        <div className={classes.buttonWrapper}>
                            <Button type="submit" className={classes.saveButton}
                                color="primary"
                                size='small'
                                disabled={invalid || submitting || pristine}
                                variant="contained">
                                ADD
                            </Button>
                            {submitting && <CircularProgress size={20} className={classes.circularProgress} />}
                        </div>
                    </form>
                );
            }

            renderTextField = ({ input, label, meta: { touched, error }, ...custom }: TextFieldProps) => (
                <TextField
                    helperText={touched && error}
                    label={label}
                    className={this.props.classes.textField}
                    error={touched && !!error}
                    autoComplete='off'
                    {...input}
                    {...custom}
                />
            )

        }

    );