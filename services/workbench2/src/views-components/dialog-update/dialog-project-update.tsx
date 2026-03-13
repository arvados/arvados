// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useEffect, useState } from 'react';
import { compose, Dispatch } from 'redux';
import { connect } from 'react-redux';
import { ProjectUpdateFormDialogData, PROJECT_UPDATE_FORM_NAME } from 'store/projects/project-update-actions';
import { updateProjectRunner } from 'store/workbench/workbench-actions'
import { updateGroup } from 'store/groups-panel/groups-panel-actions';
import { DialogForm } from 'components/dialog-form/dialog-form';
import { DialogTextField, DialogRichTextField } from 'components/dialog-form/dialog-text-field';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { FormGroup, FormLabel, DialogTitle, DialogContent } from '@mui/material';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { useStateWithValidation } from 'common/useStateWithValidation';
import { PROJECT_NAME_VALIDATION, PROJECT_NAME_VALIDATION_ALLOW_SLASH, PROJECT_DESCRIPTION_VALIDATION } from 'validators/validators';
import { DialogResourcePropertiesForm } from 'views-components/resource-properties-form/resource-properties-form';
import { PropertyChips, getVocabularyFromChips, getChipsFromVocabulary } from 'components/chips/chips';
import { RootState } from 'store/store';
import { Vocabulary } from 'models/vocabulary';
import { withDialog, WithDialogProps } from 'store/dialog/with-dialog';
import { GroupClass } from 'models/group';
import { isEqual } from 'lodash';

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
    updateProject: (data: ProjectUpdateFormDialogData, setSubmitErr: (errMsg: string) => void) =>
        dispatch<any>(updateProjectRunner(data, setSubmitErr)),
    updateGroup: (data: ProjectUpdateFormDialogData, setSubmitErr: (errMsg: string) => void) =>
        dispatch<any>(updateGroup(data, setSubmitErr))
});

type DialogProjectProps = WithDialogProps<{sourcePanel: GroupClass} & ProjectUpdateFormDialogData> & {
    updateProject: (data: ProjectUpdateFormDialogData, setSubmitErr: (errMsg: string) => void) => void;
    updateGroup: (data: ProjectUpdateFormDialogData, setSubmitErr: (errMsg: string) => void) => void;
    vocabulary: Vocabulary;
    allowSlash: boolean;
};

export const DialogProjectUpdate = compose(
    connect(mapState, mapDispatch),
    withStyles(styles),
    withDialog(PROJECT_UPDATE_FORM_NAME)
)(({ data, closeDialog, open, vocabulary, allowSlash, classes, updateProject, updateGroup }: DialogProjectProps & WithStyles<CssRules>) => {
        const initialData = data || { uuid: '', name: '', description: '', properties: {} };
    const initialProperties = initialData.properties || {};
        const [projectName, setProjectName, projectNameErrs] = useStateWithValidation(initialData.name || '', PROJECT_NAME_VALIDATION, 'Project Name');
        const [description, setDescription, descriptionErrs] = useStateWithValidation(initialData.description || '', PROJECT_DESCRIPTION_VALIDATION, 'Description');
    const [chips, setChips] = useState<PropertyChips>(getChipsFromVocabulary(initialProperties, vocabulary));
        const [formErrors, setFormErrors] = useState<string[]>([]);
        const [submitErr, setSubmitErr] = useState<string>('');
        const [isSubmitting, setIsSubmitting] = useState<boolean>(false);

        const sourcePanel = data?.sourcePanel || GroupClass.PROJECT;
            const isGroup = sourcePanel === GroupClass.ROLE;
            const title = isGroup ? 'Edit Group' : 'Edit Project';
        const currentProperties = getVocabularyFromChips(chips, vocabulary);
        const submitDisabled = !projectNameErrs.length && !descriptionErrs.length &&
            projectName === (initialData.name || '') &&
            description === (initialData.description || '') &&
            isEqual(currentProperties, initialProperties);

        useEffect(() => {
            if (data) {
                setProjectName(data.name || '');
                setDescription(data.description || '');
                setChips(getChipsFromVocabulary(data.properties || {}, vocabulary));
            }
        }, [data, vocabulary]);

        useEffect(() => {
            setFormErrors([...projectNameErrs, ...descriptionErrs]);
            if (submitErr) {
                setFormErrors(prevErrors => [...prevErrors, submitErr]);
            }
        }, [projectNameErrs, descriptionErrs, submitErr]);

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
                <DialogTitle>{title}</DialogTitle>
                <DialogContent>
                    <DialogTextField
                        label={isGroup ? "Group Name" : "Project Name"}
                        defaultValue={projectName}
                        setValue={setProjectName}
                        validators={allowSlash ? PROJECT_NAME_VALIDATION_ALLOW_SLASH : PROJECT_NAME_VALIDATION}
                        submitErr={submitErr}
                        setSubmitErr={setSubmitErr}
                    />
                    <DialogRichTextField
                        label="Description"
                        defaultValue={description}
                        setValue={setDescription}
                        validators={PROJECT_DESCRIPTION_VALIDATION}
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
                </DialogContent>
            </>
        );

        return (
            <DialogForm
                fields={fields()}
                submitLabel='Save'
                formErrors={formErrors}
                submitDisabled={submitDisabled}
                isSubmitting={isSubmitting}
                onSubmit={(ev) => {
                    ev.preventDefault();
                    setIsSubmitting(true);
                    const updateFn = sourcePanel === GroupClass.ROLE ? updateGroup : updateProject;
                    updateFn({
                        uuid: initialData.uuid,
                        name: projectName,
                        description: description,
                        properties: currentProperties,
                    }, setSubmitErr);
                }}
                closeDialog={closeDialog}
                clearFormValues={() => {
                    setProjectName('');
                    setDescription('');
                    setChips({} as PropertyChips);
                }}
                open={open}
            />
        );
    }
);
