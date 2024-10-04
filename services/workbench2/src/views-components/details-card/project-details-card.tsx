// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useState, useRef, useEffect } from 'react';
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
    | 'showMore'
    | 'noDescription'
    | 'userNameContainer'
    | 'cardContent'
    | 'nameSection'
    | 'namePlate'
    | 'faveIcon'
    | 'frozenIcon'
    | 'chipToggle'
    | 'chipSection'
    | 'tag'
    | 'description'
    | 'oneLineDescription'
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
        padding: '0.2rem 0.4rem 0.2rem 1rem',
    },
    descriptionToggle: {
        display: 'flex',
        flexDirection: 'row',
        cursor: 'pointer',
        marginTop: '-0.25rem',
        paddingBottom: '0.5rem',
    },
    cardContent: {
        display: 'flex',
        flexDirection: 'column',
        paddingTop: 0,
        paddingLeft: '0.1rem',
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
    showMore: {
        marginTop: 0,
        cursor: 'pointer',
    },
    chipToggle: {
        display: 'flex',
        alignItems: 'center',
        height: '2rem',
    },
    chipSection: {
        marginBottom: '-1rem',
    },
    tag: {
        marginRight: '0.75rem',
        marginBottom: '0.5rem',
    },
    description: {
        marginTop: 0,
        marginRight: '2rem',
    },
    oneLineDescription: {
        marginTop: 0,
        marginRight: '2rem',
        marginBottom: '-0.85rem',
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
        const [showDescription, setShowDescription] = useState(false);
        const [showProperties, setShowProperties] = useState(false);
        const [isMultiLine, setIsMultiLine] = useState(false);
        const descriptionRef = useRef<HTMLDivElement>(null);

        useEffect(() => {
            const checkIfMultiLine = () => {
              const element = descriptionRef.current;
              if (element) {
                // Compare the scroll width and offset width to determine if wrapping occurs
                setIsMultiLine(element.scrollWidth > element.offsetWidth);
              }
            };
        
            checkIfMultiLine();
          }, [description]);

        const toggleDescription = () => {
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
                                {!description && (
                                    <Typography
                                        data-cy='no-description'
                                        className={classes.noDescription}
                                    >
                                        no description available
                                    </Typography>
                                )}
                            </section>
                        }
                    />
                    {isSelected && <MultiselectToolbar injectedStyles={classes.toolbarStyles} />}
                </Grid>
                <section onClick={(ev) => ev.stopPropagation()}>
                    {description ? (
                        <section
                            onClick={toggleDescription}
                            className={classes.descriptionToggle}
                            data-cy='toggle-description'
                        >
                            <ExpandChevronRight expanded={showDescription} />
                            <section className={classes.showMore}>
                                <Collapse
                                    in={showDescription}
                                    timeout='auto'
                                    collapsedSize='1.25rem'
                                >
                                    {/* Hidden paragraph for measuring the text to determine if it is longer than one line */}
                                        <div ref={descriptionRef} style={{ position: 'absolute', visibility: 'hidden', whiteSpace: 'nowrap', width: '100%' }}>{description}</div>
                                    
                                    <Typography
                                        className={isMultiLine ? classes.description : classes.oneLineDescription}
                                        data-cy='project-description'
                                        //dangerouslySetInnerHTML is ok here only if description is sanitized,
                                        //which it is before it is loaded into the redux store
                                        dangerouslySetInnerHTML={{ __html: description }}
                                    />
                                </Collapse>
                            </section>
                        </section>
                    ) : (
                        <></>
                    )}
                    {typeof currentResource.properties === 'object' && Object.keys(currentResource.properties).length > 0 ? (
                        <section
                            onClick={toggleProperties}
                            className={classes.descriptionToggle}
                        >
                            <div
                                className={classes.chipToggle}
                                data-cy='toggle-chips'
                            >
                                <ExpandChevronRight expanded={showProperties} />
                            </div>
                            <section className={classes.showMore}>
                                <Collapse
                                    in={showProperties}
                                    timeout='auto'
                                    collapsedSize='35px'
                                >
                                    <div
                                        className={classes.description}
                                        data-cy='project-description'
                                    >
                                        <CardContent className={classes.cardContent}>
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
                                        </CardContent>
                                    </div>
                                </Collapse>
                            </section>
                        </section>
                    ) : null}
                </section>
            </Card>
        );
    })
);
