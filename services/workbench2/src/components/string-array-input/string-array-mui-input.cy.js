// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { StringArrayMuiInput } from './string-array-mui-input';

describe('StringArrayMuiInput Component', () => {
    beforeEach(() => {
        // Mount the component with required props
        cy.mount(
            <StringArrayMuiInput
                input={{
                    name: 'test',
                    value: [],
                    onChange: cy.stub().as('onChange'),
                    onBlur: cy.stub(),
                    onFocus: cy.stub(),
                }}
                meta={{
                    touched: false,
                    error: undefined,
                }}
                label='Test Input'
            />
        );
    });

    it('renders with initial empty state', () => {
        cy.get('input').should('exist');
        cy.get('input').should('have.value', '');
        cy.get('.MuiChip-root').should('not.exist');
    });

    it('calls onChange on Enter key press', () => {
        const testValue = 'test value';
        cy.get('input').type(`${testValue}{enter}`);
        cy.get('@onChange').should('have.been.calledWith', [testValue]);
    });

    it('calls onChange on Add button click', () => {
        const testValue = 'test value';
        cy.get('input').type(testValue);
        cy.get('button').click();
        cy.get('@onChange').should('have.been.calledWith', [testValue]);
    });

    it('trims whitespace from input values', () => {
        const testValue = '  test value  ';
        const trimmedValue = 'test value';

        cy.get('input').type(`${testValue}{enter}`);

        cy.get('@onChange').should('have.been.calledWith', [trimmedValue]);
    });

    it('clears input after adding value', () => {
        cy.get('input').type('test value{enter}');
        cy.get('input').should('have.value', '');
    });
});
