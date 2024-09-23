// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useState, useEffect } from 'react';
import { useAsyncInterval } from './use-async-interval';

const TestComponent = ({callback}) => {
  useAsyncInterval(callback, 1000);
  return <div>test</div>;
};

describe('useAsyncInterval', () => {
  it('should fire repeatedly after the interval', () => {
    cy.clock();
    const syncCallback = cy.spy().as('syncCallback');
    cy.mount(<TestComponent callback={syncCallback} />);

    cy.get('@syncCallback').should('not.have.been.called');

    cy.tick(1000);
    cy.wait(0);
    
    cy.get('@syncCallback').should('have.been.calledOnce');
    
    cy.tick(1000);
    cy.wait(0);

    cy.get('@syncCallback').should('have.been.calledTwice');

    cy.tick(1000);
    cy.wait(0);

    cy.get('@syncCallback').should('have.been.calledThrice');
    cy.clock().invoke('restore');
  });

    it('should wait for async callbacks to complete in between polling', async () => {
        cy.clock();

        const delayedCallback = cy.stub().callsFake(() => {
            return new Promise((resolve) => {
              setTimeout(() => {
                resolve('done');
              }, 2000);
            });
          }).as('delayedCallback');

        cy.mount(<TestComponent
            callback={delayedCallback}
        />);

        // cb queued with setInterval but not called
        cy.get('@delayedCallback').should('not.have.been.called');

        // Wait 2 seconds for first tick
        cy.tick(2000);
        cy.wait(0);

        // First cb called after 2 seconds
        cy.get('@delayedCallback').should('have.been.calledOnce');

        // Wait for cb to resolve for 2 seconds
        cy.tick(2000);
        cy.wait(0);
        cy.get('@delayedCallback').should('have.been.calledOnce');

        // Wait 2 seconds for second tick
        cy.tick(2000);
        cy.wait(0);
        cy.get('@delayedCallback').should('have.been.calledTwice');

        // Wait for cb to resolve for 2 seconds
        cy.tick(2000);
        cy.wait(0);
        cy.get('@delayedCallback').should('have.been.calledTwice');

        // Wait 2 seconds for third tick
        cy.tick(2000);
        cy.wait(0);
        cy.get('@delayedCallback').should('have.been.calledThrice');

        // Wait for cb to resolve for 2 seconds
        cy.tick(2000);
        cy.wait(0);
        cy.get('@delayedCallback').should('have.been.calledThrice');
    });
});
