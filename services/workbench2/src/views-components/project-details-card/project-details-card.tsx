// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Card, CardHeader, WithStyles, withStyles, Typography, CardContent } from '@material-ui/core';
import { StyleRulesCallback } from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';
import { RootState } from 'store/store';
import { connect } from 'react-redux';
import { getResource } from 'store/resources/resources';
import { MultiselectToolbar } from 'components/multiselect-toolbar/MultiselectToolbar';
import { DetailsAttribute } from 'components/details-attribute/details-attribute';
import { RichTextEditorLink } from 'components/rich-text-editor-link/rich-text-editor-link';
import { getPropertyChip } from '../resource-properties-form/property-chip';
import { ProjectResource } from 'models/project';
import { GroupClass } from 'models/group';
import { ResourceWithName } from 'views-components/data-explorer/renderers';
import { formatDate } from 'common/formatters';
import { resourceLabel } from 'common/labels';
import { ResourceKind } from 'models/resource';
import { UserResource } from 'models/user';
import { UserResourceAccountStatus, FrozenProject } from 'views-components/data-explorer/renderers';
import { FavoriteStar, PublicFavoriteStar } from 'views-components/favorite-star/favorite-star';
import { FavoritesState } from 'store/favorites/favorites-reducer';
import { PublicFavoritesState } from 'store/public-favorites/public-favorites-reducer';

type CssRules = 'root' | 'cardheader' | 'fadeout' | 'nameContainer' | 'activeIndicator' | 'cardcontent' | 'attributesection' | 'attribute' | 'chipsection' | 'tag';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
        marginBottom: '1rem',
    },
    fadeout: {
        maxWidth: '25rem',
        minWdidth: '18rem',
        height: '1.5rem',
        overflow: 'hidden',
        WebkitMaskImage: '-webkit-gradient(linear, left bottom, right bottom, from(rgba(0,0,0,1)), to(rgba(0,0,0,0)))',
    },
    nameContainer: {
        display: 'flex',
    },
    activeIndicator: {
        margin: '0.3rem auto auto 1rem',
    },
    cardheader: {
        paddingTop: '0.4rem',
    },
    cardcontent: {
        display: 'flex',
        flexDirection: 'column',
        marginTop: '-1rem',
    },
    attributesection: {
        display: 'flex',
        flexWrap: 'wrap',
    },
    attribute: {
        marginBottom: '0.5rem',
        marginRight: '1rem',
        border: '1px solid lightgrey',
        padding: '0.5rem',
        borderRadius: '5px',
    },
    chipsection: {
        display: 'flex',
        flexWrap: 'wrap',
    },
    tag: {
        marginRight: '1rem',
        marginTop: '0.5rem',
    },
});

const mapStateToProps = (state: RootState) => {
    const currentRoute = state.router.location?.pathname.split('/') || [];
    const currentItemUuid = currentRoute[currentRoute.length - 1];
    const currentResource = getResource(currentItemUuid)(state.resources);
    return {
        currentResource,
    };
};

type DetailsCardProps = WithStyles<CssRules> & {
    currentResource: ProjectResource | UserResource;
};

type UserCardProps = WithStyles<CssRules> & {
    currentResource: UserResource;
};

type ProjectCardProps = WithStyles<CssRules> & {
    currentResource: ProjectResource;
};

export const ProjectDetailsCard = connect(mapStateToProps)(
    withStyles(styles)((props: DetailsCardProps) => {
        const { classes, currentResource } = props;
        switch (currentResource.kind as string) {
            case ResourceKind.USER:
                return <UserCard classes={classes} currentResource={currentResource as UserResource} />;
            case ResourceKind.PROJECT:
                return <ProjectCard classes={classes} currentResource={currentResource as ProjectResource} />;
            default:
                return null;
        }
    })
);

const UserCard: React.FC<UserCardProps> = ({ classes, currentResource }) => {
    const { fullName, uuid, username, email, isAdmin } = currentResource as UserResource & { fullName: string };

    return (
        <Card className={classes.root}>
            <CardHeader
                className={classes.cardheader}
                title={
                    <section className={classes.nameContainer}>
                        <Typography
                            noWrap
                            variant='h6'
                        >
                            {fullName}
                        </Typography>
                        <Typography className={classes.activeIndicator}>
                            <UserResourceAccountStatus uuid={uuid} />
                        </Typography>
                    </section>
                }
                action={<MultiselectToolbar inputSelectedUuid={uuid} />}
            />
            <CardContent className={classes.cardcontent}>
                <section className={classes.attributesection}>
                    <Typography
                        component='div'
                        className={classes.attribute}
                    >
                        <DetailsAttribute
                            label='Username'
                            value={username}
                        />
                    </Typography>
                    <Typography
                        component='div'
                        className={classes.attribute}
                    >
                        <DetailsAttribute
                            label='Email'
                            value={email}
                        />
                    </Typography>
                    <Typography
                        component='div'
                        className={classes.attribute}
                    >
                        <DetailsAttribute
                            label='Admin'
                            value={isAdmin ? 'Yes' : 'No'}
                        />
                    </Typography>
                    <Typography
                        component='div'
                        className={classes.attribute}
                    >
                        <DetailsAttribute
                            label='UUID'
                            linkToUuid={currentResource.uuid}
                            value={currentResource.uuid}
                        />
                    </Typography>
                </section>
            </CardContent>
        </Card>
    );
};

const ProjectCard: React.FC<ProjectCardProps> = ({ classes, currentResource }) => {
    const { name, uuid, description } = currentResource as ProjectResource;
    return (
        <Card className={classes.root}>
            <CardHeader
                className={classes.cardheader}
                title={
                    <Typography
                        noWrap
                        variant='h6'
                    >
                        {name}
                        <FavoriteStar resourceUuid={currentResource.uuid} />
                        <PublicFavoriteStar resourceUuid={currentResource.uuid} />
                        {currentResource.kind === ResourceKind.PROJECT && <FrozenProject item={currentResource} />}
                    </Typography>
                }
                subheader={
                    description ? (
                        <section>
                            <Typography className={classes.fadeout}>{description.replace(/<[^>]*>/g, '').slice(0, 45)}...</Typography>
                            <RichTextEditorLink
                                title={`Description of ${name}`}
                                content={description}
                                label='Show full description'
                            />
                        </section>
                    ) : (
                        'no description available'
                    )
                }
                action={<MultiselectToolbar inputSelectedUuid={uuid} />}
            />
            <CardContent className={classes.cardcontent}>
                <section className={classes.attributesection}>
                    <Typography
                        component='div'
                        className={classes.attribute}
                    >
                        <DetailsAttribute
                            label='Type'
                            value={'groupClass' in currentResource && currentResource.groupClass === GroupClass.FILTER ? 'Filter group' : resourceLabel(ResourceKind.PROJECT)}
                        />
                    </Typography>
                    <Typography
                        component='div'
                        className={classes.attribute}
                    >
                        <DetailsAttribute
                            label='Owner'
                            linkToUuid={currentResource.ownerUuid}
                            uuidEnhancer={(uuid: string) => <ResourceWithName uuid={uuid} />}
                        />
                    </Typography>
                    <Typography
                        component='div'
                        className={classes.attribute}
                    >
                        <DetailsAttribute
                            label='Last modified'
                            value={formatDate(currentResource.modifiedAt)}
                        />
                    </Typography>
                    <Typography
                        component='div'
                        className={classes.attribute}
                    >
                        <DetailsAttribute
                            label='Created at'
                            value={formatDate(currentResource.createdAt)}
                        />
                    </Typography>
                    <Typography
                        component='div'
                        className={classes.attribute}
                    >
                        <DetailsAttribute
                            label='UUID'
                            linkToUuid={currentResource.uuid}
                            value={currentResource.uuid}
                        />
                    </Typography>
                </section>
                <section className={classes.chipsection}>
                    <Typography component='div'>
                        {'properties' in currentResource &&
                            typeof currentResource.properties === 'object' &&
                            Object.keys(currentResource.properties).map((k) =>
                                Array.isArray(currentResource.properties[k])
                                    ? currentResource.properties[k].map((v: string) => getPropertyChip(k, v, undefined, classes.tag))
                                    : getPropertyChip(k, currentResource.properties[k], undefined, classes.tag)
                            )}
                    </Typography>
                </section>
            </CardContent>
        </Card>
    );
};
