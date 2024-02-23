// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { StyleRulesCallback, Card, CardHeader, WithStyles, withStyles, Typography, CardContent, Tooltip, Collapse } from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';
import { RootState } from 'store/store';
import { connect } from 'react-redux';
import { getResource } from 'store/resources/resources';
import { getPropertyChip } from '../resource-properties-form/property-chip';
import { ProjectResource } from 'models/project';
import { ResourceKind } from 'models/resource';
import { UserResource } from 'models/user';
import { UserResourceAccountStatus } from 'views-components/data-explorer/renderers';
import { FavoriteStar, PublicFavoriteStar } from 'views-components/favorite-star/favorite-star';
import { MoreVerticalIcon, FreezeIcon } from 'components/icon/icon';
import { Resource } from 'models/resource';
import { IconButton } from '@material-ui/core';
import { ContextMenuResource, openUserContextMenu } from 'store/context-menu/context-menu-actions';
import { openContextMenu, resourceUuidToContextMenuKind } from 'store/context-menu/context-menu-actions';
import { CollectionResource } from 'models/collection';
import { ContextMenuKind } from 'views-components/context-menu/context-menu';
import { Dispatch } from 'redux';
import classNames from 'classnames';
import { loadDetailsPanel } from 'store/details-panel/details-panel-action';

type CssRules =
    | 'root'
    | 'selected'
    | 'cardHeader'
    | 'descriptionLabel'
    | 'showMore'
    | 'noDescription'
    | 'nameContainer'
    | 'cardContent'
    | 'subHeader'
    | 'namePlate'
    | 'faveIcon'
    | 'frozenIcon'
    | 'contextMenuSection'
    | 'chipSection'
    | 'tag'
    | 'description';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
        marginBottom: '1rem',
        flex: '0 0 auto',
        paddingTop: '0.2rem',
        border: '2px solid transparent',
    },
    selected: {
        border: '2px solid #ccc',
    },
    showMore: {
        color: theme.palette.primary.main,
        cursor: 'pointer',
    },
    noDescription: {
        color: theme.palette.grey['600'],
        fontStyle: 'italic',
    },
    nameContainer: {
        display: 'flex',
    },
    cardHeader: {
        paddingTop: '0.4rem',
    },
    descriptionLabel: {
        paddingTop: '1rem',
        marginBottom: 0,
        minHeight: '2.5rem',
        marginRight: '0.8rem',
    },
    cardContent: {
        display: 'flex',
        flexDirection: 'column',
        transition: 'height 0.3s ease',
    },
    subHeader: {
        display: 'flex',
        flexDirection: 'row',
        justifyContent: 'space-between',
        marginTop: '-2rem',
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
    contextMenuSection: {
        display: 'flex',
        flexDirection: 'row',
        alignItems: 'center',
        marginTop: '0.6rem',
    },
    chipSection: {
        display: 'flex',
        flexWrap: 'wrap',
    },
    tag: {
        marginRight: '1rem',
        marginTop: '0.5rem',
    },
    description: {
        marginTop: '1rem',
    },
});

const mapStateToProps = (state: RootState) => {
    const currentRoute = state.router.location?.pathname.split('/') || [];
    const currentItemUuid = currentRoute[currentRoute.length - 1];
    const currentResource = getResource(currentItemUuid)(state.resources);
    const frozenByUser = currentResource && getResource((currentResource as ProjectResource).frozenByUuid as string)(state.resources);
    const frozenByFullName = frozenByUser && (frozenByUser as Resource & { fullName: string }).fullName;
    const isSelected = currentItemUuid === state.detailsPanel.resourceUuid && state.detailsPanel.isOpened === true;

    return {
        isAdmin: state.auth.user?.isAdmin,
        currentResource,
        frozenByFullName,
        isSelected,
    };
};

const mapDispatchToProps = (dispatch: Dispatch) => ({
    handleCardClick: (uuid: string) => {
        dispatch<any>(loadDetailsPanel(uuid));
    },
    handleContextMenu: (event: React.MouseEvent<HTMLElement>, resource: any, isAdmin: boolean) => {
        event.stopPropagation();
        // When viewing the contents of a filter group, all contents should be treated as read only.
        let readOnly = false;
        if (resource.groupClass === 'filter') {
            readOnly = true;
        }
        let menuKind = dispatch<any>(resourceUuidToContextMenuKind(resource.uuid, readOnly));
        if (menuKind === ContextMenuKind.ROOT_PROJECT) {
            menuKind = ContextMenuKind.USER_DETAILS;
        }
        if (menuKind && resource) {
            dispatch<any>(
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
    isSelected: boolean;
    handleContextMenu: (event: React.MouseEvent<HTMLElement>, resource: ContextMenuResource, isAdmin: boolean) => void;
    handleCardClick: (resource: any) => void;
};

type UserCardProps = WithStyles<CssRules> & {
    currentResource: UserResource;
    isAdmin: boolean;
    isSelected: boolean;
    handleContextMenu: (event: React.MouseEvent<HTMLElement>, resource: ContextMenuResource, isAdmin: boolean) => void;
    handleCardClick: (resource: any) => void;
};

type ProjectCardProps = WithStyles<CssRules> & {
    currentResource: ProjectResource;
    frozenByFullName: string | undefined;
    isAdmin: boolean;
    isSelected: boolean;
    handleContextMenu: (event: React.MouseEvent<HTMLElement>, resource: ContextMenuResource, isAdmin: boolean) => void;
    handleCardClick: (resource: any) => void;
};

export const ProjectDetailsCard = connect(
    mapStateToProps,
    mapDispatchToProps
)(
    withStyles(styles)((props: DetailsCardProps) => {
        const { classes, currentResource, frozenByFullName, handleContextMenu, handleCardClick, isAdmin, isSelected } = props;
        if (!currentResource) {
            return null;
        }
        switch (currentResource.kind as string) {
            case ResourceKind.USER:
                return (
                    <UserCard
                        classes={classes}
                        currentResource={currentResource as UserResource}
                        isAdmin={isAdmin}
                        isSelected={isSelected}
                        handleContextMenu={(ev) => handleContextMenu(ev, currentResource as any, isAdmin)}
                        handleCardClick={handleCardClick}
                    />
                );
            case ResourceKind.PROJECT:
                return (
                    <ProjectCard
                        classes={classes}
                        currentResource={currentResource as ProjectResource}
                        frozenByFullName={frozenByFullName}
                        isAdmin={isAdmin}
                        isSelected={isSelected}
                        handleContextMenu={(ev) => handleContextMenu(ev, currentResource as any, isAdmin)}
                        handleCardClick={handleCardClick}
                    />
                );
            default:
                return null;
        }
    })
);

const UserCard: React.FC<UserCardProps> = ({ classes, currentResource, handleContextMenu, handleCardClick, isAdmin, isSelected }) => {
    const { fullName, uuid } = currentResource as UserResource & { fullName: string };

    return (
        <Card className={classNames(classes.root, isSelected ? classes.selected : '')} onClick={()=>handleCardClick(uuid)} data-cy='user-details-card'>
            <CardHeader
                className={classes.cardHeader}
                title={
                    <section className={classes.nameContainer}>
                        <Typography
                            noWrap
                            variant='h6'
                        >
                            {fullName}
                        </Typography>
                    </section>
                }
                action={
                    <section className={classes.contextMenuSection}>
                        {!currentResource.isActive && (
                            <Typography>
                                <UserResourceAccountStatus uuid={uuid} />
                            </Typography>
                        )}
                        <Tooltip
                            title='More options'
                            disableFocusListener
                        >
                            <IconButton
                                aria-label='More options'
                                data-cy='kebab-icon'
                                onClick={(ev) => handleContextMenu(ev, currentResource as any, isAdmin)}
                            >
                                <MoreVerticalIcon />
                            </IconButton>
                        </Tooltip>
                    </section>
                }
            />
        </Card>
    );
};

const ProjectCard: React.FC<ProjectCardProps> = ({ classes, currentResource, frozenByFullName, handleContextMenu, handleCardClick, isAdmin, isSelected }) => {
    const { name, description, uuid } = currentResource as ProjectResource;
    const [showDescription, setShowDescription] = React.useState(false);

    const toggleDescription = () => {
        setShowDescription(!showDescription);
    };

    return (
        <Card className={classNames(classes.root, isSelected ? classes.selected : '')} onClick={()=>handleCardClick(uuid)} data-cy='project-details-card'>
            <CardHeader
                className={classes.cardHeader}
                title={
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
                        {!!frozenByFullName && (
                            <Tooltip
                                className={classes.frozenIcon}
                                disableFocusListener
                                title={<span>Project was frozen by {frozenByFullName}</span>}
                            >
                                <FreezeIcon style={{ fontSize: 'inherit' }} />
                            </Tooltip>
                        )}
                    </section>
                }
                action={
                    <section className={classes.contextMenuSection}>
                        <Tooltip
                            title='More options'
                            disableFocusListener
                            >
                            <IconButton
                                aria-label='More options'
                                onClick={(ev) => handleContextMenu(ev, currentResource as any, isAdmin)}
                            >
                                <MoreVerticalIcon data-cy='kebab-icon' />
                            </IconButton>
                        </Tooltip>
                    </section>
                }
            />
            <CardContent className={classes.cardContent}>
                <section className={classes.subHeader}>
                    <section className={classes.chipSection}>
                        <Typography component='div'>
                            {typeof currentResource.properties === 'object' &&
                                Object.keys(currentResource.properties).map((k) =>
                                    Array.isArray(currentResource.properties[k])
                                        ? currentResource.properties[k].map((v: string) => getPropertyChip(k, v, undefined, classes.tag))
                                        : getPropertyChip(k, currentResource.properties[k], undefined, classes.tag)
                                )}
                        </Typography>
                    </section>
                    <section className={classes.descriptionLabel} onClick={(ev)=>ev.stopPropagation()}>
                        {description ? (
                            <Typography
                                className={classes.showMore}
                                onClick={toggleDescription}
                                data-cy='toggle-description'
                            >
                                {!showDescription ? "Show full description" : "Hide full description"}
                            </Typography>
                        ) : (
                            <Typography className={classes.noDescription} data-cy="no-description">no description available</Typography>
                        )}
                    </section>
                </section>
                <Collapse in={showDescription} timeout='auto'>
                    <section onClick={(ev)=>ev.stopPropagation()}>
                        <Typography className={classes.description} data-cy='project-description'>
                            {description}
                        </Typography>
                    </section>
                </Collapse>
            </CardContent>
        </Card>
    );
};
