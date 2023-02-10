// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { configure, shallow, mount } from "enzyme";
import { BannerComponent } from './banner';
import { Button } from "@material-ui/core";
import Adapter from "enzyme-adapter-react-16";
import servicesProvider from '../../common/service-provider';

configure({ adapter: new Adapter() });

jest.mock('../../common/service-provider', () => ({
    getServices: jest.fn(),
}));

describe('<BannerComponent />', () => {

    let props;

    beforeEach(() => {
        props = {
            isOpen: false,
            bannerUUID: undefined,
            keepWebInlineServiceUrl: '',
            openBanner: jest.fn(),
            closeBanner: jest.fn(),
            classes: {} as any,
        }
    });

    it('renders without crashing', () => {
        // when
        const banner = shallow(<BannerComponent {...props} />);
        
        // then
        expect(banner.find(Button)).toHaveLength(1);
    });

    it('calls collectionService', () => {
        // given
        props.isOpen = true;
        props.bannerUUID = '123';
        const mocks = {
            collectionService: {
                files: jest.fn(() => ({ then: (callback) => callback([{ name: 'banner.html' }]) })),
                getFileContents: jest.fn(() => ({ then: (callback) => callback('<h1>Test</h1>') }))
            }
        };
        (servicesProvider.getServices as any).mockImplementation(() => mocks);

        // when
        const banner = mount(<BannerComponent {...props} />);

        // then
        expect(servicesProvider.getServices).toHaveBeenCalled();
        expect(mocks.collectionService.files).toHaveBeenCalled();
        expect(mocks.collectionService.getFileContents).toHaveBeenCalled();
        expect(banner.html()).toContain('<h1>Test</h1>');
    });
});

