// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useState } from 'react';
import { connect } from 'react-redux';
import { Grid, Typography } from '@mui/material';
import { RootState } from 'store/store';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { ArvadosTheme } from 'common/custom-theme';
import withStyles from '@mui/styles/withStyles';
import { WithStyles } from '@mui/styles';
import { getResource } from 'store/resources/resources';
import { DetailsAttribute } from 'components/details-attribute/details-attribute';
import { getPropertyChip } from 'views-components/resource-properties-form/property-chip';
import { ExpandChevronRight } from 'components/expand-chevron-right/expand-chevron-right';
import { CollapsibleDescription } from 'components/collapsible-description/collapsible-description';
import { CollectionResource } from 'models/collection';
import { ProjectResource } from 'models/project';
import { WorkflowResource } from 'models/workflow';
import { ResourceKind } from 'models/resource';
import { Process, getProcess } from 'store/processes/process';
import { ContainerRequestResource } from 'models/container-request';
import { ContainerResource } from 'models/container';
import { ProcessRuntimeStatus } from 'views-components/process-runtime-status/process-runtime-status';
import { isUserResource } from 'models/user';
import { getRegisteredWorkflowPanelData } from 'views-components/details-panel/workflow-details';
import { AuthState } from 'store/auth/auth-reducer';
import { DataTableDefaultView } from 'components/data-table-default-view/data-table-default-view';

type CssRules = 'root' | 'tag';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        justifyContent: 'space-between',
        padding: theme.spacing(1),
    },
    tag: {
        marginRight: theme.spacing(0.5),
        marginBottom: theme.spacing(0.5),
    },
});

type OverviewPanelProps = {
    auth: AuthState;
    resource: ProjectResource | CollectionResource | ContainerRequestResource | WorkflowResource | undefined;
    process?: Process;
    container?: ContainerResource;
    detailsElement: React.ReactNode;
    progressIndicator: string[];
} & WithStyles<CssRules>;

const mapStateToProps = (state: RootState): Pick<OverviewPanelProps, 'auth' |'resource' | 'container' | 'progressIndicator'> => {
    const resource = getResource<any>(state.properties.currentRouteUuid)(state.resources);
    const process = getProcess(resource?.uuid)(state.resources) || undefined;
    return {
        auth: state.auth,
        resource: resource?.containerRequest ? process : resource,
        container: process?.container,
        progressIndicator: state.progressIndicator
    };
};

export const OverviewPanel = connect(mapStateToProps)(withStyles(styles)((({ auth,resource, container, detailsElement, progressIndicator, classes }: OverviewPanelProps) => {
    const working = progressIndicator.length > 0;
    if (isUserResource(resource)) {
        return null
    };
    if (!resource) {
        if (!working) {
            return <DataTableDefaultView />
        };
        return null;
    }
    const hasDescription = resource.description && resource.description.length > 0;
    const [showDescription, setShowDescription] = useState(false);

    React.useEffect(() => {
        setShowDescription(false);
    }, [resource]);

    return (
        <section className={classes.root}>
            <section>
                {resource.kind === ResourceKind.CONTAINER_REQUEST && <Grid item xs={12}>
                    <ProcessRuntimeStatus runtimeStatus={container?.runtimeStatus} containerCount={resource.containerCount} />
                </Grid>}
                <Grid item xs={12} md={12}>
                    <DetailsAttribute
                        label={'Description'}
                        button={hasDescription
                                    ? <ExpandChevronRight expanded={showDescription} onClick={() => setShowDescription(!showDescription)} />
                                    : undefined}>
                        {hasDescription
                            ? <CollapsibleDescription description={resource.description} showDescription={showDescription} />
                            : <Typography>No description available</Typography>}
                    </DetailsAttribute>
                    <section data-cy='details-element'>
                        {detailsElement}
                    </section>
                </Grid>
            </section>
            <PropertiesElement auth={auth} resource={resource} classes={classes} />
        </section>
    );
})));

const PropertiesElement = ({auth, resource, classes}: { auth: AuthState, resource: ProjectResource | CollectionResource | ContainerRequestResource | WorkflowResource | undefined, classes: any }) => {
    if (!resource) {
        return null;
    }
    if (resource.kind === ResourceKind.WORKFLOW) {
        const wfData = getRegisteredWorkflowPanelData(resource, auth);
        if (Object.keys(wfData.gitprops).length === 0) {
            return null;
        }
        return <section data-cy='resource-properties'>
            {Object.keys(wfData.gitprops).map(k =>
                getPropertyChip(k, wfData.gitprops[k], undefined, classes.tag)
            )}
        </section>;
    }
    if (typeof resource.properties === 'object' && Object.keys(resource.properties).length > 0) {
        // remove cwl_input and cwl_output which are displayed in their own tabs
        const properties = {...resource.properties};
        delete properties['cwl_input'];
        delete properties['cwl_output'];

        return <section data-cy='resource-properties'>
            {Object.keys(properties).map((k) =>
                Array.isArray(properties[k])
                ? properties[k].map((v: string) => getPropertyChip(k, v, undefined, classes.tag))
                : getPropertyChip(k, properties[k], undefined, classes.tag)
            )}
        </section>;
    }
    return null;
}
