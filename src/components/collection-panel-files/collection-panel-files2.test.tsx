// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { configure, shallow, mount } from "enzyme";
import { WithStyles } from "@material-ui/core";
import Adapter from "enzyme-adapter-react-16";
import { TreeItem, TreeItemStatus } from '../tree/tree';
import { FileTreeData } from '../file-tree/file-tree-data';
import { CollectionFileType } from "../../models/collection-file";
import { CollectionPanelFilesComponent, CollectionPanelFilesProps, CssRules } from './collection-panel-files2';
import { SearchInput } from '../search-input/search-input';

configure({ adapter: new Adapter() });

jest.mock('components/file-tree/file-tree', () => ({
    FileTree: () => 'FileTree',
}));

describe('<CollectionPanelFiles />', () => {
    let props: CollectionPanelFilesProps & WithStyles<CssRules>;

    beforeEach(() => {
        props = {
            classes: {} as Record<CssRules, string>,
            items: [],
            isWritable: true,
            isLoading: false,
            tooManyFiles: false,
            onUploadDataClick: jest.fn(),
            onSearchChange: jest.fn(),
            onItemMenuOpen: jest.fn(),
            onOptionsMenuOpen: jest.fn(),
            onSelectionToggle: jest.fn(),
            onCollapseToggle: jest.fn(),
            onFileClick: jest.fn(),
            loadFilesFunc: jest.fn(),
            currentItemUuid: '',
        };
    });

    it('renders properly', () => {
        // when
        const wrapper = shallow(<CollectionPanelFilesComponent {...props} />);

        // then
        expect(wrapper).not.toBeUndefined();
    });

    it('filters out files', () => {
        // given
        const searchPhrase = 'test';
        const items: Array<TreeItem<FileTreeData>> = [
            {
                data: {
                    url: '',
                    type: CollectionFileType.DIRECTORY,
                    name: 'test',
                },
                id: '1',
                open: true,
                active: true,
                status: TreeItemStatus.LOADED,
            },
            {
                data: {
                    url: '',
                    type: CollectionFileType.FILE,
                    name: 'test123',
                },
                id: '2',
                open: true,
                active: true,
                status: TreeItemStatus.LOADED,
            },
            {
                data: {
                    url: '',
                    type: CollectionFileType.FILE,
                    name: 'another-file',
                },
                id: '3',
                open: true,
                active: true,
                status: TreeItemStatus.LOADED,
            }
        ];

        // setup
        props.items = items;
        const wrapper = mount(<CollectionPanelFilesComponent {...props} />);
        wrapper.find(SearchInput).simulate('change', { target: { value: searchPhrase } });

        // when
        setTimeout(() => { // we have to use set timeout because of the debounce
            expect(wrapper.find('FileTree').prop('items'))
            .toEqual([
                {
                    data: { url: '', type: 'directory', name: 'test' },
                    id: '1',
                    open: true,
                    active: true,
                    status: 'loaded'
                },
                {
                    data: { url: '', type: 'file', name: 'test123' },
                    id: '2',
                    open: true,
                    active: true,
                    status: 'loaded'
                }
            ]);
        }, 0);
    });
});