// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { StyleRulesCallback, Card, CardHeader, WithStyles, withStyles, Typography, CardContent, Tooltip, Collapse, Grid } from '@material-ui/core';
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
import { FreezeIcon } from 'components/icon/icon';
import { Resource } from 'models/resource';
import { ContextMenuResource } from 'store/context-menu/context-menu-actions';
import { openContextMenu, resourceUuidToContextMenuKind } from 'store/context-menu/context-menu-actions';
import { CollectionResource } from 'models/collection';
import { ContextMenuKind } from 'views-components/context-menu/context-menu';
import { Dispatch } from 'redux';
import { loadDetailsPanel } from 'store/details-panel/details-panel-action';
import { ExpandChevronRight } from 'components/expand-chevron-right/expand-chevron-right';
import { MultiselectToolbar } from 'components/multiselect-toolbar/MultiselectToolbar';
import { setSelectedResourceUuid } from 'store/selected-resource/selected-resource-actions';
import { deselectAllOthers } from 'store/multiselect/multiselect-actions';

type CssRules =
    | 'root'
    | 'cardHeaderContainer'
    | 'cardHeader'
    | 'descriptionToggle'
    | 'showMore'
    | 'noDescription'
    | 'userNameContainer'
    | 'cardContent'
    | 'nameSection'
    | 'namePlate'
    | 'faveIcon'
    | 'frozenIcon'
    | 'accountStatusSection'
    | 'chipSection'
    | 'tag'
    | 'description';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
        marginBottom: '1rem',
        flex: '0 0 auto',
        padding: 0,
        minHeight: '3rem',
    },
    showMore: {
        cursor: 'pointer',
    },
    noDescription: {
        color: theme.palette.grey['600'],
        fontStyle: 'italic',
        padding: '0  0 0.5rem 1rem',
        marginTop: '-0.5rem',
    },
    userNameContainer: {
        display: 'flex',
        alignItems: 'center',
        minHeight: '2.7rem',
    },
    cardHeaderContainer: {
        width: '100%',
        display: 'flex',
        flexDirection: 'row',
        justifyContent: 'space-between',
    },
    cardHeader: {
        minWidth: '30rem',
        padding: '0.2rem 0.4rem 0.2rem 1rem',
    },
    descriptionToggle: {
        display: 'flex',
        flexDirection: 'row',
        cursor: 'pointer',
        paddingBottom: '0.5rem',
    },
    cardContent: {
        display: 'flex',
        flexDirection: 'column',
        paddingTop: 0,
        paddingLeft: '0.5rem',
    },
    nameSection: {
        display: 'flex',
        flexDirection: 'row',
        alignItems: 'center',
    },
    namePlate: {
        display: 'flex',
        flexDirection: 'row',
        alignItems: 'center',
        margin: 0,
        minHeight: '2.7rem',
    },
    faveIcon: {
        fontSize: '0.8rem',
        margin: 'auto 0 1rem 0.3rem',
        color: theme.palette.text.primary,
    },
    frozenIcon: {
        fontSize: '0.5rem',
        marginLeft: '0.3rem',
        height: '1rem',
        color: theme.palette.text.primary,
    },
    accountStatusSection: {
        display: 'flex',
        flexDirection: 'row',
        alignItems: 'center',
        paddingLeft: '1rem',
    },
    chipSection: {
        marginBottom: '-2rem',
    },
    tag: {
        marginRight: '0.75rem',
        marginBottom: '0.5rem',
    },
    description: {
        maxWidth: '95%',
        marginTop: 0,
    },
});

const mapStateToProps = ({ auth, selectedResourceUuid, resources, properties }: RootState) => {
    const currentResource = getResource(properties.currentRouteUuid)(resources);
    const frozenByUser = currentResource && getResource((currentResource as ProjectResource).frozenByUuid as string)(resources);
    const frozenByFullName = frozenByUser && (frozenByUser as Resource & { fullName: string }).fullName;
    const isSelected = selectedResourceUuid === properties.currentRouteUuid;

    return {
        isAdmin: auth.user?.isAdmin,
        currentResource,
        frozenByFullName,
        isSelected,
    };
};

const mapDispatchToProps = (dispatch: Dispatch) => ({
    handleCardClick: (uuid: string) => {
        dispatch<any>(loadDetailsPanel(uuid));
        dispatch<any>(setSelectedResourceUuid(uuid));
        dispatch<any>(deselectAllOthers(uuid));
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
                        handleCardClick={handleCardClick}
                    />
                );
            default:
                return null;
        }
    })
);

const UserCard: React.FC<UserCardProps> = ({ classes, currentResource, handleCardClick, isSelected }) => {
    const { fullName, uuid } = currentResource as UserResource & { fullName: string };

    return (
        <Card
            className={classes.root}
            onClick={() => handleCardClick(uuid)}
            data-cy='user-details-card'
        >
            <Grid
                container
                wrap='nowrap'
                className={classes.cardHeaderContainer}
            >
                <CardHeader
                    className={classes.cardHeader}
                    title={
                        <section className={classes.userNameContainer}>
                            <Typography
                                noWrap
                                variant='h6'
                            >
                                {fullName}
                            </Typography>
                            <section className={classes.accountStatusSection}>
                                {!currentResource.isActive && (
                                    <Typography>
                                        <UserResourceAccountStatus uuid={uuid} />
                                    </Typography>
                            )}
                            </section>
                        </section>
                    }
                />
                {isSelected && <MultiselectToolbar />}
            </Grid>
        </Card>
    );
};

const ProjectCard: React.FC<ProjectCardProps> = ({ classes, currentResource, frozenByFullName, handleCardClick, isSelected }) => {
    const { name, description, uuid } = currentResource as ProjectResource;
    const [showDescription, setShowDescription] = React.useState(false);
    const [showProperties, setShowProperties] = React.useState(false);

    const toggleDescription = () => {
        console.log(showDescription, showProperties);
        setShowDescription(!showDescription);
    };

    const toggleProperties = () => {
        setShowProperties(!showProperties);
    };

    return (
        <Card
            className={classes.root}
            onClick={() => handleCardClick(uuid)}
            data-cy='project-details-card'
        >
            <Grid
                container
                wrap='nowrap'
                className={classes.cardHeaderContainer}
            >
                <CardHeader
                    className={classes.cardHeader}
                    title={
                        <section className={classes.nameSection}>
                            <section className={classes.namePlate}>
                                <Typography
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
                        </section>
                    }
                />
                {isSelected && <MultiselectToolbar />}
            </Grid>
            <section onClick={(ev) => ev.stopPropagation()}>
                {description ? (
                    <section
                        onClick={toggleDescription}
                        className={classes.descriptionToggle}
                    >
                        <ExpandChevronRight expanded={showDescription} />
                        <section className={classes.showMore}>
                            <Collapse
                                in={showDescription}
                                timeout='auto'
                                collapsedHeight='1.25rem'
                            >
                                <Typography
                                    className={classes.description}
                                    data-cy='project-description'
                                >
                                    {description}
                                </Typography>
                            </Collapse>
                        </section>
                    </section>
                ) : (
                    <Typography
                        className={classes.noDescription}
                        data-cy='no-description'
                    >
                        no description available
                    </Typography>
                )}
                {typeof currentResource.properties === 'object' && Object.keys(currentResource.properties).length > 0 ? (
                    <section
                        onClick={toggleProperties}
                        className={classes.descriptionToggle}
                    >
                        <ExpandChevronRight expanded={showProperties} />
                        <section className={classes.showMore}>
                            <Collapse
                                in={showProperties}
                                timeout='auto'
                                collapsedHeight='35px'
                            >
                                <div
                                    className={classes.description}
                                    data-cy='project-description'
                                >
                                    <CardContent className={classes.cardContent}>
                                        <Typography component='div' className={classes.chipSection}>
                                            {Object.keys(currentResource.properties).map((k) =>
                                                Array.isArray(currentResource.properties[k])
                                                    ? currentResource.properties[k].map((v: string) => getPropertyChip(k, v, undefined, classes.tag))
                                                    : getPropertyChip(k, currentResource.properties[k], undefined, classes.tag)
                                            )}
                                        </Typography>
                                    </CardContent>
                                </div>
                            </Collapse>
                        </section>
                    </section>
                ) : null}
            </section>
        </Card>
    );
};
