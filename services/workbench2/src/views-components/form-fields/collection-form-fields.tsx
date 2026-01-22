// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { Field, Validator } from "redux-form";
import { DialogTextField } from "components/dialog-form/dialog-text-field";
import { TextField, RichEditorTextField } from "components/text-field/text-field";
import {
    COLLECTION_NAME_VALIDATION, COLLECTION_NAME_VALIDATION_ALLOW_SLASH,
    COLLECTION_DESCRIPTION_VALIDATION, COLLECTION_PROJECT_VALIDATION
} from "validators/validators";
import { ProjectTreePickerField, DirectoryTreePickerField } from "views-components/projects-tree-picker/tree-picker-field";
import { PickerIdProp } from 'store/tree-picker/picker-id';
import { connect } from "react-redux";
import { RootState } from "store/store";
import { MultiCheckboxField } from "components/checkbox-field/checkbox-field";
import { getStorageClasses } from "common/config";
import { ERROR_MESSAGE } from "validators/require";

interface CollectionNameFieldProps {
    validate: Validator[];
}

// See implementation note on declaration of ProjectNameField

export const CollectionNameField = connect(
    (state: RootState) => {
        return {
            validate: (state.auth.config.clusterConfig.Collections.ForwardSlashNameSubstitution === "" ?
                COLLECTION_NAME_VALIDATION : COLLECTION_NAME_VALIDATION_ALLOW_SLASH)
        };
    })((props: CollectionNameFieldProps) =>
        <span data-cy='name-field'><Field
            name='name'
            component={TextField as any}
            validate={props.validate}
            label="Collection Name"
            autoFocus={true} /></span>
    );

type DialogCollectionNameFieldProps = {
    defaultValue?: string;
    setValue: (value: string) => void;
    validators: Validator[];
}

export const DialogCollectionNameField = connect(
    (state: RootState) => {
        return {
            validators: (state.auth.config.clusterConfig.Collections.ForwardSlashNameSubstitution === "" ?
                COLLECTION_NAME_VALIDATION : COLLECTION_NAME_VALIDATION_ALLOW_SLASH)
        };
    })(({ defaultValue, setValue, validators }: DialogCollectionNameFieldProps) => {
        return <span data-cy='name-field'>
            <DialogTextField
                label='Collection Name'
                defaultValue={defaultValue || ''}
                setValue={setValue}
                validators={validators}
            />
        </span>
    })

export const CollectionDescriptionField = () =>
    <Field
        name='description'
        component={RichEditorTextField as any}
        validate={COLLECTION_DESCRIPTION_VALIDATION}
        label="Description" />;

export const CollectionProjectPickerField = (props: PickerIdProp) =>
    <Field
        name="projectUuid"
        pickerId={props.pickerId}
        component={ProjectTreePickerField}
        validate={COLLECTION_PROJECT_VALIDATION} />;

const validateDirectory = (val) => (val && val.uuid ? undefined : ERROR_MESSAGE);

export const DirectoryPickerField = (props: PickerIdProp) =>
    <Field
        name="destination"
        pickerId={props.pickerId}
        component={DirectoryTreePickerField as any}
        validate={validateDirectory} />;

interface StorageClassesProps {
    items: string[];
    defaultClasses?: string[];
}

export const CollectionStorageClassesField = connect(
    (state: RootState) => {
        return {
            items: getStorageClasses(state.auth.config)
        };
    })(
    (props: StorageClassesProps) =>
        <Field
            name='storageClassesDesired'
            label='Storage classes'
            minSelection={1}
            rowLayout={true}
            defaultValues={props.defaultClasses}
            helperText='At least one class should be selected'
            component={MultiCheckboxField}
            items={props.items} />);
