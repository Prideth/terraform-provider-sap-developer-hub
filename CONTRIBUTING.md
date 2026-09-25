# Contributing

## Ground rules

1. **Every resource and data source must trace to an officially documented
   public API.** Never a private/UI-only endpoint discovered by browser
   inspection or reverse engineering. If SAP has not documented a public
   API for something, it does not belong in this provider - see
   [`DESIGN.md`](DESIGN.md) §12 for the current list of things researched
   and deliberately left out for exactly this reason, and add to it rather
   than working around it.
2. **Respect the scope boundary.** This provider manages Developer Hub
   catalog/consumer-lifecycle objects only. BTP provisioning belongs to the
   `SAP/btp` provider; capability activation and product/API authoring
   belong to `terraform-provider-integration-suite`. See `DESIGN.md` §2.
3. **Quality over quantity.** A resource ships only once every item in
   [Definition of done](#definition-of-done) below is met - not as a
   follow-up.

## Branch model

- `master` is the stable/release branch, fast-forwarded only on explicit
  release.
- `dev` is the permanent integration branch.
- All other work happens on `feature/<name>` branches, cut from `dev` and
  merged back into `dev`, where `<name>` describes the change
  (`feature/domain-categories`, `feature/governance-settings`, ...). No
  other branch names or prefixes are used.

## Development setup

```shell
go build ./...
go test ./...
```

See [`README.md`](README.md#development) for the full `make` targets.

## Adding or changing a resource

1. Check the **newest** version of the API first, in the order `DESIGN.md`
   §20 lays out: the live tenant's `$metadata`, then the `api.sap.com`
   OpenAPI spec for the relevant `DevPortal_*` artifact, then
   `help.sap.com`. Add the research to `DESIGN.md` §5 with a citation to
   the exact source and version, and update the §20 status table. If you
   cannot find a documented API, stop - see Ground rule 1.
2. Add or extend the domain client in `internal/client/developerhub`, with
   unit tests against a local `httptest` server covering the happy path,
   not-found, and at least one conflict/validation error. Add every new
   OData entity set, property and navigation property to
   `developerhub.Contract` (`TestContract_CoversEveryWireField` fails if
   you forget), then run `make verify-api` against a tenant.
3. Add the resource/data source under `internal/provider`, following the
   existing files' conventions (schema description referencing `DESIGN.md`,
   `RequiresReplace` only where SAP genuinely requires it, not-found
   handling on Read, import support).
4. Add an example under `examples/`.
5. Regenerate docs: `make docs`.
6. Update `FEATURE_MATRIX.md`.

## Definition of done

- Schema, with descriptions referencing the relevant `DESIGN.md` section.
- Create/Read/Delete, and Update if and only if SAP documents an update
  operation (otherwise the fields must be `RequiresReplace`, with a comment
  explaining why).
- Import support, unless the object genuinely cannot be looked up by a
  stable id (document why in the resource's schema description if so).
- Unit tests for the domain client and, where there is non-trivial mapping
  logic (like attribute reconciliation), for the resource itself.
- A second `terraform apply` after a successful `apply` must produce an
  empty plan.
- Sensitive fields marked `Sensitive: true`, never logged.

## Commit and PR expectations

- Commit messages are plain, descriptive English sentences explaining what
  changed and why - not Conventional Commits prefixes.
- Run `gofmt`, `go vet`, the relevant tests, and `make docs` before every
  commit; a commit should represent a working state.

## Code of Conduct

Be respectful and constructive. This is a community project with no formal
code of conduct document yet; use good judgment.
