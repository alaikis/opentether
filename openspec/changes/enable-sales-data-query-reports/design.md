## Context

The required scenario has three roles: normal employee, administrator, and designated full-access group. Employees such as 林烽 must query only their own sales metrics. Administrators or members of designated groups must query all corresponding data. Reports must use actual datasource query results, not placeholder rows.

## Goals / Non-Goals

**Goals:**

- Provide deterministic helpers to decide whether a user has full sales-data access.
- Ensure self-scoped Text2SQL can filter by employee identity through semantic permissions.
- Execute report section SQL using the selected datasource connection.
- Support table and simple chart data sections for sales reports.
- Cover behavior with unit tests.

**Non-Goals:**

- Build a full ABAC policy center.
- Add LDAP/OIDC identity synchronization.
- Build a visual report designer.
- Implement arbitrary write operations against business systems.

## Decisions

Use the existing `UserContext` group data and role to derive full-data access. Administrators always have full access. Designated groups use conservative group code/name matching such as `sales_admin`, `sales_manager`, `data_admin`, and `admin`.

For report execution, use the existing external datasource pool infrastructure when a datasource ID is present. Table sections read `section.Query`; chart sections read `label_column` and `value_column` from query results.

## Risks / Trade-offs

- Group naming is a pragmatic MVP convention → future policy center should replace it.
- Report SQL is read-only validated and limited, but still depends on correct datasource permissions.
- Complex charts are out of scope; MVP supports label/value chart queries.
