// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { useAsyncInterval } from './use-async-interval';
import { configure, mount } from 'enzyme';
import Adapter from 'enzyme-adapter-react-16';
import FakeTimers from "@sinonjs/fake-timers";

configure({ adapter: new Adapter() });
const clock = FakeTimers.install();

jest.mock('react', () => {
    const originalReact = jest.requireActual('react');
    const mUseRef = jest.fn();
    return {
        ...originalReact,
        useRef: mUseRef,
    };
});

const TestComponent = (props): JSX.Element => {
    useAsyncInterval(props.callback, 2000);
    return <span />;
};

describe('useAsyncInterval', () => {
    it('should fire repeatedly after the interval', async () => {
        const mockedReact = React as jest.Mocked<typeof React>;
        const ref = { current: {} };
        mockedReact.useRef.mockReturnValue(ref);

        const syncCallback = jest.fn();
        const testComponent = mount(<TestComponent
            callback={syncCallback}
        />);

        // cb queued with interval but not called
        expect(syncCallback).not.toHaveBeenCalled();

        // wait for first tick
        await clock.tickAsync(2000);
        expect(syncCallback).toHaveBeenCalledTimes(1);

        // wait for second tick
        await clock.tickAsync(2000);
        expect(syncCallback).toHaveBeenCalledTimes(2);

        // wait for third tick
        await clock.tickAsync(2000);
        expect(syncCallback).toHaveBeenCalledTimes(3);
    });

    it('should wait for async callbacks to complete in between polling', async () => {
        const mockedReact = React as jest.Mocked<typeof React>;
        const ref = { current: {} };
        mockedReact.useRef.mockReturnValue(ref);

        const delayedCallback = jest.fn(() => (
            new Promise<void>((resolve) => {
                setTimeout(() => {
                    resolve();
                }, 2000);
            })
        ));
        const testComponent = mount(<TestComponent
            callback={delayedCallback}
        />);

        // cb queued with setInterval but not called
        expect(delayedCallback).not.toHaveBeenCalled();

        // Wait 2 seconds for first tick
        await clock.tickAsync(2000);
        // First cb called after 2 seconds
        expect(delayedCallback).toHaveBeenCalledTimes(1);
        // Wait for cb to resolve for 2 seconds
        await clock.tickAsync(2000);
        expect(delayedCallback).toHaveBeenCalledTimes(1);

        // Wait 2 seconds for second tick
        await clock.tickAsync(2000);
        expect(delayedCallback).toHaveBeenCalledTimes(2);
        // Wait for cb to resolve for 2 seconds
        await clock.tickAsync(2000);
        expect(delayedCallback).toHaveBeenCalledTimes(2);

        // Wait 2 seconds for third tick
        await clock.tickAsync(2000);
        expect(delayedCallback).toHaveBeenCalledTimes(3);
        // Wait for cb to resolve for 2 seconds
        await clock.tickAsync(2000);
        expect(delayedCallback).toHaveBeenCalledTimes(3);
    });
});
