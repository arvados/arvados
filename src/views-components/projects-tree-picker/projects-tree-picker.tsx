// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Dispatch } from 'redux';
import { connect, DispatchProp } from 'react-redux';
import { RootState } from 'store/store';
import { values, pipe } from 'lodash/fp';
import { HomeTreePicker } from 'views-components/projects-tree-picker/home-tree-picker';
import { SharedTreePicker } from 'views-components/projects-tree-picker/shared-tree-picker';
import { FavoritesTreePicker } from 'views-components/projects-tree-picker/favorites-tree-picker';
import { SearchProjectsPicker } from 'views-components/projects-tree-picker/search-projects-picker';
import {
    getProjectsTreePickerIds, treePickerActions, treePickerSearchActions, initProjectsTreePicker,
    SHARED_PROJECT_ID, FAVORITES_PROJECT_ID
} from 'store/tree-picker/tree-picker-actions';
import { TreeItem } from 'components/tree/tree';
import { ProjectsTreePickerItem } from 'store/tree-picker/tree-picker-middleware';
import { PublicFavoritesTreePicker } from './public-favorites-tree-picker';
import { SearchInput } from 'components/search-input/search-input';
import { withStyles, StyleRulesCallback, WithStyles } from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';

export interface ToplevelPickerProps {
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

const mapStateToProps = (state: RootState, props: ToplevelPickerProps): ProjectsTreePickerSearchProps => {
    const { search } = getProjectsTreePickerIds(props.pickerId);
    return {
        ...props,
        projectSearch: state.treePickerSearch.projectSearchValues[search],
        collectionFilter: state.treePickerSearch.collectionFilterValues[search],
    };
};

const mapDispatchToProps = (dispatch: Dispatch, props: ToplevelPickerProps): (ProjectsTreePickerActionProps & DispatchProp) => {
    const { home, shared, favorites, publicFavorites, search } = getProjectsTreePickerIds(props.pickerId);
    const params = {
        includeCollections: props.includeCollections,
        includeFiles: props.includeFiles,
        options: props.options
    };
    dispatch(treePickerSearchActions.SET_TREE_PICKER_LOAD_PARAMS({ pickerId: home, params }));
    dispatch(treePickerSearchActions.SET_TREE_PICKER_LOAD_PARAMS({ pickerId: shared, params }));
    dispatch(treePickerSearchActions.SET_TREE_PICKER_LOAD_PARAMS({ pickerId: favorites, params }));
    dispatch(treePickerSearchActions.SET_TREE_PICKER_LOAD_PARAMS({ pickerId: publicFavorites, params }));
    dispatch(treePickerSearchActions.SET_TREE_PICKER_LOAD_PARAMS({ pickerId: search, params }));

    return {
        onProjectSearch: (projectSearchValue: string) => dispatch(treePickerSearchActions.SET_TREE_PICKER_PROJECT_SEARCH({ pickerId: search, projectSearchValue })),
        onCollectionFilter: (collectionFilterValue: string) => {
            dispatch(treePickerSearchActions.SET_TREE_PICKER_COLLECTION_FILTER({ pickerId: home, collectionFilterValue }));
            dispatch(treePickerSearchActions.SET_TREE_PICKER_COLLECTION_FILTER({ pickerId: shared, collectionFilterValue }));
            dispatch(treePickerSearchActions.SET_TREE_PICKER_COLLECTION_FILTER({ pickerId: favorites, collectionFilterValue }));
            dispatch(treePickerSearchActions.SET_TREE_PICKER_COLLECTION_FILTER({ pickerId: publicFavorites, collectionFilterValue }));
            dispatch(treePickerSearchActions.SET_TREE_PICKER_COLLECTION_FILTER({ pickerId: search, collectionFilterValue }));
        },
        dispatch
    }
};

type CssRules = 'pickerHeight' | 'searchFlex' | 'scrolledBox';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    pickerHeight: {
        height: "calc(70vh - 100px)"
    },
    searchFlex: {
        display: "flex",
        justifyContent: "space-around",
        paddingBottom: "1em"
    },
    scrolledBox: {
        height: "calc(100% - 50px)",
        overflow: "scroll"
    }
});

type ProjectsTreePickerCombinedProps = ToplevelPickerProps & ProjectsTreePickerSearchProps & ProjectsTreePickerActionProps & DispatchProp & WithStyles<CssRules>;

export const ProjectsTreePicker = connect(mapStateToProps, mapDispatchToProps)(
    withStyles(styles)(
        class FileInputComponent extends React.Component<ProjectsTreePickerCombinedProps> {

            componentDidMount() {
                const { home, shared, favorites, publicFavorites, search } = getProjectsTreePickerIds(this.props.pickerId);

                this.props.dispatch<any>(initProjectsTreePicker(this.props.pickerId));

                this.props.dispatch(treePickerSearchActions.SET_TREE_PICKER_PROJECT_SEARCH({ pickerId: search, projectSearchValue: "" }));
                this.props.dispatch(treePickerSearchActions.SET_TREE_PICKER_COLLECTION_FILTER({ pickerId: search, collectionFilterValue: "" }));
                this.props.dispatch(treePickerSearchActions.SET_TREE_PICKER_COLLECTION_FILTER({ pickerId: home, collectionFilterValue: "" }));
                this.props.dispatch(treePickerSearchActions.SET_TREE_PICKER_COLLECTION_FILTER({ pickerId: shared, collectionFilterValue: "" }));
                this.props.dispatch(treePickerSearchActions.SET_TREE_PICKER_COLLECTION_FILTER({ pickerId: favorites, collectionFilterValue: "" }));
                this.props.dispatch(treePickerSearchActions.SET_TREE_PICKER_COLLECTION_FILTER({ pickerId: publicFavorites, collectionFilterValue: "" }));
            }

            componentWillUnmount() {
                const { home, shared, favorites, publicFavorites, search } = getProjectsTreePickerIds(this.props.pickerId);
                // Release all the state, we don't need it to hang around forever.
                this.props.dispatch(treePickerActions.RESET_TREE_PICKER({ pickerId: search }));
                this.props.dispatch(treePickerActions.RESET_TREE_PICKER({ pickerId: home }));
                this.props.dispatch(treePickerActions.RESET_TREE_PICKER({ pickerId: shared }));
                this.props.dispatch(treePickerActions.RESET_TREE_PICKER({ pickerId: favorites }));
                this.props.dispatch(treePickerActions.RESET_TREE_PICKER({ pickerId: publicFavorites }));
            }

            render() {
                const pickerId = this.props.pickerId;
                const onProjectSearch = this.props.onProjectSearch;
                const onCollectionFilter = this.props.onCollectionFilter;

                const { home, shared, favorites, publicFavorites, search } = getProjectsTreePickerIds(pickerId);
                const relatedTreePickers = getRelatedTreePickers(pickerId);
                const p = {
                    includeCollections: this.props.includeCollections,
                    includeFiles: this.props.includeFiles,
                    showSelection: this.props.showSelection,
                    options: this.props.options,
                    toggleItemActive: this.props.toggleItemActive,
                    toggleItemSelection: this.props.toggleItemSelection,
                    relatedTreePickers,
                    disableActivation,
                };
                return <div className={this.props.classes.pickerHeight} >
                    <span className={this.props.classes.searchFlex}>
                        <SearchInput value="" label="Search for a Project" selfClearProp='' onSearch={onProjectSearch} debounce={500} />
                        {this.props.includeCollections &&
                            <SearchInput value="" label="Filter Collections list in Projects" selfClearProp='' onSearch={onCollectionFilter} debounce={500} />}
                    </span>

                    <div className={this.props.classes.scrolledBox}>
                        {this.props.projectSearch ?
                            <div data-cy="projects-tree-search-picker">
                                <SearchProjectsPicker {...p} pickerId={search} />
                            </div>
                            :
                            <>
                                <div data-cy="projects-tree-home-tree-picker">
                                    <HomeTreePicker {...p} pickerId={home} />
                                </div>
                                <div data-cy="projects-tree-shared-tree-picker">
                                    <SharedTreePicker {...p} pickerId={shared} />
                                </div>
                                <div data-cy="projects-tree-public-favourites-tree-picker">
                                    <PublicFavoritesTreePicker {...p} pickerId={publicFavorites} />
                                </div>
                                <div data-cy="projects-tree-favourites-tree-picker">
                                    <FavoritesTreePicker {...p} pickerId={favorites} />
                                </div>
                            </>}
                    </div>
                </div >;
            }
        }));

const getRelatedTreePickers = pipe(getProjectsTreePickerIds, values);
const disableActivation = [SHARED_PROJECT_ID, FAVORITES_PROJECT_ID];
