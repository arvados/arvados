// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Dispatch } from 'redux';
import { connect } from 'react-redux'
import { WithDialogProps } from 'store/dialog/with-dialog';
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
import { createCollection } from 'store/collections/collection-create-actions';
import { PropertyChips, getVocabularyFromChips } from 'components/chips/chips';
import { RootState } from 'store/store';
import { DialogMultiCheckboxField } from 'components/checkbox-field/checkbox-field'
import { DialogFileUploaderField } from '../file-uploader/file-uploader';

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
    createCollection: (data: CollectionCreateFormDialogData) => dispatch<any>(createCollection(data))
});

type DialogCollectionProps = WithDialogProps<CollectionCreateFormDialogData> & ReturnType<typeof mapDispatch> & ReturnType<typeof mapState>;

export const DialogCollectionCreate = connect(mapState, mapDispatch)(withStyles(styles)(({ createCollection, data, closeDialog, open, vocabulary, classes }: DialogCollectionProps & WithStyles<CssRules>) =>{
    const [collectionName, setCollectionName, collectionNameErrs] = useStateWithValidation('', [...REQUIRED_VALIDATION, ...COLLECTION_NAME_VALIDATION], 'Collection Name');
    const [description, setDescription, descriptionErrs] = useStateWithValidation('', MAXLENGTH_524288_VALIDATION, 'Description');
    const [chips, setChips, chipsErrs] = useStateWithValidation({} as PropertyChips, [], 'Properties');
    const [formErrors, setFormErrors] = React.useState<string[]>([]);

    React.useEffect(() => {
        setFormErrors([...collectionNameErrs, ...descriptionErrs, ...chipsErrs]);
    }, [collectionNameErrs, descriptionErrs, chipsErrs]);

    const fields = () => (
        <>
            <DialogTitle>New collection</DialogTitle>
            <DialogContent>
                <ResourceParentField ownerUuid={data.ownerUuid} />
                <DialogCollectionNameField setValue={setCollectionName} />
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
                    value={[]}
                    name="storageClassesDesired"
                    defaultValues={['default']}
                    label="Storage Classes"
                    onChange={() => {}}
                />
                <DialogFileUploaderField
                    label="Files"
                    onChange={() => {}}
                />
            </DialogContent>
        </>
)

    return <DialogForm
        fields={fields()}
        submitLabel='Create a Collection'
        formErrors={formErrors}
        onSubmit={(ev) => {
            ev.preventDefault();
            createCollection({
                ownerUuid: data.ownerUuid,
                name: collectionName,
                description: description,
                storageClassesDesired: [],
                properties: getVocabularyFromChips(chips, vocabulary)
            })
        }}
        closeDialog={closeDialog}
        clearFormValues={() => {
            setCollectionName('');
            setDescription('');
            setChips({} as PropertyChips);
        }}
        open={open}
    />;
}));

