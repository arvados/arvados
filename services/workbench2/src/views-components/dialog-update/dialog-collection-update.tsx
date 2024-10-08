// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { InjectedFormProps } from 'redux-form';
import { WithDialogProps } from 'store/dialog/with-dialog';
import { CollectionUpdateFormDialogData, COLLECTION_UPDATE_FORM_NAME } from 'store/collections/collection-update-actions';
import { FormDialog } from 'components/form-dialog/form-dialog';
import {
    CollectionNameField,
    CollectionDescriptionField,
    CollectionStorageClassesField
} from 'views-components/form-fields/collection-form-fields';
import { UpdateCollectionPropertiesForm } from 'views-components/collection-properties/update-collection-properties-form';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { FormGroup, FormLabel } from '@mui/material';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { resourcePropertiesList } from 'views-components/resource-properties/resource-properties-list';

type CssRules = 'propertiesForm';

const styles: CustomStyleRulesCallback<CssRules> = theme => ({
    propertiesForm: {
        marginTop: theme.spacing(2),
        marginBottom: theme.spacing(2),
    },
});

type DialogCollectionProps = WithDialogProps<{}> & InjectedFormProps<CollectionUpdateFormDialogData>;

export const DialogCollectionUpdate = (props: DialogCollectionProps) =>
    <FormDialog
        dialogTitle='Edit Collection'
        formFields={CollectionEditFields as any}
        submitLabel='Save'
        {...props}
    />;

const UpdateCollectionPropertiesList = resourcePropertiesList(COLLECTION_UPDATE_FORM_NAME);

const CollectionEditFields = withStyles(styles)(
    ({ classes }: WithStyles<CssRules>) => <span>
        <CollectionNameField />
        <CollectionDescriptionField />
        <div className={classes.propertiesForm}>
            <FormLabel>Properties</FormLabel>
            <FormGroup>
                <UpdateCollectionPropertiesForm />
                <UpdateCollectionPropertiesList />
            </FormGroup>
        </div>
        <CollectionStorageClassesField />
    </span>);
