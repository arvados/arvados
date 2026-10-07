// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

describe("SSH public key management", () => {
    let activeUser;

    before(function () {
        // Only set up common users once. These aren't set up as aliases because
        // aliases are cleaned up after every test. Also it doesn't make sense
        // to set the same users on beforeEach() over and over again, so we
        // separate a little from Cypress' 'Best Practices' here.
        cy.getUser("user", "Active", "User", false, true)
            .as("activeUser")
            .then(function () {
                activeUser = this.activeUser;
            });
    });

    function doAddKeyTest(keyName, keyContent) {
        cy.loginAs(activeUser);
        cy.goToPath("/ssh-keys-admin");

        cy.get("button[data-cy='add-new-ssh-key']").click();
        cy.get("input[name='publicKey']").fill(keyContent);
        cy.get("input[name='name']").fill(keyName);
        cy.get("button[data-cy='form-submit-btn']").click();
        cy.get("div[data-cy='snackbar'] div[role='alert'] span")
            .contains("Public key has been successfully created.")
            .should("be.visible");
        cy.get("main table td").contains(keyName).should("exist");
    }

    it("adds a new RSA key", () => {
        const keyName = `Test RSA key ${Math.floor(999999 * Math.random())} (8192 bits)`;
        const keyContent = `ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAEAQCpquYVt0yCFNC2lm/me65YGNKIDhzJgZbxAjJe7U8wu6rhq44Uz4nB1+Tr8dS1iG7wCBXSzJkMl9LjF2WyTi85OQ1/Y/0av3B4I2E52O1d9/xYDHU0IsmxJmh6t/2t5FPhzUQ7P2AjQA4lwgEksbxrAliBgPHJ+Ga1w4fNEqcDZydpSgNQM8EQ29WVOkK6ZVT/PPRVN4jUzMxRkOYzIsa7BcAiOkyZ0UDC3ZpEKcittyoLJFLRmcZk4+8eC4pNRZkMGIdSYfaXnC2kLHczzAHvIYe4z9JoIBCdGzz+w0VrhyYfBBV6hALFlsTalbQ4oVNc4ZPMMH9WC/wAa1vXyB85dENRlsUawRYTbGvgeo1inLe/7SwOzoJg/hDEMA7W2oEwOxA0R+h1RK1EFz0rPSA3iuUVPCPqU+kDkDzqUWhx/a7e7pb2/+DaCDsm4czElSv4cDc3VSfNdjF6DmVAON1nUNkF7jdJQaB0M6MFaj0++RhZEcfcttha5EcbeHwvZEojCAvUtnowMLgIdZFVngt+4t5DgBZVaa2s28nv4/JVe8kmv9l2xDMTuFC7FVgM5o/Ev7Ci5PwT52gqyU8Yi5U2JsK6D3zFzc8MvWeiCEE1Gue66fOzgHiSqRGlAk9n7TVlL9vn6dIvtCOmWqdlYdhhbf0RoxnN3Tn6xAebT7zWN8EcljjlF8zrKpJ7XSGRBrv147HtTUsVAx9cgV+wJbdTy9yxC3jkmKPubKkLwDkB2jlvqlyog/b+izZxmN1QzrdNIto9mEGma5zFWZ5wde8avevXIfrCR9gGaicujCr2TzT3R1ZqqmVA5tWE2SLwklzsql3NAOZlA7wLu5W8exED/FYfBbV+zTe9K2LkLwRrP4JusKfDiIjUpO+5cXBioxPXR28i7z0Jb/AUPB645IR6ZKu/IvmZpdTpY9VRoqLAUFYJ5uH+uaat4fWuhNiJdQmBxgtS4pByXi/HurmN+fJJfo+Zu9UsGS6NQfOYmMzeDeQBNCNQ64GPUrtAeBwgIk3Qx0r+tbQHwap/+xt6UJfoSB3QON2RLuvj6zDVn4ihc6PAuO0oaBcUnxs9nVqmToUiTBf9dYJxulrPvKXAiIgQMi+mR8LZGY+3EJVUnMyaE1RTHCRCmpMHorK0aQ27BUKjFWsSK7mXsMNgR6PrC5doX8gL1VlHwtEV76Z9/cpgSfohAQB+ekgqfhL1KwwwyGfpYapdgiaEcKaaynmNbIwwCm9qGgeYEL66MNQrmwTG0m/BvFff9u+8wruRMhWQyqLrjhNXDfdR3WhUGtKW/3qsReuJL91obx4C0AGUx0r92oGvgUZo/MDhbGAjQqrbKLeLMCNBg9HW3cbp+HxFUW7z ${keyName}`;
        doAddKeyTest(keyName, keyContent);
    });

    it("adds a new Ed25519 key", () => {
        const keyName = `Test Ed25519 key ${Math.floor(999999 * Math.random())} (8192 bits)`;
        const keyContent = `ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMDYMMyHskXXKhskNSXJcBGvBGEICaJR8ggbFklMpr2F ${keyName}`;
        doAddKeyTest(keyName, keyContent);
    });
});
