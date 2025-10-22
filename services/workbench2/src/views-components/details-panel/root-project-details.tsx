// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { connect } from 'react-redux';
import { RootProjectIcon } from 'components/icon/icon';
import { formatDateTime } from 'common/formatters';
import { DetailsData } from "./details-data";
import { DetailsAttribute } from "components/details-attribute/details-attribute";
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { ArvadosTheme } from 'common/custom-theme';
import { Dispatch } from 'redux';
import { openProjectUpdateDialog, ProjectUpdateFormDialogData } from 'store/projects/project-update-actions';
import { RootState } from 'store/store';
import { ResourcesState } from 'store/resources/resources';
import { UserResource } from 'models/user';
import { UserResourceFullName } from 'views-components/data-explorer/renderers';

export class RootProjectDetails extends DetailsData<UserResource> {
    getIcon(className?: string) {
        return <RootProjectIcon className={className} />;
    }

    getDetails() {
        return <RootProjectDetailsComponent rootProject={this.item} />;
    }
}

type CssRules = 'tag' | 'editIcon' | 'editButton';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    tag: {
        marginRight: theme.spacing(0.5),
        marginBottom: theme.spacing(0.5),
    },
    editIcon: {
        paddingRight: theme.spacing(0.5),
        fontSize: '1.125rem',
    },
    editButton: {
        boxShadow: 'none',
        padding: '2px 10px 2px 5px',
        fontSize: '0.75rem'
    },
});

interface RootProjectDetailsComponentDataProps {
    rootProject: any;
}

const mapStateToProps = (state: RootState): { resources: ResourcesState } => {
    return {
        resources: state.resources
    };
};

const mapDispatchToProps = (dispatch: Dispatch) => ({
    onClick: (prj: ProjectUpdateFormDialogData) =>
        () => dispatch<any>(openProjectUpdateDialog(prj)),
});

type RootProjectDetailsComponentProps = RootProjectDetailsComponentDataProps & WithStyles<CssRules>;

export const RootProjectDetailsComponent = connect(mapStateToProps, mapDispatchToProps)(
    withStyles(styles)(
        ({ rootProject}: RootProjectDetailsComponentProps & { resources: ResourcesState }) => <div>
            <DetailsAttribute label='Type' value="Root Project" />
            <DetailsAttribute label='User' />
            <UserResourceFullName uuid={rootProject.uuid} link={true} />
            <DetailsAttribute label='Created at' value={formatDateTime(rootProject.createdAt)} />
            <DetailsAttribute label='UUID' linkToUuid={rootProject.uuid} value={rootProject.uuid} />
        </div>
    ));
