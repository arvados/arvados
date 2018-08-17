// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { reduxForm, Field, reset } from 'redux-form';
import { compose, Dispatch } from 'redux';
import { ArvadosTheme } from '~/common/custom-theme';
import { StyleRulesCallback, withStyles, WithStyles, Button, CircularProgress, Grid } from '@material-ui/core';
import { TagProperty } from '~/models/tag';
import { TextField } from '~/components/text-field/text-field';
import { createCollectionTag, COLLECTION_TAG_FORM_NAME } from '~/store/collection-panel/collection-panel-action';
import { TAG_VALUE_VALIDATION, TAG_KEY_VALIDATION } from '~/validators/validators';

type CssRules = 'buttonWrapper' | 'saveButton' | 'circularProgress';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
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
                    <form onSubmit={handleSubmit}>
                        <Grid container justify="flex-start" alignItems="baseline" spacing={24}>
                            <Grid item xs={3} component={"span"}>
                                <Field name="key"
                                    disabled={submitting}
                                    component={TextField}
                                    validate={TAG_KEY_VALIDATION}
                                    label="Key" />
                            </Grid>
                            <Grid item xs={5} component={"span"}>
                                <Field name="value"
                                    disabled={submitting}
                                    component={TextField}
                                    validate={TAG_VALUE_VALIDATION}
                                    label="Value" />
                            </Grid>
                            <Grid item component={"span"} className={classes.buttonWrapper}>
                                <Button type="submit" className={classes.saveButton}
                                    color="primary"
                                    size='small'
                                    disabled={invalid || submitting || pristine}
                                    variant="contained">
                                    ADD
                                </Button>
                                {submitting && <CircularProgress size={20} className={classes.circularProgress} />}
                            </Grid>
                        </Grid>
                    </form>
                );
            }
        }

    );
