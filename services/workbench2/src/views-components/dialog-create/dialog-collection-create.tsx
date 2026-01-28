// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Dispatch, compose } from 'redux';
import { connect } from 'react-redux'
import { WithDialogProps, withDialog } from 'store/dialog/with-dialog';
import { DialogContent, DialogTitle } from '@mui/material/';
import { CollectionCreateFormDialogData } from 'store/collections/collection-create-actions';
import {
    DialogCollectionNameField,
} from 'views-components/form-fields/collection-form-fields';
import { ResourceParentField } from '../form-fields/resource-form-fields';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { FormLabel } from '@mui/material';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { DialogForm } from 'components/dialog-form/dialog-form';
import { useStateWithValidation } from 'common/useStateWithValidation';
import { COLLECTION_NAME_VALIDATION, MAXLENGTH_524288_VALIDATION, REQUIRED_VALIDATION } from 'validators/validators';
import { DialogRichTextField } from 'components/dialog-form/dialog-text-field';
import { DialogResourcePropertiesForm } from 'views-components/resource-properties-form/resource-properties-form'
import { createCollection } from 'store/workbench/workbench-actions';
import { PropertyChips, getVocabularyFromChips } from 'components/chips/chips';
import { RootState } from 'store/store';
import { DialogMultiCheckboxField } from 'components/checkbox-field/checkbox-field'
import { DialogFileUploaderField } from '../file-uploader/file-uploader';
import { Vocabulary } from 'models/vocabulary';
import { COLLECTION_CREATE_FORM_NAME } from 'store/collections/collection-create-actions';

type CssRules = 'propertiesForm';

const styles: CustomStyleRulesCallback<CssRules> = theme => ({
    propertiesForm: {
        marginTop: theme.spacing(2),
        marginBottom: theme.spacing(2),
    },
});

const mapState = (state: RootState) => ({
    vocabulary: state.properties.vocabulary
});

const mapDispatch = (dispatch: Dispatch) => ({
    createCollection: (data: CollectionCreateFormDialogData, setSubmitErr: (errMsg: string) => void) => dispatch<any>(createCollection(data, setSubmitErr))
});

type DialogCollectionProps = WithDialogProps<CollectionCreateFormDialogData> & {
    createCollection: (data: CollectionCreateFormDialogData, setSubmitErr: (errMsg: string) => void) => void;
    vocabulary: Vocabulary;
};

export const DialogCollectionCreate = compose(
    connect(mapState, mapDispatch),
    withStyles(styles),
    withDialog(COLLECTION_CREATE_FORM_NAME)
)(({ createCollection, data, closeDialog, open, vocabulary, classes }: DialogCollectionProps & WithStyles<CssRules>) =>{
    const [collectionName, setCollectionName, collectionNameErrs] = useStateWithValidation('', [...REQUIRED_VALIDATION, ...COLLECTION_NAME_VALIDATION], 'Collection Name');
    const [description, setDescription, descriptionErrs] = useStateWithValidation('', MAXLENGTH_524288_VALIDATION, 'Description');
    const [chips, setChips] = React.useState<PropertyChips>({} as PropertyChips);
    const [storageClassesDesired, setStorageClassesDesired] = React.useState<string[]>([]);
    const [formErrors, setFormErrors] = React.useState<string[]>([]);
    const [submitErr, setSubmitErr] = React.useState<string>('');
    const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

    React.useEffect(() => {
        setFormErrors([...collectionNameErrs, ...descriptionErrs]);
        if (submitErr) {
            setFormErrors(prevErrors => [...prevErrors, submitErr]);
        }
    }, [collectionNameErrs, descriptionErrs, submitErr]);

    React.useEffect(() => {
        if (!open) {
            setIsSubmitting(false);
        }
        if (isSubmitting && submitErr) {
            setIsSubmitting(false);
        }
    }, [open, submitErr]);

    const fields = () => (
        <>
            <DialogTitle>New collection</DialogTitle>
            <DialogContent>
                <ResourceParentField ownerUuid={data ? data.ownerUuid : ''} />
                <DialogCollectionNameField setValue={setCollectionName} submitErr={submitErr} setSubmitErr={setSubmitErr} />
                <DialogRichTextField
                    label="Description"
                    defaultValue={description}
                    setValue={setDescription}
                    validators={MAXLENGTH_524288_VALIDATION}
                />
            <FormLabel>Properties</FormLabel>
                <DialogResourcePropertiesForm
                    setChips={setChips}
                    onSubmit={(ev)=> ev.preventDefault()}
                    />
                <DialogMultiCheckboxField
                    name="storageClassesDesired"
                    defaultValues={['default']}
                    label="Storage Classes"
                    onChange={setStorageClassesDesired}
                />
                <DialogFileUploaderField />
            </DialogContent>
        </>
    )

    return <DialogForm
        fields={fields()}
        submitLabel='Create a Collection'
        formErrors={formErrors}
        isSubmitting={isSubmitting}
        onSubmit={(ev) => {
            ev.preventDefault();
            setIsSubmitting(true);
            createCollection({
                ownerUuid: data.ownerUuid,
                name: collectionName,
                description: description,
                storageClassesDesired: storageClassesDesired,
                properties: getVocabularyFromChips(chips, vocabulary),
            },
            setSubmitErr);
        }}
        closeDialog={closeDialog}
        clearFormValues={() => {
            setCollectionName('');
            setDescription('');
            setChips({} as PropertyChips);
            setStorageClassesDesired([]);
        }}
        open={open}
    />;
});

