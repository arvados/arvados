// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { Card, CardHeader, Typography, CardContent, Tooltip, Collapse, Grid } from '@mui/material';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { ArvadosTheme } from 'common/custom-theme';
import { RootState } from 'store/store';
import { connect } from 'react-redux';
import { getResource } from 'store/resources/resources';
import { getPropertyChip } from '../resource-properties-form/property-chip';
import { ProjectResource } from 'models/project';
import { FavoriteStar, PublicFavoriteStar } from 'views-components/favorite-star/favorite-star';
import { FreezeIcon } from 'components/icon/icon';
import { Resource } from 'models/resource';
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
    | 'noDescription'
    | 'userNameContainer'
    | 'cardContent'
    | 'nameSection'
    | 'namePlate'
    | 'faveIcon'
    | 'frozenIcon'
    | 'chipSection'
    | 'tag'
    | 'description'
    | 'toolbarStyles';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
        marginBottom: '1rem',
        flex: '0 0 auto',
        padding: 0,
        minHeight: '3rem',
    },
    noDescription: {
        color: theme.palette.grey['600'],
        fontStyle: 'italic',
        marginLeft: '2rem',
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
        alignItems: 'center',
        justifyContent: 'space-between',
    },
    cardHeader: {
        minWidth: '30rem',
        padding: '0.2rem',
    },
    descriptionToggle: {
        marginLeft: '-16px',
    },
    cardContent: {
        display: 'flex',
        flexDirection: 'column',
        marginTop: '.5rem',
        paddingTop: '0px',
        paddingBottom: '0px',
        paddingLeft: '.5rem',
        paddingRight: '.5rem',
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
        marginLeft: '.5rem',
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
    chipSection: {
        marginBottom: '.5rem',
    },
    tag: {
        marginRight: '0.75rem',
        marginBottom: '0.5rem',
    },
    description: {
        marginTop: 0,
        marginRight: '2rem',
        marginLeft: '8px',
        maxWidth: "50em",
    },
    toolbarStyles: {
        marginRight: '-0.5rem',
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
});

type ProjectCardProps = WithStyles<CssRules> & {
    currentResource: ProjectResource;
    frozenByFullName: string | undefined;
    isAdmin: boolean;
    isSelected: boolean;
    handleCardClick: (resource: any) => void;
};

export const ProjectCard = connect(
    mapStateToProps,
    mapDispatchToProps
)(
    withStyles(styles)((props: ProjectCardProps) => {
        const { classes, currentResource, frozenByFullName, handleCardClick, isSelected } = props;
        const { name, description, uuid } = currentResource as ProjectResource;
        const [showDescription, setShowDescription] = React.useState(false);

        const toggleDescription = () => {
            setShowDescription(!showDescription);
        };

        const hasDescription = !!(description && description.length > 0);
        const hasProperties = (typeof currentResource.properties === 'object' && Object.keys(currentResource.properties).length > 0);
        const expandable = hasDescription || hasProperties;

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
                                                 {expandable && <span className={classes.descriptionToggle}
                                                                      onClick={toggleDescription}
                                                                      data-cy="toggle-description">
                                                     <ExpandChevronRight expanded={showDescription} />
                                                 </span>}
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
                                                                            {!hasDescription && (
                                                                                <Typography
                                                                                    data-cy='no-description'
                                                                                    className={classes.noDescription}
                                                                                >
                                                                                             no description available
                                                                                </Typography>
                                                                            )}

                                </section>
                            </section>
                        }
                    />
                    {isSelected && <MultiselectToolbar injectedStyles={classes.toolbarStyles} />}
                </Grid>

                {expandable && <Collapse
                                   in={showDescription}
                                   timeout='auto'
                                   collapsedSize='0rem'
                               >
                    <CardContent className={classes.cardContent}>
                        {hasProperties &&
                         <section data-cy='project-properties'>
                             <Typography
                                 component='div'
                                 className={classes.chipSection}
                             >
                                 {Object.keys(currentResource.properties).map((k) =>
                                     Array.isArray(currentResource.properties[k])
                                     ? currentResource.properties[k].map((v: string) => getPropertyChip(k, v, undefined, classes.tag))
                                     : getPropertyChip(k, currentResource.properties[k], undefined, classes.tag)
                                 )}
                             </Typography>
                         </section>}

                        {hasDescription && (
                            <section data-cy='project-description'>
                                <Typography
                                    className={classes.description}
                                    component='div'
                                    //dangerouslySetInnerHTML is ok here only if description is sanitized,
                                    //which it is before it is loaded into the redux store
                                    dangerouslySetInnerHTML={{ __html: description }}
                                />
                            </section>
                        )}
                    </CardContent>
                </Collapse>}
            </Card>
        );
    })
);
