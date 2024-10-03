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
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { ArvadosTheme } from 'common/custom-theme';
import { ResourceKind } from 'models/resource';
import { CollectionFileType } from 'models/collection-file';
import { DefaultView } from 'components/default-view/default-view';
import { ProjectDetailsComponent } from 'views-components/details-panel/project-details';
import { CollectionDetailsAttributes } from 'views/collection-panel/collection-panel';
import { RootProjectDetailsComponent } from 'views-components/details-panel/root-project-details';
import { DetailsAttribute } from 'components/details-attribute/details-attribute';
import { formatFileSize } from 'common/formatters';

export interface ToplevelPickerProps {
    currentUuids?: string[];
    pickerId: string;
    cascadeSelection: boolean;
    includeCollections?: boolean;
    includeDirectories?: boolean;
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
        projectSearch: state.treePickerSearch.projectSearchValues[search] || state.treePickerSearch.collectionFilterValues[search],
        collectionFilter: state.treePickerSearch.collectionFilterValues[search],
    };
};

const mapDispatchToProps = (dispatch: Dispatch, props: ToplevelPickerProps): (ProjectsTreePickerActionProps & DispatchProp) => {
    const { home, shared, favorites, publicFavorites, search } = getProjectsTreePickerIds(props.pickerId);
    const params = {
        includeCollections: props.includeCollections,
        includeDirectories: props.includeDirectories,
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

type CssRules = 'pickerHeight' | 'searchFlex' | 'scrolledBox' | 'detailsBox' | 'twoCol';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    pickerHeight: {
        height: "100%",
    },
    searchFlex: {
        display: "flex",
        justifyContent: "space-around",
        height: "64px",
    },
    scrolledBox: {
        overflow: "scroll",
        width: "calc(100% - 320px)",
        marginRight: "8px",
        height: "100%",
    },
    twoCol: {
        display: "flex",
        flexDirection: "row",
        height: "calc(100% - 64px)",
    },
    detailsBox: {
        width: "320px",
        height: "100%",
        overflow: "scroll",
        borderLeft: "1px solid rgba(0, 0, 0, 0.12)",
        paddingLeft: "4px",
    }
});

type ProjectsTreePickerCombinedProps = ToplevelPickerProps & ProjectsTreePickerSearchProps & ProjectsTreePickerActionProps & DispatchProp & WithStyles<CssRules>;

interface SelectionComponentState {
    activeItem?: ProjectsTreePickerItem;
}

const Details = (props: { res?: ProjectsTreePickerItem }) => {
    if (props.res) {
        if ('kind' in props.res) {
            switch (props.res.kind) {
                case ResourceKind.PROJECT:
                    return <ProjectDetailsComponent project={props.res} hideEdit={true} />
                case ResourceKind.COLLECTION:
                    return <CollectionDetailsAttributes item={props.res} />;
                case ResourceKind.USER:
                    return <RootProjectDetailsComponent rootProject={props.res} />;
                    // case ResourceKind.PROCESS:
                    //                         return new ProcessDetails(res);
                    // case ResourceKind.WORKFLOW:
                    //     return new WorkflowDetails(res);
            }
        } else if ('type' in props.res) {
            if (props.res.type === CollectionFileType.FILE) {
                return <>
                    <DetailsAttribute label='Type' value="File" />
                    <DetailsAttribute label='Size' value={formatFileSize(props.res.size)} />
                </>;
                } else {
                    return <DetailsAttribute label='Type' value="Directory" />
                }
                }
                }
                return <DefaultView messages={['Select a file or folder to view its details.']} />;
                };


                export const ProjectsTreePicker = connect(mapStateToProps, mapDispatchToProps)(
                withStyles(styles)(
                class FileInputComponent extends React.Component<ProjectsTreePickerCombinedProps> {
                    state: SelectionComponentState = {
                    };

                    componentDidMount() {
                    const { home, shared, favorites, publicFavorites, search } = getProjectsTreePickerIds(this.props.pickerId);

                    const preloadParams = this.props.currentUuids ? {
                    selectedItemUuids: this.props.currentUuids,
                    includeDirectories: !!this.props.includeDirectories,
                    includeFiles: !!this.props.includeFiles,
                    multi: !!this.props.showSelection,
                    } : undefined;
                    this.props.dispatch<any>(initProjectsTreePicker(this.props.pickerId, preloadParams));

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

                    setSelection(event: React.MouseEvent<HTMLElement>,
                    item: TreeItem<ProjectsTreePickerItem>,
                    pickerId: string) {
                    this.setState({activeItem: item.data});
                    console.log(item.data);
                    }

                    render() {
                    const pickerId = this.props.pickerId;
                    const onProjectSearch = this.props.onProjectSearch;
                    const onCollectionFilter = this.props.onCollectionFilter;

                    const { home, shared, favorites, publicFavorites, search } = getProjectsTreePickerIds(pickerId);
                    const relatedTreePickers = getRelatedTreePickers(pickerId);
                    const _this = this;
                    const p = {
                    cascadeSelection: this.props.cascadeSelection,
                    includeCollections: this.props.includeCollections,
                    includeDirectories: this.props.includeDirectories,
                    includeFiles: this.props.includeFiles,
                    showSelection: this.props.showSelection,
                    options: this.props.options,
                    toggleItemActive: (event: React.MouseEvent<HTMLElement>,
                    item: TreeItem<ProjectsTreePickerItem>,
                    pickerId: string): void => {
                    _this.setSelection(event, item, pickerId);
                    if (_this.props.toggleItemActive) {
                    _this.props.toggleItemActive(event, item, pickerId);
                    }
                    },
                    toggleItemSelection: this.props.toggleItemSelection,
                    relatedTreePickers,
                    disableActivation,
                    };


                    return <>
                    <div className={this.props.classes.searchFlex}>
                        <span data-cy="picker-dialog-project-search"><SearchInput value="" label="Project search" selfClearProp='' onSearch={onProjectSearch} debounce={500} width="18rem"  /></span>
                        {this.props.includeCollections &&
                         <span data-cy="picker-dialog-collection-search" ><SearchInput value="" label="Collection search" selfClearProp='' onSearch={onCollectionFilter} debounce={500} width="18rem" /></span>}
                        </div>

                        <div className={this.props.classes.twoCol}>
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

                            <div className={this.props.classes.detailsBox} data-cy="picker-dialog-details">
                                <Details res={this.state.activeItem} />
                            </div>
                        </div>
                        </>;
            }
        }));

const getRelatedTreePickers = pipe(getProjectsTreePickerIds, values);
const disableActivation = [SHARED_PROJECT_ID, FAVORITES_PROJECT_ID];
