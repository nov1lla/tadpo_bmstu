Feature: Authentication with 2FA and password flows

  The system protects authentication with a second factor ("email" notification).
  A user can change password (planned) and recover after lockout.

  Scenario: Login with 2FA and planned password change
    Given a technical user is registered
    When I start login
    Then I receive a "login" code
    When I verify login with the code
    Then I am authenticated
    When I request a password change
    Then I receive a "password_change" code
    When I confirm the password change with the code
    Then login with the old password is rejected
    And login with the new password succeeds

  Scenario: Limited attempts lock the account
    Given a technical user is registered
    When I start login
    And I enter a wrong code 3 times
    Then my account becomes locked

  Scenario: Recovery after lockout
    Given a technical user is registered
    And my account is locked
    When I request account recovery
    Then I receive a "recovery" code
    When I confirm recovery with a new password
    Then login with the recovered password succeeds

