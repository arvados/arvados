// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { DatePicker } from './date-picker';
import moment from 'moment';

describe('DatePicker Component', () => {
    let defaultProps;

    beforeEach(() => {
        defaultProps = {
            label: 'Test Date',
            input: {
                value: '',
                onChange: cy.stub().as('onChange'),
            },
        };

        cy.mount(<DatePicker {...defaultProps} />);
    });

    it('renders with label', () => {
        cy.get('label').should('contain', 'Test Date');
    });

    it('initializes with current date when no minDate provided', () => {
        const today = moment().format('MM/DD/YYYY');
        cy.get('input').should('have.value', today);
    });

    it('initializes with minDate when provided', () => {
        const startValue = moment().add(1, 'year')

        cy.mount(
            <DatePicker
                {...defaultProps}
                startValue={startValue}
            />
        );

        cy.get('input').should('have.value', startValue.format('MM/DD/YYYY'));
    });

    it('opens calendar on click', () => {
        cy.get('button').click();
        // cypress doesn't find the calendar div, so check if the current month is displayed
        cy.contains(moment().format('MMMM'));
    });

    it('disables past dates when disablePast is true', () => {
        cy.mount(
            <DatePicker
                {...defaultProps}
                disablePast
            />
        );

        cy.get('button').click();

        const yesterday = moment().subtract(1, 'day').format('D');
        // MUI uses 'disabled="disabled"' instead of 'disabled={true}'
        cy.get('div[role="dialog"]').contains(yesterday).should('have.attr', 'disabled', 'disabled');
    });

    it('calls onChange when date is selected', () => {
        // Click any future date (first day of next month)
        const futureDate = moment().add(1, "month").startOf("month");
        const dayToSelect = futureDate.format('D');

        cy.get('button').click();
        cy.get('button[title="Next month"]').click();
        cy.get('div[role="dialog"]').contains(dayToSelect).first().click();

        // Verify onChange was called with the correct date
        cy.get('@onChange').should('have.been.called');
    });

    it('updates input value when date is selected', () => {
        cy.get('input').click();

        const futureDate = moment().add(1, "month").startOf("month");
        const dayToSelect = futureDate.startOf('day').valueOf();

        cy.get('button').click();
        cy.get('button[title="Next month"]').click();
        cy.get(`[data-timestamp="${dayToSelect}"]`).click();

        const expectedDate = futureDate.format('MM/DD/YYYY');
        cy.get('input').should('have.value', expectedDate);
    });

    it('handles keyboard navigation', () => {
        cy.get('button').click();

        // Navigate using arrow keys
        cy.get('div[role="dialog"]').should('exist').type('{rightarrow}').type('{enter}');

        // Verify a date was selected
        cy.get('@onChange').should('have.been.called');
    });

    it('initializes with startValue', () => {
        const expectedDate = moment().add(1, 'week').format('MM/DD/YYYY');

        cy.mount(
            <DatePicker
                label='Test Date'
                startValue={expectedDate}
                input={{
                    value: '',
                    onChange: cy.stub(),
                }}
            />
        );

        cy.get('input').should('have.value', expectedDate);
    });
});
