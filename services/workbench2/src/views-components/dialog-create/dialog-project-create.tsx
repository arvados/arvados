// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useState, useEffect} from 'react';
import { DialogTitle, DialogContent, FormGroup, FormLabel } from '@mui/material';
import { Dispatch, compose } from 'redux';
import { connect } from 'react-redux';
import { WithDialogProps, withDialog } from 'store/dialog/with-dialog';
import { ProjectCreateFormDialogData, PROJECT_CREATE_FORM_NAME } from 'store/projects/project-create-actions';
import { ResourceParentField } from '../form-fields/resource-form-fields';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { GroupClass } from 'models/group';
import { DialogForm } from 'components/dialog-form/dialog-form';
import { useStateWithValidation } from 'common/useStateWithValidation';
import { PROJECT_NAME_VALIDATION, PROJECT_NAME_VALIDATION_ALLOW_SLASH, PROJECT_DESCRIPTION_VALIDATION, REQUIRED_VALIDATION, MAXLENGTH_524288_VALIDATION } from 'validators/validators';
import { DialogTextField, DialogRichTextField } from 'components/dialog-form/dialog-text-field';
import { DialogResourcePropertiesForm } from 'views-components/resource-properties-form/resource-properties-form';
import { createProjectRunner } from 'store/workbench/workbench-actions';
import { createGroupRunner, GroupCreateFormDialogData } from 'store/groups-panel/groups-panel-actions';
import { PropertyChips, getVocabularyFromChips } from 'components/chips/chips';
import { RootState } from 'store/store';
import { Vocabulary } from 'models/vocabulary';
import { Participant, ParticipantSelect } from 'views-components/sharing-dialog/participant-select';

type CssRules = 'propertiesForm' | 'description';

const styles: CustomStyleRulesCallback<CssRules> = theme => ({
    propertiesForm: {
        marginTop: theme.spacing(2),
        marginBottom: theme.spacing(2),
    },
    description: {
        marginTop: theme.spacing(2),
        marginBottom: theme.spacing(2),
    },
});

const mapState = (state: RootState) => ({
    vocabulary: state.properties.vocabulary,
    allowSlash: state.auth.config.clusterConfig.Collections.ForwardSlashNameSubstitution !== ""
});

const mapDispatch = (dispatch: Dispatch) => ({
    createProject: (data: ProjectCreateFormDialogData, setSubmitErr: (err: string) => void) => dispatch<any>(createProjectRunner(data, setSubmitErr)),
    createGroup: (data: GroupCreateFormDialogData, setSubmitErr: (err: string) => void) => dispatch<any>(createGroupRunner(data, setSubmitErr))
});

type DialogProjectProps = WithDialogProps<{sourcePanel: GroupClass, ownerUuid: string}> & {
    createProject: (data: ProjectCreateFormDialogData, setSubmitErr: (err: string) => void) => void;
    createGroup: (data: GroupCreateFormDialogData, setSubmitErr: (err: string) => void) => void;
    vocabulary: Vocabulary;
    allowSlash: boolean;
};

export const DialogProjectCreate = compose(
    connect(mapState, mapDispatch),
    withStyles(styles),
    withDialog(PROJECT_CREATE_FORM_NAME)
)(({ createProject, createGroup, data, closeDialog, open, vocabulary, allowSlash, classes }: DialogProjectProps & WithStyles<CssRules>) => {
    const [projectName, setProjectName, projectNameErrs] = useStateWithValidation('',
        [...REQUIRED_VALIDATION, ...(allowSlash ? PROJECT_NAME_VALIDATION_ALLOW_SLASH : PROJECT_NAME_VALIDATION)],
        'Project Name');
    const [description, setDescription, descriptionErrs] = useStateWithValidation('', MAXLENGTH_524288_VALIDATION, 'Description');
    const [chips, setChips] = useState<PropertyChips>({} as PropertyChips);
    const [users, setUsers] = useState<Participant[]>([]);
    const [formErrors, setFormErrors] = useState<string[]>([]);
    const [submitErr, setSubmitErr] = useState<string>('');
    const [isSubmitting, setIsSubmitting] = useState<boolean>(false);

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

    const sourcePanel = data?.sourcePanel || GroupClass.PROJECT;
    const isGroup = sourcePanel === GroupClass.ROLE;
    const title = isGroup ? 'New Group' : 'New Project';

    const fields = () => (
        <>
            <DialogTitle>{title}</DialogTitle>
            <DialogContent>
                {(isGroup === false) && <ResourceParentField ownerUuid={data ? data.ownerUuid : ''} />}
                <div data-cy="name-field">
                    <DialogTextField
                        label={isGroup ? "Group Name" : "Project Name"}
                        defaultValue={projectName}
                        setValue={setProjectName}
                        validators={allowSlash ? PROJECT_NAME_VALIDATION_ALLOW_SLASH : PROJECT_NAME_VALIDATION}
                        submitErr={submitErr}
                        setSubmitErr={setSubmitErr}
                    />
                </div>
                {isGroup && (
                    <div data-cy="users-field">
                        <ParticipantSelect
                            onlyPeople
                            label='Search for users to add to the group'
                            items={users}
                            onSelect={(user: Participant) => setUsers([...users, user])}
                            onDelete={(index: number) => setUsers(users.filter((_, i) => i !== index))}
                        />
                    </div>
                )}
                <div className={classes.description}>
                    <DialogRichTextField
                        label="Description"
                        defaultValue={description}
                        setValue={setDescription}
                        validators={PROJECT_DESCRIPTION_VALIDATION}
                    />
                </div>
                <div className={classes.propertiesForm}>
                    <FormLabel>Properties</FormLabel>
                    <FormGroup>
                        <DialogResourcePropertiesForm
                            setChips={setChips}
                            onSubmit={(ev) => ev.preventDefault()}
                        />
                    </FormGroup>
                </div>
            </DialogContent>
        </>
    );

    const submitFunc = () => {
        if (isGroup) {
            const groupData: GroupCreateFormDialogData = {
                name: projectName,
                description: description,
                properties: getVocabularyFromChips(chips, vocabulary),
                users,
            };
            createGroup(groupData, setSubmitErr);
        } else {
            const projectData: ProjectCreateFormDialogData = {
                ownerUuid: data.ownerUuid,
                name: projectName,
                description: description,
                properties: getVocabularyFromChips(chips, vocabulary),
            };
            createProject(projectData, setSubmitErr);
        }
    }

    return <DialogForm
        fields={fields()}
        submitLabel='Create'
        formErrors={formErrors}
        isSubmitting={isSubmitting}
        onSubmit={(ev) => {
            ev.preventDefault();
            setIsSubmitting(true);
            submitFunc();
        }}
        closeDialog={closeDialog}
        clearFormValues={() => {
            setProjectName('');
            setDescription('');
            setChips({} as PropertyChips);
            setUsers([]);
        }}
        open={open}
    />;
});
