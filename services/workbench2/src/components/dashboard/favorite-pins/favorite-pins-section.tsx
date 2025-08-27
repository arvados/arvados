// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useEffect, useState } from 'react';
import { isEqual } from 'lodash';
import { Collapse, Grid } from '@mui/material';
import withStyles from '@mui/styles/withStyles';
import { WithStyles } from '@mui/styles';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { connect } from 'react-redux';
import { RootState } from 'store/store';
import { ArvadosTheme } from 'common/custom-theme';
import { ExpandChevronRight } from 'components/expand-chevron-right/expand-chevron-right';
import { GroupContentsResource } from 'services/groups-service/groups-service';
import { FavePinItem } from './favorite-pins-item';
import { LinkResource } from 'models/link';
import { ResourcesState, getPopulatedResources, getResource } from 'store/resources/resources';

type CssRules = 'root' | 'title' | 'hr' | 'list';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
    },
    title: {
        margin: '0 1rem',
        padding: '4px',
        cursor: 'pointer',
    },
    hr: {
        marginTop: '0',
        marginBottom: '0',
    },
    list: {
        marginTop: '0.5rem',
        paddingLeft: '1rem',
        width: '98.5%',
    },
});

const mapStateToProps = (state: RootState): Pick<FavePinsSectionProps, 'faves' | 'resources'> => {
    return {
        faves: state.dataExplorer.favoritePins.items,
        resources: state.resources,
    };
};

type FavePinsSectionProps = {
    faves: string[];
    resources: ResourcesState;
};

export const FavePinsSection = connect(
    mapStateToProps
)(
    withStyles(styles)(
        React.memo(({ faves, resources, classes }: FavePinsSectionProps & WithStyles<CssRules>) => {
            const [items, setItems] = useState<GroupContentsResource[]>([]);
            const [isOpen, setIsOpen] = useState(true);

            useEffect(() => {
                const faveLinks = faves.reduce((acc: LinkResource[], fave: string): LinkResource[] => {
                        const faveLink = getResource<LinkResource>(fave)(resources)
                        if (faveLink) acc.push(faveLink);
                        return acc;
                    }, []);
                const sortedFaves = faveLinks.sort((a, b) => b.createdAt.localeCompare(a.createdAt))
                setItems(getPopulatedResources(sortedFaves.map(item => item.headUuid), resources));
            }, [faves, resources]);

            return (
                <div className={classes.root}>
                    <div
                        className={classes.title}
                        onClick={() => setIsOpen(!isOpen)}
                    >
                        <span>Favorites</span>
                        <ExpandChevronRight expanded={isOpen} />
                        <hr className={classes.hr} />
                    </div>
                    <Collapse in={isOpen}>
                        <div className={classes.list}>
                            <Grid
                                container
                                spacing={2}
                                direction='row'
                                justifyContent='flex-start'
                                alignItems='flex-start'
                                >
                                {items.map((item) => (
                                    <Grid item xs={12} sm={6} md={5} lg={4} xl={3} key={item.uuid}>
                                        <FavePinItem
                                            item={item}
                                        />
                                    </Grid>
                                ))}
                            </Grid>
                        </div>
                    </Collapse>
                </div>
            );
        }, preventRerender)
    )
);

// return true to prevent re-render, false to allow re-render
function preventRerender(prevProps: FavePinsSectionProps, nextProps: FavePinsSectionProps) {
    if (!isEqual(prevProps.faves, nextProps.faves)) {
        return false;
    }
    if (!isEqual(prevProps.resources, nextProps.resources)) {
        return false;
    }
    return true;
}

