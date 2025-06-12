// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Dispatch } from 'redux';
import { Link } from 'react-router-dom';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { Grid, Typography, Tooltip, Link as ButtonLink } from '@mui/material';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { connect } from "react-redux";
import { ArvadosTheme } from 'common/custom-theme';
import { RootState } from 'store/store';
import { DetailsAttribute } from 'components/details-attribute/details-attribute';
import { getResource } from 'store/resources/resources';
import { formatDate, formatFileSize } from "common/formatters";
import { ResourceWithName, RenderResponsiblePerson } from 'views-components/data-explorer/renderers';
import { GroupContentsResource } from 'services/groups-service/groups-service';
import { getUserFullname, UserResource } from 'models/user';
import { Resource, ResourceKind } from 'models/resource';
import { navigateToProcess } from 'store/collection-panel/collection-panel-action';
import { CollectionResource, getCollectionUrl } from 'models/collection';
import { openDetailsPanel } from 'store/details-panel/details-panel-action';

type CssRules = 'label' | 'value' | 'link' | 'button' | 'warningLabel'

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    label: {
        fontSize: '0.875rem',
    },
    value: {
        textTransform: 'none',
        fontSize: '0.875rem'
    },
    link: {
        fontSize: '0.875rem',
        color: theme.palette.primary.main,
        '&:hover': {
            cursor: 'pointer'
        }
    },
    button: {
        cursor: 'pointer'
    },
    warningLabel: {
        fontStyle: 'italic'
    },
});

const mapStateToProps = (state: RootState): Omit<CollectionAttributesProps, 'navigateToProcess' | 'showVersionBrowser'> => {
    const item = getResource<CollectionResource>(state.properties.currentRouteUuid)(state.resources);
    const { responsiblePersonUUID, responsiblePersonName } = getResponsibleData(state, item?.uuid);
    return {
        item, responsiblePersonUUID, responsiblePersonName
    };
};

const mapDispatchToProps = (dispatch: Dispatch): Pick<CollectionAttributesProps, 'navigateToProcess' | 'showVersionBrowser'> => ({
    navigateToProcess: (uuid: string) => dispatch<any>(navigateToProcess(uuid)),
    showVersionBrowser: (item: CollectionResource) => dispatch<any>(openDetailsPanel(item.uuid, 1))
});


interface CollectionAttributesProps {
    item?: CollectionResource;
    responsiblePersonUUID: string;
    responsiblePersonName: string;
    navigateToProcess: (uuid: string) => void;
    showVersionBrowser: (item: CollectionResource) => void;
}

export const CollectionAttributes = connect(mapStateToProps, mapDispatchToProps)(withStyles(styles)((props: CollectionAttributesProps & WithStyles<CssRules>) => {
    if (!props.item) {
        return null;
    }
    const { item, classes, responsiblePersonUUID, responsiblePersonName } = props;
    const isOldVersion = item && item.currentVersionUuid !== item.uuid;
    const mdSize = 4;
    return <Grid container>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                label={isOldVersion ? "This version's UUID" : "Collection UUID"}
                linkToUuid={item.uuid} />
        </Grid>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                label={isOldVersion ? "This version's PDH" : "Portable data hash"}
                linkToUuid={item.portableDataHash} />
        </Grid>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                label='Owner' linkToUuid={item.ownerUuid}
                uuidEnhancer={(uuid: string) => <ResourceWithName uuid={uuid} />} />
        </Grid>
        {responsiblePersonUUID && <Grid item xs={12} md={mdSize} data-cy="responsible-person-wrapper">
            <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                label='Responsible person' linkToUuid={item.ownerUuid}
                uuidEnhancer={(uuid: string) => <RenderResponsiblePerson responsiblePersonUUID={responsiblePersonUUID} responsiblePersonName={responsiblePersonName} />} />
        </Grid>}
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                label='Head version'
                value={isOldVersion ? undefined : 'this one'}
                linkToUuid={isOldVersion ? item.currentVersionUuid : undefined} />
        </Grid>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute
                classLabel={classes.label} classValue={classes.value}
                label='Version number'
                value={
                <Tooltip title="Open version browser">
                    <ButtonLink underline='none' className={classes.button} onClick={() => props.showVersionBrowser(item)}>
                        <span data-cy='collection-version-number'>{item.version}</span>
                    </ButtonLink>
                </Tooltip>
                }
            />
        </Grid>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute label='Created at' value={formatDate(item.createdAt)} />
        </Grid>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute label='Last modified' value={formatDate(item.modifiedAt)} />
        </Grid>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                label='Number of files' value={<span data-cy='collection-file-count'>{item.fileCount}</span>} />
        </Grid>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                label='Content size' value={formatFileSize(item.fileSizeTotal)} />
        </Grid>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                label='Storage classes' value={item.storageClassesDesired ? item.storageClassesDesired.join(', ') : ["default"]} />
        </Grid>
        {(item.properties.container_request || item.properties.containerRequest) &&
            <Grid item xs={12} md={mdSize}
                onClick={() => props.navigateToProcess(item.properties.container_request || item.properties.containerRequest)}>
                <DetailsAttribute classLabel={classes.link} label='Link to process' />
            </Grid>
        }
        {isOldVersion &&
            <Grid item xs={12} md={12}>
                <Typography className={classes.warningLabel} variant="caption">
                    This is an old version. Make a copy to make changes. Go to the <Link to={getCollectionUrl(item.currentVersionUuid)}>head version</Link> for sharing options.
                </Typography>
            </Grid>
        }
    </Grid>;
}));

const getResponsibleData = (state: RootState, uuid: string | undefined) => {
        let responsiblePersonName: string = "";
        let responsiblePersonUUID: string = "";
        let responsiblePersonProperty: string = "";

        if (state.auth.config.clusterConfig.Collections.ManagedProperties) {
            let index = 0;
            const keys = Object.keys(state.auth.config.clusterConfig.Collections.ManagedProperties);

            while (!responsiblePersonProperty && keys[index]) {
                const key = keys[index];
                if (state.auth.config.clusterConfig.Collections.ManagedProperties[key].Function === "original_owner") {
                    responsiblePersonProperty = key;
                }
                index++;
            }
        }

        let resource: Resource | undefined = getResource<GroupContentsResource & UserResource>(uuid)(state.resources);

        while (resource && resource.kind !== ResourceKind.USER && responsiblePersonProperty) {
            responsiblePersonUUID = (resource as CollectionResource).properties[responsiblePersonProperty];
            resource = getResource<GroupContentsResource & UserResource>(responsiblePersonUUID)(state.resources);
        }

        if (resource && resource.kind === ResourceKind.USER) {
            responsiblePersonName = getUserFullname(resource as UserResource) || (resource as GroupContentsResource).name;
        }

        return { responsiblePersonUUID, responsiblePersonName, };
    }
