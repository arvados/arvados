// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Dispatch } from 'redux';
import { connect } from 'react-redux';
import { RootState } from 'store/store';
import { values, pipe } from 'lodash/fp';
import { HomeTreePicker } from 'views-components/projects-tree-picker/home-tree-picker';
import { SharedTreePicker } from 'views-components/projects-tree-picker/shared-tree-picker';
import { FavoritesTreePicker } from 'views-components/projects-tree-picker/favorites-tree-picker';
import { getProjectsTreePickerIds, treePickerSearchActions, SHARED_PROJECT_ID, FAVORITES_PROJECT_ID } from 'store/tree-picker/tree-picker-actions';
import { TreeItem } from 'components/tree/tree';
import { ProjectsTreePickerItem } from 'store/tree-picker/tree-picker-middleware';
import { PublicFavoritesTreePicker } from './public-favorites-tree-picker';
import { SearchInput } from 'components/search-input/search-input';

export interface ProjectsTreePickerProps {
    pickerId: string;
    includeCollections?: boolean;
    includeFiles?: boolean;
    showSelection?: boolean;
    options?: { showOnlyOwned: boolean, showOnlyWritable: boolean };
    toggleItemActive?: (event: React.MouseEvent<HTMLElement>, item: TreeItem<ProjectsTreePickerItem>, pickerId: string) => void;
    toggleItemSelection?: (event: React.MouseEvent<HTMLElement>, item: TreeItem<ProjectsTreePickerItem>, pickerId: string) => void;
}

interface ProjectsTreePickerSearchProps {
    projectSearch: string;
    collectionFilter: string;
}

interface ProjectsTreePickerActionProps {
    onProjectSearch: (value: string) => void;
    onCollectionFilter: (value: string) => void;
}

type ProjectsTreePickerCombinedProps = ProjectsTreePickerProps & ProjectsTreePickerSearchProps & ProjectsTreePickerActionProps;

const mapStateToProps = (state: RootState, props: ProjectsTreePickerProps): ProjectsTreePickerSearchProps => ({
    projectSearch: "",
    collectionFilter: "",
    ...props
});

const mapDispatchToProps = (dispatch: Dispatch, props: ProjectsTreePickerProps): ProjectsTreePickerActionProps => {
    const { home, shared, favorites, publicFavorites } = getProjectsTreePickerIds(props.pickerId);
    const params = {
        includeCollections: props.includeCollections,
        includeFiles: props.includeFiles,
        options: props.options
    };
    dispatch(treePickerSearchActions.SET_TREE_PICKER_LOAD_PARAMS({ pickerId: home, params }));
    dispatch(treePickerSearchActions.SET_TREE_PICKER_LOAD_PARAMS({ pickerId: shared, params }));
    dispatch(treePickerSearchActions.SET_TREE_PICKER_LOAD_PARAMS({ pickerId: favorites, params }));
    dispatch(treePickerSearchActions.SET_TREE_PICKER_LOAD_PARAMS({ pickerId: publicFavorites, params }));

    return {
        onProjectSearch: (projectSearchValue: string) => dispatch(treePickerSearchActions.SET_TREE_PICKER_PROJECT_SEARCH({ pickerId: props.pickerId, projectSearchValue })),
        onCollectionFilter: (collectionFilterValue: string) => {
            dispatch(treePickerSearchActions.SET_TREE_PICKER_COLLECTION_FILTER({ pickerId: home, collectionFilterValue }));
            dispatch(treePickerSearchActions.SET_TREE_PICKER_COLLECTION_FILTER({ pickerId: shared, collectionFilterValue }));
            dispatch(treePickerSearchActions.SET_TREE_PICKER_COLLECTION_FILTER({ pickerId: favorites, collectionFilterValue }));
            dispatch(treePickerSearchActions.SET_TREE_PICKER_COLLECTION_FILTER({ pickerId: publicFavorites, collectionFilterValue }));
        }
    }
};

export const ProjectsTreePicker = connect(mapStateToProps, mapDispatchToProps)(({ pickerId, onProjectSearch, onCollectionFilter, ...props }: ProjectsTreePickerCombinedProps) => {
    const { home, shared, favorites, publicFavorites } = getProjectsTreePickerIds(pickerId);
    const relatedTreePickers = getRelatedTreePickers(pickerId);
    const p = {
        ...props,
        relatedTreePickers,
        disableActivation,
    };
    return <div>
        <span>
            <SearchInput value="" label="Search Projects" selfClearProp='' onSearch={onProjectSearch} debounce={200} />
            <SearchInput value="" label="Filter Collections" selfClearProp='' onSearch={onCollectionFilter} debounce={200} />
        </span>
        <div data-cy="projects-tree-home-tree-picker">
            <HomeTreePicker pickerId={home} {...p} />
        </div>
        <div data-cy="projects-tree-shared-tree-picker">
            <SharedTreePicker pickerId={shared} {...p} />
        </div>
        <div data-cy="projects-tree-public-favourites-tree-picker">
            <PublicFavoritesTreePicker pickerId={publicFavorites} {...p} />
        </div>
        <div data-cy="projects-tree-favourites-tree-picker">
            <FavoritesTreePicker pickerId={favorites} {...p} />
        </div>
    </div>;
});

const getRelatedTreePickers = pipe(getProjectsTreePickerIds, values);
const disableActivation = [SHARED_PROJECT_ID, FAVORITES_PROJECT_ID];
