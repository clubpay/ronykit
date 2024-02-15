# RonyKIT Test Environment
Writing tests for RonyKIT is a bit tricky, as it requires a running instance of RonyKIT to be able to run the tests.
This is why we have a separate test environment for RonyKIT. We are using a separate go module to reduce the
dependencies of the main packages. In this way, we can keep the main package as lightweight as possible.

