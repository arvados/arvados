// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useEffect, useState } from 'react';
import { compose, Dispatch } from 'redux';
import { connect } from 'react-redux';
import { CollectionUpdateFormDialogData, updateCollection } from 'store/collections/collection-update-actions';
import { DialogForm } from 'components/dialog-form/dialog-form';
import { DialogCollectionNameField } from 'views-components/form-fields/collection-form-fields';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { FormGroup, FormLabel, DialogTitle, DialogContent } from '@mui/material';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { useStateWithValidation } from 'common/useStateWithValidation';
import { COLLECTION_NAME_VALIDATION, COLLECTION_DESCRIPTION_VALIDATION } from 'validators/validators';
import { DialogRichTextField } from 'components/dialog-form/dialog-text-field';
import { DialogResourcePropertiesForm } from 'views-components/resource-properties-form/resource-properties-form';
import { PropertyChips, getVocabularyFromChips, getChipsFromVocabulary } from 'components/chips/chips';
import { RootState } from 'store/store';
import { DialogMultiCheckboxField } from 'components/checkbox-field/checkbox-field';
import { Vocabulary } from 'models/vocabulary';
import { getStorageClasses } from 'common/config';
import { COLLECTION_UPDATE_FORM_NAME } from 'store/collections/collection-update-actions';
import { withDialog, WithDialogProps } from 'store/dialog/with-dialog';

type CssRules = 'propertiesForm';

const styles: CustomStyleRulesCallback<CssRules> = theme => ({
    propertiesForm: {
        marginTop: theme.spacing(2),
        marginBottom: theme.spacing(2),
    },
});

const mapState = (state: RootState) => ({
    vocabulary: state.properties.vocabulary,
    storageClasses: getStorageClasses(state.auth.config)
});

const mapDispatch = (dispatch: Dispatch) => ({
    updateCollection: (data: CollectionUpdateFormDialogData, setSubmitErr: (errMsg: string) => void) =>
        dispatch<any>(updateCollection(data))
});

type DialogCollectionProps = WithDialogProps<CollectionUpdateFormDialogData> & {
    updateCollection: (data: CollectionUpdateFormDialogData, setSubmitErr: (errMsg: string) => void) => void;
    vocabulary: Vocabulary;
    storageClasses: string[];
};

export const DialogCollectionUpdate = compose(
    connect(mapState, mapDispatch),
    withStyles(styles),
    withDialog(COLLECTION_UPDATE_FORM_NAME)
)(({ data, closeDialog, open, vocabulary, storageClasses, classes, updateCollection }: DialogCollectionProps & WithStyles<CssRules>) => {
        const [initialData, setInitialData] = useState<CollectionUpdateFormDialogData>(data);
        const [collectionName, setCollectionName, collectionNameErrs] = useStateWithValidation(initialData.name || '', COLLECTION_NAME_VALIDATION, 'Collection Name');
        const [description, setDescription, descriptionErrs] = useStateWithValidation(initialData.description || '', COLLECTION_DESCRIPTION_VALIDATION, 'Description');
        const [chips, setChips] = useState<PropertyChips>(getChipsFromVocabulary(initialData.properties || {}, vocabulary));
        const [storageClassesDesired, setStorageClassesDesired] = useState<string[]>(storageClasses || ['default']);
        const [formErrors, setFormErrors] = useState<string[]>([]);
        const [submitErr, setSubmitErr] = useState<string>('');
        const [isSubmitting, setIsSubmitting] = useState<boolean>(false);

        useEffect(() => {
            if (data.name) setCollectionName(data.name);
            if (data.description) setDescription(data.description);
            if (data.properties) setChips(getChipsFromVocabulary(data.properties, vocabulary));
            if (data.storageClassesDesired) setStorageClassesDesired(data.storageClassesDesired);
            setInitialData(data);
        }, [data.name, data.description, data.properties, data.storageClassesDesired]);

        useEffect(() => {
            setFormErrors([...collectionNameErrs, ...descriptionErrs]);
            if (submitErr) {
                setFormErrors(prevErrors => [...prevErrors, submitErr]);
            }
        }, [collectionNameErrs, descriptionErrs, submitErr]);

        useEffect(() => {
            if (!open) {
                setIsSubmitting(false);
            }
            if (isSubmitting && submitErr) {
                setIsSubmitting(false);
            }
        }, [open, submitErr]);

        const fields = () => (
            <>
                <DialogTitle>Edit Collection</DialogTitle>
                <DialogContent>
                    <DialogCollectionNameField
                        setValue={setCollectionName}
                        defaultValue={initialData.name}
                        submitErr={submitErr}
                        setSubmitErr={setSubmitErr}
                    />
                    <DialogRichTextField
                        label="Description"
                        defaultValue={description}
                        setValue={setDescription}
                        validators={COLLECTION_DESCRIPTION_VALIDATION}
                    />
                    <div className={classes.propertiesForm}>
                        <FormLabel>Properties</FormLabel>
                        <FormGroup>
                            <DialogResourcePropertiesForm
                                initialProperties={getChipsFromVocabulary(initialData.properties || {}, vocabulary)}
                                setChips={setChips}
                                onSubmit={(ev) => ev.preventDefault()}
                            />
                        </FormGroup>
                    </div>
                    <DialogMultiCheckboxField
                        name="storageClassesDesired"
                        defaultValues={storageClassesDesired}
                        label="Storage Classes"
                        onChange={setStorageClassesDesired}
                        minSelection={1}
                        helperText='At least one class should be selected'
                    />
                </DialogContent>
            </>
        );

        return (
            <DialogForm
                fields={fields()}
                submitLabel='Save'
                formErrors={formErrors}
                isSubmitting={isSubmitting}
                onSubmit={(ev) => {
                    ev.preventDefault();
                    setIsSubmitting(true);
                    updateCollection({
                        uuid: initialData.uuid,
                        name: collectionName,
                        description: description,
                        storageClassesDesired: storageClassesDesired,
                        properties:  getVocabularyFromChips(chips, vocabulary),
                    }, setSubmitErr);
                }}
                closeDialog={closeDialog}
                clearFormValues={() => {
                    setCollectionName('');
                    setDescription('');
                    setChips({} as PropertyChips);
                    setStorageClassesDesired([]);
                }}
                open={open}
            />
        );
    }
);
