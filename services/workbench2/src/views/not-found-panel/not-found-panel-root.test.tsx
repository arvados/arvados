// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { mount, configure } from 'enzyme';
import Adapter from "enzyme-adapter-react-16";
import { ThemeProvider, Theme, StyledEngineProvider } from '@mui/material';
import { StyledComponentProps } from '@mui/styles';
import { ClusterConfigJSON } from 'common/config';
import { CustomTheme } from 'common/custom-theme';
import { NotFoundPanelRoot, NotFoundPanelRootDataProps, CssRules } from './not-found-panel-root';


declare module '@mui/styles/defaultTheme' {
  // eslint-disable-next-line @typescript-eslint/no-empty-interface
  interface DefaultTheme extends Theme {}
}


configure({ adapter: new Adapter() });

describe('NotFoundPanelRoot', () => {
    let props: NotFoundPanelRootDataProps & StyledComponentProps<CssRules>;

    beforeEach(() => {
        props = {
            classes: {
                root: 'root',
                title: 'title',
                active: 'active',
            },
            clusterConfig: {
                Mail: {
                    SupportEmailAddress: 'support@example.com'
                }
            } as ClusterConfigJSON,
            location: null,
        };
    });

    it('should render component', () => {
        // given
        const expectedMessage = "The page you requested was not found";

        // when
        const wrapper = mount(
            <StyledEngineProvider injectFirst>
                <ThemeProvider theme={CustomTheme}>
                    <NotFoundPanelRoot {...props} />
                </ThemeProvider>
            </StyledEngineProvider>
            );

        // then
        expect(wrapper.find('p').text()).toContain(expectedMessage);
    });

    it('should render component without email url when no email', () => {
        // setup
        props.clusterConfig.Mail.SupportEmailAddress = '';

        // when
        const wrapper = mount(
            <StyledEngineProvider injectFirst>
                <ThemeProvider theme={CustomTheme}>
                    <NotFoundPanelRoot {...props} />
                </ThemeProvider>
            </StyledEngineProvider>
            );

        // then
        expect(wrapper.find('a').length).toBe(0);
    });

    it('should render component with additional message and email url', () => {
        // given
        const hash = '123hash123';
        const pathname = `/collections/${hash}`;

        // setup
        props.location = {
            pathname,
        } as any;

        // when
        const wrapper = mount(
            <StyledEngineProvider injectFirst>
                <ThemeProvider theme={CustomTheme}>
                    <NotFoundPanelRoot {...props} />
                </ThemeProvider>
            </StyledEngineProvider>
            );

        // then
        expect(wrapper.find('p').first().text()).toContain(hash);

        // and
        expect(wrapper.find('a').length).toBe(1);
    });
});