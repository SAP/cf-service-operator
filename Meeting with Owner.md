Hi Colleagues,
Thxs for the the get to know each other meeting.
Sorry, if I did understand wrongly, but I would call you team “S4 SRE Team”.
We are the Steampunk Infrastructure Team.

Our summary:
-	S4 SRE Team is currently not using the CF-Service-Operator (using BTP-Service-Operator instead)
-	Helmchart and Docker Repos will remain separated
-	Steampunk Infra Team is responsible for new PPMS Component and ECCN onboarding (existing PPMS is deprecated) for Steampunk productive usage
-	Steampunk Infra Team will introduce first unit test and first integration test
-	S4 SRE Team will not invest in test coverage for now
-	Future option: Steampunk Infra Team may take ownership of the repo
- Contacts S4 SRE Team: Christoph Barbian (Architect), Tobias Graf (DM), Michael Leibel
- Contacts Steampunk Infra Team: Uwe Freidank (Architect), Bence Kiraly (PO), Santiago Ventura, Ralf Hammer

Question:
-	Can we have permission to create feature branches in your repo? 
That would make our development process more efficient.


# Meeting with Owner

## Team
Christoph Barbian (Chief Architect)
Tobias Graf (DM)
Michael Leibel (?)

## History
S4 replaced it with BTP-Service-Operator.

## Questions
- Should Steampunk developers be added to org SAP in github.com?
- Can we contribute with gingko, gomega and envtest?
- Can we merge Helmchart to Source Code Repo (see also https://github.com/SAP/sap-btp-service-operator/tree/main/sapbtp-operator-charts)?
- Compliant Build Pipelines (asked also in Slack)

## Test history
- Started also, but where
- envtest is prefered (alternative e2e tests)

## Docker and Helm chart
- Reason:
  - Concept of kyma module operator: extra module for cf-service-operator, which deploys cf-service-operator
  - kubebuilder restrict having 3 different components within 1 repo

## Open Source
- OSPO Team (inbound aproval process)
- PPMS was created by them
- ECCN is missing, but they would start to
- 