// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { configure, mount } from "enzyme";
import * as Adapter from 'enzyme-adapter-react-16';
import { AutoLogoutComponent, AutoLogoutProps, LAST_ACTIVE_TIMESTAMP } from './auto-logout';

configure({ adapter: new Adapter() });

describe('<AutoLogoutComponent />', () => {
    let props: AutoLogoutProps;
    const sessionIdleTimeout = 300;
    const lastWarningDuration = 60;
    const eventListeners = {};
    jest.useFakeTimers();

    beforeAll(() => {
        window.addEventListener = jest.fn((event, cb) => {
            eventListeners[event] = cb;
        });
    });

    beforeEach(() => {
        props = {
            sessionIdleTimeout: sessionIdleTimeout,
            lastWarningDuration: lastWarningDuration,
            doLogout: jest.fn(),
            doWarn: jest.fn(),
            doCloseWarn: jest.fn(),
        };
        mount(<div><AutoLogoutComponent {...props} /></div>);
    });

    it('should logout after idle timeout', () => {
        jest.runTimersToTime((sessionIdleTimeout-1)*1000);
        expect(props.doLogout).not.toBeCalled();
        jest.runTimersToTime(1*1000);
        expect(props.doLogout).toBeCalled();
    });

    it('should warn the user previous to close the session', () => {
        jest.runTimersToTime((sessionIdleTimeout-lastWarningDuration-1)*1000);
        expect(props.doWarn).not.toBeCalled();
        jest.runTimersToTime(1*1000);
        expect(props.doWarn).toBeCalled();
    });

    it('should reset the idle timer when activity event is received', () => {
        jest.runTimersToTime((sessionIdleTimeout-lastWarningDuration-1)*1000);
        expect(props.doWarn).not.toBeCalled();
        // Simulate activity from other window/tab
        eventListeners.storage({
            key: LAST_ACTIVE_TIMESTAMP,
            newValue: '42' // value currently doesn't matter
        })
        jest.runTimersToTime(1*1000);
        // Warning should not appear because idle timer was reset
        expect(props.doWarn).not.toBeCalled();
    });
});