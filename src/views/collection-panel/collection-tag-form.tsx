// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { reduxForm, Field, reset } from 'redux-form';
import { compose, Dispatch } from 'redux';
import { ArvadosTheme } from '~/common/custom-theme';
import { StyleRulesCallback, withStyles, WithStyles, Button, CircularProgress, Grid, Typography } from '@material-ui/core';
import { TagProperty } from '~/models/tag';
import { TextField } from '~/components/text-field/text-field';
import { createCollectionTag, COLLECTION_TAG_FORM_NAME } from '~/store/collection-panel/collection-panel-action';
import { TAG_VALUE_VALIDATION, TAG_KEY_VALIDATION } from '~/validators/validators';

type CssRules = 'root' | 'keyField' | 'valueField' | 'buttonWrapper' | 'saveButton' | 'circularProgress';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
        display: 'flex'
    },
    keyField: {
        width: '25%',
        marginRight: theme.spacing.unit * 3
    },
    valueField: {
        width: '40%',
        marginRight: theme.spacing.unit * 3
    },
    buttonWrapper: {
        paddingTop: '14px',
        position: 'relative',
    },
    saveButton: {
        boxShadow: 'none'
    },
    circularProgress: {
        position: 'absolute',
        top: -9,
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
                    <form onSubmit={handleSubmit} className={classes.root}>
                        <div className={classes.keyField}>
                            <Field name="key"
                                disabled={submitting}
                                component={TextField}
                                validate={TAG_KEY_VALIDATION}
                                label="Key" />
                        </div>
                        <div className={classes.valueField}>
                            <Field name="value"
                                disabled={submitting}
                                component={TextField}
                                validate={TAG_VALUE_VALIDATION}
                                label="Value" />
                        </div>
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
        }

    );
