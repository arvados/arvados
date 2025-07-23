// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useEffect, useState } from 'react';
import { isEqual } from 'lodash';
import { Collapse } from '@mui/material';
import withStyles from '@mui/styles/withStyles';
import { WithStyles } from '@mui/styles';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { connect } from 'react-redux';
import { Dispatch } from 'redux';
import { RootState } from 'store/store';
import { ArvadosTheme } from 'common/custom-theme';
import { loadFavoritePanel } from 'store/favorite-panel/favorite-panel-action';
import { ExpandChevronRight } from 'components/expand-chevron-right/expand-chevron-right';
import { GroupContentsResource } from 'services/groups-service/groups-service';
import { FavePinItem } from './favorite-pins-item';
import { getResource } from 'store/resources/resources';
import { LinkResource } from 'models/link';
import { ResourcesState } from 'store/resources/resources';

type CssRules = 'root' | 'title' | 'hr' | 'list';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
    },
    title: {
        margin: '0 1rem',
        padding: '4px',
    },
    hr: {
        marginTop: '0',
        marginBottom: '0',
    },
    list: {
        marginTop: '0.5rem',
        display: 'flex',
        flexWrap: 'wrap',
        justifyContent: 'flex-start',
        width: '100%',
    },
});

const mapStateToProps = (state: RootState): Pick<FavePinsSectionProps, 'faves' | 'resources'> => {
    return {
        faves: state.favoritesLinks,
        resources: state.resources,
    };
};

const mapDispatchToProps = (dispatch: Dispatch): Pick<FavePinsSectionProps, 'loadFavoritePanel' > => ({
    loadFavoritePanel: () => dispatch<any>(loadFavoritePanel()),
});

type FavePinsSectionProps = {
    faves: LinkResource[];
    resources: ResourcesState;
    loadFavoritePanel: () => void;
};

export const FavePinsSection = connect(
    mapStateToProps,
    mapDispatchToProps
)(
    withStyles(styles)(
        React.memo(({ faves, resources, loadFavoritePanel, classes }: FavePinsSectionProps & WithStyles<CssRules>) => {
            const [items, setItems] = useState<GroupContentsResource[]>([]);
            const [isOpen, setIsOpen] = useState(true);

            useEffect(() => {
                const sortedFaves = faves.sort((a, b) => b.createdAt.localeCompare(a.createdAt)).slice(0, 12); //max 12 items
                setItems(getResources(sortedFaves, resources));
            }, [faves, resources]);

            useEffect(() => {
                loadFavoritePanel();
                // eslint-disable-next-line react-hooks/exhaustive-deps
            }, []);

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
                            {items.map((item) => (
                                <FavePinItem
                                    key={item.uuid}
                                    item={item}
                                />
                            ))}
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

const getResources = (faves: LinkResource[], resources: ResourcesState) => {
    return faves.reduce((acc: GroupContentsResource[], fave: LinkResource) => {
        const res = getResource<GroupContentsResource>(fave.headUuid)(resources);
        if (res) acc.push(res);
        return acc;
    }, []);
};
