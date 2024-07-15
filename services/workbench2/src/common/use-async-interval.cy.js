// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useState, useEffect } from 'react';
import { useAsyncInterval } from './use-async-interval';

// import { configure, mount } from 'enzyme';
// import Adapter from 'enzyme-adapter-react-16';
// import FakeTimers from "@sinonjs/fake-timers";

// configure({ adapter: new Adapter() });
// const clock = FakeTimers.install();

// jest.mock('react', () => {
//     const originalReact = jest.requireActual('react');
//     const mUseRef = jest.fn();
//     return {
//         ...originalReact,
//         useRef: mUseRef,
//     };
// });

// const TestComponent = (props) => {
//     useAsyncInterval(props.callback, 2000);
//     return <span />;
// };

const TestComponent = ({asyncCallback}) => {

  useAsyncInterval(asyncCallback, 1000);

  return <div>test</div>;
};

describe('useAsyncInterval', () => {
  it('should fire repeatedly after the interval', () => {
    cy.clock();
    const asyncCallback = cy.spy().as('asyncCallback');
    cy.mount(<TestComponent asyncCallback={asyncCallback} />);

    cy.get('@asyncCallback').should('not.have.been.called');

    cy.tick(1000);
    cy.wait(0);
    
    cy.get('@asyncCallback').should('have.been.calledOnce');
    
    cy.tick(1000);
    cy.wait(0);

    cy.get('@asyncCallback').should('have.been.calledTwice');

    cy.tick(1000);
    cy.wait(0);

    cy.get('@asyncCallback').should('have.been.calledThrice');
    cy.clock().invoke('restore');
  });

    it('should wait for async callbacks to complete in between polling', async () => {
        // const mockedReact = React as jest.Mocked<typeof React>;
        // const ref = { current: {} };
        // mockedReact.useRef.mockReturnValue(ref);

        const delayedCallback = jest.fn(() => (
            new Promise((resolve) => {
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
