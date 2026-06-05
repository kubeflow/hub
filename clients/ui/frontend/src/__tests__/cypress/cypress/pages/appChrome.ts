class AppChrome {
  visit() {
    cy.visit('/');
    this.waitForA11y();
  }

  waitForA11y() {
    cy.findByTestId('app-page-title').should('exist');
    cy.testA11y();
  }

  shouldBeUnauthorized() {
    cy.findByTestId('unauthorized-error');
    return this;
  }

  findNavToggle() {
    return cy.get('#page-nav-toggle');
  }

  findSideBar() {
    return cy.get('#page-sidebar');
  }

  findNavSection(name: string) {
    return this.findSideBar().findByRole('button', { name });
  }

  findNavItem(name: string, section?: string) {
    if (section) {
      this.findNavSection(section)
        // do not fail if the section is not found
        .should('have.length.at.least', 0)
        .then(($el) => {
          if ($el.attr('aria-expanded') === 'false') {
            cy.wrap($el).click();
          }
        });
    }
    return this.findSideBar().findByRole('link', { name });
  }
}

export const appChrome = new AppChrome();
