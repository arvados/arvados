// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { reduxForm, InjectedFormProps, reset } from 'redux-form';
import { compose, Dispatch } from 'redux';
import { Paper, StyleRulesCallback, withStyles, WithStyles, Button, Grid, IconButton, CircularProgress } from '@material-ui/core';
import {
    SEARCH_BAR_ADVANCED_FORM_NAME, SEARCH_BAR_ADVANCED_FORM_PICKER_ID,
    searchAdvancedData,
    setSearchValueFromAdvancedData
} from 'store/search-bar/search-bar-actions';
import { ArvadosTheme } from 'common/custom-theme';
import { CloseIcon } from 'components/icon/icon';
import { SearchBarAdvancedFormData } from 'models/search-bar';
import {
    SearchBarTypeField, SearchBarClusterField, SearchBarProjectField, SearchBarTrashField,
    SearchBarDateFromField, SearchBarDateToField, SearchBarPropertiesField,
    SearchBarSaveSearchField, SearchBarQuerySearchField, SearchBarPastVersionsField
} from 'views-components/form-fields/search-bar-form-fields';
import { treePickerActions } from "store/tree-picker/tree-picker-actions";

type CssRules = 'container' | 'closeIcon' | 'label' | 'buttonWrapper'
    | 'button' | 'circularProgress' | 'searchView' | 'selectGrid';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    container: {
        padding: theme.spacing.unit * 2,
        borderBottom: `1px solid ${theme.palette.grey["200"]}`,
        position: 'relative',
    },
    closeIcon: {
        position: 'absolute',
        top: '12px',
        right: '12px'
    },
    label: {
        color: theme.palette.grey["500"],
        fontSize: '0.8125rem',
        alignSelf: 'center'
    },
    buttonWrapper: {
        marginRight: '14px',
        marginTop: '14px',
        position: 'relative',
    },
    button: {
        boxShadow: 'none'
    },
    circularProgress: {
        position: 'absolute',
        top: 0,
        bottom: 0,
        left: 0,
        right: 0,
        margin: 'auto'
    },
    searchView: {
        color: theme.palette.common.black,
        borderRadius: `0 0 ${theme.spacing.unit / 2}px ${theme.spacing.unit / 2}px`
    },
    selectGrid: {
        marginBottom: theme.spacing.unit * 2
    }
});

// ToDo: maybe we should remove invalid and prostine
interface SearchBarAdvancedViewFormDataProps {
    submitting: boolean;
    invalid: boolean;
    pristine: boolean;
}

// ToDo: maybe we should remove tags
export interface SearchBarAdvancedViewDataProps {
    tags: any;
    saveQuery: boolean;
}

export interface SearchBarAdvancedViewActionProps {
    closeAdvanceView: () => void;
}

type SearchBarAdvancedViewProps = SearchBarAdvancedViewActionProps & SearchBarAdvancedViewDataProps;

type SearchBarAdvancedViewFormProps = SearchBarAdvancedViewProps & SearchBarAdvancedViewFormDataProps
    & InjectedFormProps & WithStyles<CssRules>;

const validate = (values: any) => {
    const errors: any = {};

    if (values.dateFrom && values.dateTo) {
        if (new Date(values.dateFrom).getTime() > new Date(values.dateTo).getTime()) {
            errors.dateFrom = 'Invalid date';
        }
    }

    return errors;
};

export const SearchBarAdvancedView = compose(
    reduxForm<SearchBarAdvancedFormData, SearchBarAdvancedViewProps>({
        form: SEARCH_BAR_ADVANCED_FORM_NAME,
        validate,
        onSubmit: (data: SearchBarAdvancedFormData, dispatch: Dispatch) => {
            dispatch<any>(searchAdvancedData(data));
            dispatch(reset(SEARCH_BAR_ADVANCED_FORM_NAME));
            dispatch(treePickerActions.DEACTIVATE_TREE_PICKER_NODE({ pickerId: SEARCH_BAR_ADVANCED_FORM_PICKER_ID }));
        },
        onChange: (data: SearchBarAdvancedFormData, dispatch: Dispatch, props: any, prevData: SearchBarAdvancedFormData) => {
            dispatch<any>(setSearchValueFromAdvancedData(data, prevData));
        },
    }),
    withStyles(styles))(
        ({ classes, closeAdvanceView, handleSubmit, submitting, invalid, pristine, tags, saveQuery }: SearchBarAdvancedViewFormProps) =>
            <Paper className={classes.searchView}>
                <form onSubmit={handleSubmit}>
                    <Grid container direction="column" justify="flex-start" alignItems="flex-start">
                        <Grid item xs={12} container className={classes.container}>
                            <Grid item container xs={12} className={classes.selectGrid}>
                                <Grid item xs={2} className={classes.label}>Type</Grid>
                                <Grid item xs={5}>
                                    <SearchBarTypeField />
                                </Grid>
                            </Grid>
                            <Grid item container xs={12} className={classes.selectGrid}>
                                <Grid item xs={2} className={classes.label}>Cluster</Grid>
                                <Grid item xs={5}>
                                    <SearchBarClusterField />
                                </Grid>
                            </Grid>
                            <Grid item container xs={12}>
                                <Grid item xs={2} className={classes.label}>Project</Grid>
                                <Grid item xs={10}>
                                    <SearchBarProjectField />
                                </Grid>
                            </Grid>
                            <Grid item container xs={12}>
                                <Grid item xs={2} className={classes.label} />
                                <Grid item xs={5}>
                                    <SearchBarTrashField />
                                </Grid>
                                <Grid item xs={5}>
                                    <SearchBarPastVersionsField />
                                </Grid>
                            </Grid>
                            <IconButton onClick={closeAdvanceView} className={classes.closeIcon}>
                                <CloseIcon />
                            </IconButton>
                        </Grid>
                        <Grid container item xs={12} className={classes.container} spacing={16}>
                            <Grid item xs={2} className={classes.label}>Date modified</Grid>
                            <Grid item xs={4}>
                                <SearchBarDateFromField />
                            </Grid>
                            <Grid item xs={4}>
                                <SearchBarDateToField />
                            </Grid>
                        </Grid>
                        <Grid container item xs={12} className={classes.container}>
                            <SearchBarPropertiesField />
                            <Grid container item xs={12} justify="flex-start" alignItems="center" spacing={16}>
                                <Grid item xs={2} className={classes.label} />
                                <Grid item xs={4}>
                                    <SearchBarSaveSearchField />
                                </Grid>
                                <Grid item xs={4}>
                                    {saveQuery && <SearchBarQuerySearchField />}
                                </Grid>
                            </Grid>
                            <Grid container item xs={12} justify='flex-end'>
                                <div className={classes.buttonWrapper}>
                                    <Button type="submit" className={classes.button}
                                        // ToDo: create easier condition
                                        // Question: do we need this condition?
                                        // disabled={invalid || submitting || pristine || !!(tags && tags.values && ((tags.values.key) || (tags.values.value)) && !Object.keys(tags.values).find(el => el !== 'value' && el !== 'key'))}
                                        color="primary"
                                        size='small'
                                        variant="contained">
                                        Search
                                    </Button>
                                    {submitting && <CircularProgress size={20} className={classes.circularProgress} />}
                                </div>
                            </Grid>
                        </Grid>
                    </Grid>
                </form>
            </Paper>
    );
