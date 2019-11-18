// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Dispatch, compose } from 'redux';
import { connect } from 'react-redux';
import { InjectedFormProps, formValueSelector } from 'redux-form';
import { Grid, withStyles, StyleRulesCallback, WithStyles, Button } from '@material-ui/core';
import { RootState } from '~/store/store';
import {
    SEARCH_BAR_ADVANCE_FORM_NAME,
    changeAdvanceFormProperty,
    resetAdvanceFormProperty,
    updateAdvanceFormProperties
} from '~/store/search-bar/search-bar-actions';
import { PropertyValue } from '~/models/search-bar';
import { ArvadosTheme } from '~/common/custom-theme';
import { SearchBarKeyField, SearchBarValueField } from '~/views-components/form-fields/search-bar-form-fields';
import { Chips } from '~/components/chips/chips';
import { formatPropertyValue } from "~/common/formatters";
import { Vocabulary } from '~/models/vocabulary';
import { connectVocabulary } from '../resource-properties-form/property-field-common';

type CssRules = 'label' | 'button';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    label: {
        color: theme.palette.grey["500"],
        fontSize: '0.8125rem',
        alignSelf: 'center'
    },
    button: {
        boxShadow: 'none'
    }
});

interface SearchBarAdvancedPropertiesViewDataProps {
    submitting: boolean;
    invalid: boolean;
    pristine: boolean;
    propertyValues: PropertyValue;
    fields: PropertyValue[];
    vocabulary: Vocabulary;
}

interface SearchBarAdvancedPropertiesViewActionProps {
    setProps: () => void;
    addProp: (propertyValues: PropertyValue) => void;
    getAllFields: (propertyValues: PropertyValue[]) => PropertyValue[] | [];
}

type SearchBarAdvancedPropertiesViewProps = SearchBarAdvancedPropertiesViewDataProps
    & SearchBarAdvancedPropertiesViewActionProps
    & InjectedFormProps & WithStyles<CssRules>;

const selector = formValueSelector(SEARCH_BAR_ADVANCE_FORM_NAME);
const mapStateToProps = (state: RootState) => {
    return {
        propertyValues: selector(state, 'key', 'value', 'keyID', 'valueID')
    };
};

const mapDispatchToProps = (dispatch: Dispatch) => ({
    setProps: (propertyValues: PropertyValue[]) => {
        dispatch<any>(changeAdvanceFormProperty('properties', propertyValues));
    },
    addProp: (propertyValues: PropertyValue) => {
        dispatch<any>(updateAdvanceFormProperties(propertyValues));
        dispatch<any>(resetAdvanceFormProperty('key'));
        dispatch<any>(resetAdvanceFormProperty('value'));
        dispatch<any>(resetAdvanceFormProperty('keyID'));
        dispatch<any>(resetAdvanceFormProperty('valueID'));
    },
    getAllFields: (fields: any) => {
        return fields.getAll() || [];
    }
});

export const SearchBarAdvancedPropertiesView = compose(
    connectVocabulary,
    connect(mapStateToProps, mapDispatchToProps))(
    withStyles(styles)(
        ({ classes, fields, propertyValues, setProps, addProp, getAllFields, vocabulary }: SearchBarAdvancedPropertiesViewProps) =>
            <Grid container item xs={12} spacing={16}>
                <Grid item xs={2} className={classes.label}>Properties</Grid>
                <Grid item xs={4}>
                    <SearchBarKeyField />
                </Grid>
                <Grid item xs={4}>
                    <SearchBarValueField />
                </Grid>
                <Grid container item xs={2} justify='flex-end' alignItems="center">
                    <Button className={classes.button} onClick={() => addProp(propertyValues)}
                        color="primary"
                        size='small'
                        variant="contained"
                        disabled={!Boolean(propertyValues.key && propertyValues.value)}>
                        Add
                    </Button>
                </Grid>
                <Grid item xs={2} />
                <Grid container item xs={10} spacing={8}>
                    <Chips values={getAllFields(fields)}
                        deletable
                        onChange={setProps}
                        getLabel={(field: PropertyValue) => formatPropertyValue(field, vocabulary)} />
                </Grid>
            </Grid>
    )
);
