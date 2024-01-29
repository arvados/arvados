// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Card, CardHeader, WithStyles, withStyles, Typography, CardContent, Tooltip } from '@material-ui/core';
import { StyleRulesCallback } from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';
import { RootState } from 'store/store';
import { connect } from 'react-redux';
import { getResource } from 'store/resources/resources';
import { MultiselectToolbar } from 'components/multiselect-toolbar/MultiselectToolbar';
import { getPropertyChip } from '../resource-properties-form/property-chip';
import { ProjectResource } from 'models/project';
import { ResourceKind } from 'models/resource';
import { UserResource } from 'models/user';
import { UserResourceAccountStatus } from 'views-components/data-explorer/renderers';
import { FavoriteStar, PublicFavoriteStar } from 'views-components/favorite-star/favorite-star';
import { FreezeIcon } from 'components/icon/icon';
import { Resource } from 'models/resource';
import { MoreVerticalIcon } from 'components/icon/icon';
import { IconButton } from '@material-ui/core';
import { ContextMenuResource } from 'store/context-menu/context-menu-actions';
import { resourceUuidToContextMenuKind } from 'store/context-menu/context-menu-actions';
import { openContextMenu } from 'store/context-menu/context-menu-actions'; 
import { CollectionResource } from 'models/collection';
import { RichTextEditorLink } from 'components/rich-text-editor-link/rich-text-editor-link';    

type CssRules =
    | 'root'
    | 'cardheader'
    | 'fadeout'
    | 'showmore'
    | 'nameContainer'
    | 'activeIndicator'
    | 'cardcontent'
    | 'namePlate'
    | 'faveIcon'
    | 'frozenIcon'
    | 'attributesection'
    | 'attribute'
    | 'chipsection'
    | 'tag';

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
    showmore: {
        color: theme.palette.primary.main,
        cursor: 'pointer',
        maxWidth: '10rem',
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
    namePlate: {
        display: 'flex',
        flexDirection: 'row',
    },
    faveIcon: {
        fontSize: '0.8rem',
        margin: 'auto 0 0.5rem 0.3rem',
        color: theme.palette.text.primary,
    },
    frozenIcon: {
        fontSize: '0.5rem',
        marginLeft: '0.3rem',
        marginTop: '0.57rem',
        height: '1rem',
        color: theme.palette.text.primary,
    },
    attributesection: {
        display: 'flex',
        flexWrap: 'wrap',
    },
    attribute: {
        marginBottom: '0.5rem',
        marginRight: '1rem',
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
    const frozenByUser = currentResource && getResource((currentResource as ProjectResource).frozenByUuid as string)(state.resources);
    const frozenByFullName = frozenByUser && (frozenByUser as Resource & { fullName: string }).fullName;

    return {
        isAdmin: state.auth.user?.isAdmin,
        currentResource,
        frozenByFullName,
    };
};

const mapDispatchToProps = (dispatch: any) => ({
    handleContextMenu: (event: React.MouseEvent<HTMLElement>, resource: any, isAdmin: boolean) => {
        // When viewing the contents of a filter group, all contents should be treated as read only.
        let readOnly = false;
        if (resource.groupClass === 'filter') {
            readOnly = true;
        }

        const menuKind = dispatch(resourceUuidToContextMenuKind(resource.uuid, readOnly));
        if (menuKind && resource) {
            dispatch(
                openContextMenu(event, {
                    name: resource.name,
                    uuid: resource.uuid,
                    ownerUuid: resource.ownerUuid,
                    isTrashed: 'isTrashed' in resource ? resource.isTrashed : false,
                    kind: resource.kind,
                    menuKind,
                    isAdmin,
                    isFrozen: !!resource.frozenByUuid,
                    description: resource.description,
                    storageClassesDesired: (resource as CollectionResource).storageClassesDesired,
                    properties: 'properties' in resource ? resource.properties : {},
                })
            );
        }

    },
});

type DetailsCardProps = WithStyles<CssRules> & {
    currentResource: ProjectResource | UserResource;
    frozenByFullName?: string;
    isAdmin: boolean;
    handleContextMenu: (event: React.MouseEvent<HTMLElement>, resource: ContextMenuResource, isAdmin: boolean) => void;
};

type UserCardProps = WithStyles<CssRules> & {
    currentResource: UserResource;
};

type ProjectCardProps = WithStyles<CssRules> & {
    currentResource: ProjectResource;
    frozenByFullName: string | undefined;
    isAdmin: boolean;
    handleContextMenu: (event: React.MouseEvent<HTMLElement>, resource: ContextMenuResource, isAdmin: boolean) => void;
};

export const ProjectDetailsCard = connect(
    mapStateToProps,
    mapDispatchToProps
)(
    withStyles(styles)((props: DetailsCardProps) => {
        const { classes, currentResource, frozenByFullName, handleContextMenu, isAdmin } = props;
        switch (currentResource.kind as string) {
            case ResourceKind.USER:
                return (
                    <UserCard
                        classes={classes}
                        currentResource={currentResource as UserResource}
                    />
                );
            case ResourceKind.PROJECT:
                return (
                    <ProjectCard
                        classes={classes}
                        currentResource={currentResource as ProjectResource}
                        frozenByFullName={frozenByFullName}
                        isAdmin={isAdmin}
                        handleContextMenu={(ev) => handleContextMenu(ev, currentResource as any, isAdmin)}
                    />
                );
            default:
                return null;
        }
    })
);

const UserCard: React.FC<UserCardProps> = ({ classes, currentResource }) => {
    const { fullName, uuid } = currentResource as UserResource & { fullName: string };

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
                        {!currentResource.isActive && (
                            <Typography className={classes.activeIndicator}>
                                <UserResourceAccountStatus uuid={uuid} />
                            </Typography>
                        )}
                    </section>
                }
                action={<MultiselectToolbar inputSelectedUuid={uuid} />}
            />
        </Card>
    );
};

const ProjectCard: React.FC<ProjectCardProps> = ({ classes, currentResource, frozenByFullName, handleContextMenu, isAdmin }) => {
    const { name, uuid, description } = currentResource as ProjectResource;

    return (
        <Card className={classes.root}>
            <CardHeader
                className={classes.cardheader}
                title={
                    <>
                    <section className={classes.namePlate}>
                        <Typography
                            noWrap
                            variant='h6'
                            style={{ marginRight: '1rem' }}
                        >
                            {name}
                        </Typography>
                        <FavoriteStar
                            className={classes.faveIcon}
                            resourceUuid={currentResource.uuid}
                        />
                        <PublicFavoriteStar
                            className={classes.faveIcon}
                            resourceUuid={currentResource.uuid}
                        />
                        {!!frozenByFullName && <Tooltip
                            className={classes.frozenIcon}
                            title={<span>Project was frozen by {frozenByFullName}</span>}
                        >
                            <FreezeIcon style={{ fontSize: 'inherit' }} />
                        </Tooltip>}
                    </section>
                    <section className={classes.chipsection}>
                    <Typography component='div'>
                        {typeof currentResource.properties === 'object' &&
                            Object.keys(currentResource.properties).map((k) =>
                                Array.isArray(currentResource.properties[k])
                                    ? currentResource.properties[k].map((v: string) => getPropertyChip(k, v, undefined, classes.tag))
                                    : getPropertyChip(k, currentResource.properties[k], undefined, classes.tag)
                            )}
                    </Typography>
                </section>
                    </>
                }
                    
                    
                action={<Tooltip
                    title='More options'
                    disableFocusListener
                >
                    <IconButton
                        aria-label='More options'
                        onClick={(ev) => handleContextMenu(ev, currentResource as any, isAdmin)}
                    >
                        <MoreVerticalIcon />
                    </IconButton>
                </Tooltip>}
            />
            <CardContent className={classes.cardcontent}>
                {description && (
                        <section>
                            {/* <Typography className={classes.fadeout}>{description.replace(/<[^>]*>/g, '').slice(0, 45)}...</Typography> */}
                            <div className={classes.showmore}>
                                <RichTextEditorLink
                                    title={`Description of ${name}`}
                                    content={description}
                                    label='Show full description'
                                />
                            </div>
                        </section>
                    )}
            </CardContent>
        </Card>
    );
};
