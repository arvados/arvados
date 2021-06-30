// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import axios from 'axios';
import { configure, shallow } from "enzyme";
import Adapter from 'enzyme-adapter-react-16';
import { ListItem } from '@material-ui/core';
import JSZip from 'jszip';
import { DownloadAction } from './download-action';

configure({ adapter: new Adapter() });

jest.mock('axios');

jest.mock('file-saver', () => ({
    saveAs: jest.fn(),
}));

const mock = {
    file: jest.fn(),
    generateAsync: jest.fn().mockImplementation(() => Promise.resolve('test')),
};

jest.mock('jszip', () => jest.fn().mockImplementation(() => mock));

describe('<DownloadAction />', () => {
    let props;
    let zip;

    beforeEach(() => {
        props = {};
        zip = new JSZip();
        (axios as any).get.mockImplementationOnce(() => Promise.resolve({ data: '1234' }));
    });

    it('should return null if missing href or kind of file in props', () => {
        // when
        const wrapper = shallow(<DownloadAction {...props} />);

        // then
        expect(wrapper.html()).toBeNull();
    });

    it('should return a element', () => {
        // setup
        props.href = '#';

        // when
        const wrapper = shallow(<DownloadAction {...props} />);

        // then
        expect(wrapper.html()).not.toBeNull();
    });

    it('should handle download', () => {
        // setup
        props = {
            href: ['file1'],
            kind: 'files',
            download: [],
            currentCollectionUuid: '123412-123123'
        };
        const wrapper = shallow(<DownloadAction {...props} />);

        // when
        wrapper.find(ListItem).simulate('click');

        // then
        expect(axios.get).toHaveBeenCalledWith(props.href[0]);
    });
});