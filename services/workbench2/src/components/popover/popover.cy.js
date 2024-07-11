// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { Popover } from "./popover";
import Button from "@mui/material/Button";

describe("<Popover />", () => {
    it("opens on default trigger click", () => {
        cy.mount(<Popover />);
        cy.get('[data-cy=popover]').should('not.exist');
        cy.get('button').click();
        cy.get('[data-cy=popover]').should('exist');
    });

    it("renders custom trigger", () => {
        cy.mount(<Popover triggerComponent={CustomTrigger} />);
        cy.get('button').should('have.text', 'Open popover');
    });

    it("opens on custom trigger click", () => {
        cy.mount(<Popover triggerComponent={CustomTrigger} />);
        cy.get('[data-cy=popover]').should('not.exist');
        cy.get('button').should('have.text', 'Open popover').click();
        cy.get('[data-cy=popover]').should('exist');
    });

    it("renders children when opened", () => {
        cy.mount(
            <Popover>
                <CustomTrigger />
            </Popover>
        );
        cy.get('button').click();
        cy.get('button').contains('Open popover').should('have.length', 1);
    });

    it("does not close if closeOnContentClick is not set", () => {
        cy.mount(
            <Popover>
                <CustomTrigger />
            </Popover>
        );
        cy.get('button').click();
        cy.get('button').should('have.text', 'Open popover');
        cy.contains('Open popover').click();
        cy.get('[data-cy=popover]').should('exist');
    });

    it("closes on content click if closeOnContentClick is set", () => {
        cy.mount(
            <Popover closeOnContentClick>
                <CustomTrigger />
            </Popover>
        );
        cy.get('button').click();
        cy.get('[data-cy=popover]').should('exist');
        cy.contains('Open popover').click();
        cy.get('[data-cy=popover]').should('not.exist');
    });
});

const CustomTrigger = (props) => (
    <Button {...props}>
        Open popover
    </Button>
);
