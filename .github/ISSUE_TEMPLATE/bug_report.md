---
name: Bug report
about: File a report about a problem with the Operator
title: ''
labels: ''
assignees: ''

---
**What did you do to encounter the bug?**
Steps to reproduce the behavior:
1. Go to '...'
2. Click on '....'
3. Scroll down to '....'
4. See error

**What did you expect?**
A clear and concise description of what you expected to happen.

**What happened instead?**
A clear and concise description of what happened instead

**Screenshots**
If applicable, add screenshots to help explain your problem.

**Operator Information**
 - Operator Version
 - Base Images used (ubuntu or UBI)

**Ops Manager Information**
 - Ops Manager Version
 - Is Ops Manager managed by the Operator or not?
 - Is Ops Manager in Local Mode?

**Kubernetes Cluster Information**
 - Distribution:
 - Version:
 - Image Registry location (quay, or an internal registry)

**Additional context**
Add any other context about the problem here.

If possible, please include:
 - `kubectl describe` output
 - yaml definitions for your objects
 - log files for the operator, database pods and Ops Manager
 - An [Ops Manager Diagnostic Archive](https://docs.opsmanager.mongodb.com/current/tutorial/retrieve-debug-diagnostics)
