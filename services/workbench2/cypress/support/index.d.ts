/**
  * This command tries to ensure that the elements in the DOM are actually visible
  * and done (re)rendering. This is due to how React re-renders components.
  *
  * IMPORTANT NOTES:
  *    => You should only use this command in instances where a test is failing due
  *       to detached elements. Cypress will probably give you a warning along the lines
  *       of, "Element has an effective width/height of 0". This warning is not very useful
  *       in pointing out it is due to the element being detached from the DOM AFTER the
  *       cy.get command had already retrieved it. This command can save you from that
  *       by explicitly waiting for the DOM to stop changing.
  *    => This command can take anywhere from 100ms to 5 seconds to complete
  *    => This command will exit early (500ms) when no changes are occurring in the DOM.
  *       We wait a minimum of 500ms because sometimes it can take up to around that time
  *       for mutations to start occurring.
  *
  * GitHub Issues:
  *    * https://github.com/cypress-io/cypress/issues/695 (Closed - no activity)
  *    * https://github.com/cypress-io/cypress/issues/7306 (Open - re-get detached elements)
  *
  * @example Wait for the DOM to stop changing before retrieving an element
  * cy.waitForDom().get('#an-elements-id')
  */
 waitForDom(): Chainable<any>
