// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { InjectedFormProps, Field } from 'redux-form';
import { WithDialogProps } from 'store/dialog/with-dialog';
import { CollectionCreateFormDialogData, COLLECTION_CREATE_FORM_NAME } from 'store/collections/collection-create-actions';
import { FormDialog } from 'components/form-dialog/form-dialog';
import {
    CollectionNameField,
    CollectionDescriptionField,
    CollectionStorageClassesField
} from 'views-components/form-fields/collection-form-fields';
import { FileUploaderField } from '../file-uploader/file-uploader';
import { ResourceParentField } from '../form-fields/resource-form-fields';
import { CreateCollectionPropertiesForm } from 'views-components/collection-properties/create-collection-properties-form';
import { FormGroup, FormLabel, StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core';
import { resourcePropertiesList } from 'views-components/resource-properties/resource-properties-list';

type CssRules = 'propertiesForm';

const styles: StyleRulesCallback<CssRules> = theme => ({
    propertiesForm: {
        marginTop: theme.spacing.unit * 2,
        marginBottom: theme.spacing.unit * 2,
    },
});

type DialogCollectionProps = WithDialogProps<{}> & InjectedFormProps<CollectionCreateFormDialogData>;

export const DialogCollectionCreate = (props: DialogCollectionProps) =>
    <FormDialog
        dialogTitle='New collection'
        formFields={CollectionAddFields as any}
        submitLabel='Create a Collection'
        {...props}
    />;

const CreateCollectionPropertiesList = resourcePropertiesList(COLLECTION_CREATE_FORM_NAME);

const CollectionAddFields = withStyles(styles)(
    ({ classes }: WithStyles<CssRules>) => <span>
        <ResourceParentField />
        <CollectionNameField />
        <CollectionDescriptionField />
        <div className={classes.propertiesForm}>
            <FormLabel>Properties</FormLabel>
            <FormGroup>
                <CreateCollectionPropertiesForm />
                <CreateCollectionPropertiesList />
            </FormGroup>
        </div>
        <CollectionStorageClassesField defaultClasses={['default']} />
        <Field
            name='files'
            label='Files'
            component={FileUploaderField} />
    </span>);

