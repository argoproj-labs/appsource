This file details the next features being looked into before the Alpha release of the AppSource Controller.

# Holding Multiple Project Specs in AppSource ConfigMap

Introduced in [https://github.com/aceamarco/argocd-app-source/issues/3][Issue #3]

We would like to give users the ability to use various Project templates so that 
they're not tied down to single project when creating an application.

# Testing Framework

Introduced in [https://github.com/aceamarco/argocd-app-source/issues/4](bin/controller-genIssue #4)

We would like to provide future contributors a way to test their contributions against
expected and incorrect test cases. Currently contributions are tested by stepping through 
expected resources defined in `manifests/samples`

# Post Alpha-Version

## Notifications Engine Integration with AppSource

Introduced in [https://github.com/aceamarco/argocd-app-source/issues/5][Issue #5]

This feature would be nice to have after reaching MVP with the previous two goals 
mentioned above. With this feature, users would be able to receive notifications about 
the status of their applications created by the AppSource controller.

