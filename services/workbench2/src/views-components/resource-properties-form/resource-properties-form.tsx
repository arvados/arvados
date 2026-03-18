// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { RootState } from 'store/store';
import { connect } from 'react-redux';
import { Grid } from '@mui/material';
import withStyles from '@mui/styles/withStyles';
import { DialogPropertyKeyInput, PROPERTY_KEY_FIELD_NAME, PROPERTY_KEY_FIELD_ID } from './property-key-field';
import { DialogPropertyValueInput, PROPERTY_VALUE_FIELD_NAME, PROPERTY_VALUE_FIELD_ID } from './property-value-field';
import { getTagKeyID, Vocabulary } from 'models/vocabulary';
import { ProgressButton } from 'components/progress-button/progress-button';
import { Chips, PropertyChips, formatChips } from 'components/chips/chips'

const AddButton = withStyles(theme => ({
    root: { marginTop: theme.spacing(1) }
}))(ProgressButton);
export interface ResourcePropertiesFormData {
    uuid: string;
    [PROPERTY_KEY_FIELD_NAME]: string;
    [PROPERTY_KEY_FIELD_ID]: string;
    [PROPERTY_VALUE_FIELD_NAME]: string;
    [PROPERTY_VALUE_FIELD_ID]: string;
    clearPropertyKeyOnSelect?: boolean;
}

const mapState = (state: RootState) => {
    return {
        vocabulary: state.properties.vocabulary
    }
}

type DialogResourcePropertiesFormProps = {
    initialProperties?: PropertyChips;
    vocabulary: Vocabulary,
    setChips: React.Dispatch<React.SetStateAction<PropertyChips>>,
    onSubmit: (event: React.FormEvent<HTMLFormElement>) => void,
};

export const DialogResourcePropertiesForm = connect(mapState)(({ vocabulary, setChips, initialProperties }: DialogResourcePropertiesFormProps) => {
    const [properties, setProperties] = React.useState<PropertyChips>(initialProperties || {});
    const [propertyKeyId, setPropertyKeyId] = React.useState<string | undefined>(undefined);
    const [currentKey, setCurrentKey] = React.useState<string | undefined>(undefined);
    const [currentValue, setCurrentValue] = React.useState<string | undefined>(undefined);
    const [keyErrors, setKeyErrors] = React.useState<string[]>([]);
    const [valueErrors, setValueErrors] = React.useState<string[]>([]);

    React.useEffect(() => {
        if (currentKey) {
            setPropertyKeyId(getTagKeyID(currentKey, vocabulary));
        } else {
            setPropertyKeyId(undefined);
        }
    }, [currentKey]);

    React.useEffect(() => {
        setChips(properties);
    }, [properties]);

    const addPropertyValue = (prev: PropertyChips, key: string, value: string): PropertyChips => {
        console.log('>>> Adding property to chips:', key, ':', value);
        const existing = prev[key];
        if (Array.isArray(existing)) {
            return existing.includes(value) ? prev : { ...prev, [key]: [...existing, value] };
        }
        if (typeof existing === 'string') {
            return existing === value ? prev : { ...prev, [key]: [existing, value] };
        }
        return { ...prev, [key]: value };
    };

    const handleAddProperty = (ev: React.FormEvent) => {
        ev.preventDefault();
        console.log('>>> Adding property', currentKey, ':', currentValue);
        if (!currentKey || !currentValue) return;
        setProperties(prev => addPropertyValue(prev, currentKey, currentValue));
        setCurrentValue(undefined);
    };

    const onChipsChange = (newValues: string[]) => {
        const newProperties: PropertyChips = {};
        for (const chip of newValues) {
            const [key, value] = chip.split(': ').map(s => s.trim());
            if (newProperties[key]) {
                if (Array.isArray(newProperties[key])) {
                    (newProperties[key] as string[]).push(value);
                } else {
                    newProperties[key] = [newProperties[key] as string, value];
                }
            } else {
                newProperties[key] = value;
            }
        }
        setProperties(newProperties);
    };

    return <form data-cy='resource-properties-form'>
        <Grid container spacing={2}>
            <Grid item xs
            data-cy='property-field-key'>
                <DialogPropertyKeyInput
                    clearPropertyKeyOnSelect={true}
                    vocabulary={vocabulary}
                    onSelect={setCurrentKey}
                    setKeyErrors={setKeyErrors}
                    // used to clear the value field when a new key is selected
                    setCurrentValue={setCurrentValue}
                />
            </Grid>
            <Grid item xs
            data-cy='property-field-value'>
                <DialogPropertyValueInput
                    propertyKeyId={propertyKeyId || ''}
                    propertyKeyName={currentKey || ''}
                    vocabulary={vocabulary}
                    currentValue={currentValue}
                    onSelect={setCurrentValue}
                    setValueErrors={setValueErrors}
                />
            </Grid>
            <Grid item>
                <AddButton
                    data-cy='property-add-btn'
                    disabled={keyErrors.length > 0 || valueErrors.length > 0 || !currentKey || !currentValue}
                    color='primary'
                    variant='contained'
                    onClick={handleAddProperty}
                    >
                    Add
                </AddButton>
            </Grid>
        </Grid>
        <Grid data-cy='resource-properties-list'>
            <Chips
                values={formatChips(properties)}
                clickable={true}
                deletable={true}
                onChange={onChipsChange}
            />
        </Grid>
    </form>
});
